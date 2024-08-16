package signallost

import (
	"context"
	"strconv"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	CallbackSignalName = "callback"
	ExitSignalName     = "exit"
)

func WorkflowDefinition(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)

	ao := workflow.ActivityOptions{
		// Execution of a step should not take more than 30 seconds.
		StartToCloseTimeout: 30 * time.Second,

		// 3 total attempts. Initial attempt, first retry after 30 seconds, second retry after 1 minute.
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    30 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,

			// Non-retryable error types.
			NonRetryableErrorTypes: []string{
				"InvalidArgumentError",
				"AlreadyExistsError",
				"UnauthorizedError",
				"UnimplementedError",
				"NotFoundError",
			},
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	logger.Info("Starting the workflow.")

	callbackSignalChannel := workflow.GetSignalChannel(ctx, CallbackSignalName)
	exitSignalChannel := workflow.GetSignalChannel(ctx, ExitSignalName)

	eventsCounter := 0

	eventsChannel := workflow.NewBufferedChannel(ctx, 100)

	// Add first event to the events channel.
	eventsChannel.Send(ctx, "eventOne")

	areEventsProcessed := false
	for {
		// Wait for the next signal or timeout.
		selector := workflow.NewSelector(ctx)
		exit := false

		var eventHandlerFunc = func(c workflow.ReceiveChannel, more bool) {
			var event string
			c.Receive(ctx, &event)
			// Start ActivityOne
			err := workflow.ExecuteActivity(ctx, ActivityOneDefinition, &ActivityOneInputParams{
				EventInfo: event,
			}).Get(ctx, nil)
			if err != nil {
				logger.Error("ActivityOne failed.", "Error", err)
			}
			eventsCounter++
			areEventsProcessed = true

			// If eventsCounter is less than 2, send the next event to the events channel.
			if eventsCounter < 2 {
				eventString := "event" + strconv.Itoa(eventsCounter+1)
				eventsChannel.Send(ctx, eventString)
			}
		}
		selector.AddReceive(eventsChannel, eventHandlerFunc)

		var callbackHandlerFunc = func(c workflow.ReceiveChannel, more bool) {
			var signalData string
			c.Receive(ctx, &signalData)
			logger.Info("Received signal.", "SignalData", signalData)

			// Starting ActivityTwo
			err := workflow.ExecuteActivity(ctx, ActivityTwoDefinition, &ActivityTwoInputParams{}).Get(ctx, nil)
			if err != nil {
				logger.Error("ActivityTwo failed.", "Error", err)
			}
		}
		selector.AddReceive(callbackSignalChannel, callbackHandlerFunc)
		selector.AddReceive(exitSignalChannel, func(c workflow.ReceiveChannel, more bool) {
			var signalData string
			c.Receive(ctx, &signalData)
			logger.Info("Received exit signal.", "SignalData", signalData)
			exit = true
		})

		if areEventsProcessed {
			selector.AddDefault(func() {
				logger.Info("No more events to process.")

				// Execute ActivityThree
				var activityThreeOutput *ActivityThreeOutputParams
				err := workflow.ExecuteActivity(ctx, ActivityThreeDefinition, &ActivityThreeInputParams{}).Get(ctx, &activityThreeOutput)
				if err != nil {
					logger.Error("ActivityThree failed.", "Error", err)
				}

				if activityThreeOutput.shouldExit {
					logger.Info("Exiting the workflow.")
					exit = true
				}

				areEventsProcessed = false
			})
		}

		selector.Select(ctx)

		logger.Info("Selector.HasPending: " + strconv.FormatBool(selector.HasPending()))
		logger.Info("At the end of loop, areEventsProcessed: " + strconv.FormatBool(areEventsProcessed))
		if exit {
			break
		}
	}

	logger.Info("Workflow completed.")
	return nil
}

// ======================================================

type ActivityOneInputParams struct {
	EventInfo string
}

type ActivityOneOutputParams struct {
}

// First activity that runs for 5 seconds precisely
func ActivityOneDefinition(ctx context.Context, params *ActivityOneInputParams) (*ActivityOneOutputParams, error) {
	logger := activity.GetLogger(ctx)

	logger.Info("ActivityOneDefinition starting, EventInfo: " + params.EventInfo)

	// Sleep for 2 seconds
	time.Sleep(2 * time.Second)

	logger.Info("ActivityOneDefinition finished")

	return nil, nil
}

// ======================================================

type ActivityTwoInputParams struct {
}

type ActivityTwoOutputParams struct {
}

// Second activity that runs for 5 seconds precisely
func ActivityTwoDefinition(ctx context.Context, params *ActivityTwoInputParams) (*ActivityTwoOutputParams, error) {
	logger := activity.GetLogger(ctx)

	logger.Info("ActivityTwoDefinition starting")

	// Sleep for 5 seconds
	time.Sleep(5 * time.Second)

	logger.Info("ActivityTwoDefinition finished")

	return nil, nil
}

// ======================================================

type ActivityThreeInputParams struct {
}

type ActivityThreeOutputParams struct {
	shouldExit bool
}

// Third activity that runs for 10 seconds precisely
func ActivityThreeDefinition(ctx context.Context, params *ActivityThreeInputParams) (*ActivityThreeOutputParams, error) {
	logger := activity.GetLogger(ctx)

	logger.Info("ActivityThree starting")

	// Sleep for 10 seconds
	time.Sleep(10 * time.Second)

	logger.Info("ActivityThree finished")

	return &ActivityThreeOutputParams{
		shouldExit: false,
	}, nil
}

// ======================================================
