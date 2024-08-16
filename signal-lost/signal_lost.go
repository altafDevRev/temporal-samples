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

	shouldAddDefault := true
	for {
		// Wait for the next signal or timeout.
		selector := workflow.NewSelector(ctx)
		exit := false

		var callbackHandlerFunc = func(c workflow.ReceiveChannel, more bool) {
			var signalData string
			c.Receive(ctx, &signalData)
			logger.Info("Received signal.", "SignalData", signalData)

			// Starting ActivityTwo
			err := workflow.ExecuteActivity(ctx, CallbackActivity, &CallbackInputParams{}).Get(ctx, nil)
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

		if shouldAddDefault {
			selector.AddDefault(func() {
				logger.Info("Starting default activity.")

				// Execute DefaultActivity
				var DefaultActivityOutput *DefaultActivityOutputParams
				err := workflow.ExecuteActivity(ctx, DefaultActivity, &DefaultActivityInputParams{}).Get(ctx, &DefaultActivityOutput)
				if err != nil {
					logger.Error("DefaultActivity failed.", "Error", err)
				}

				if DefaultActivityOutput.shouldExit {
					logger.Info("Exiting the workflow.")
					exit = true
				}

				shouldAddDefault = false
			})
		}

		selector.Select(ctx)

		logger.Info("Selector.HasPending: " + strconv.FormatBool(selector.HasPending()))
		logger.Info("At the end of loop, shouldAddDefault: " + strconv.FormatBool(shouldAddDefault))
		if exit {
			break
		}
	}

	logger.Info("Workflow completed.")
	return nil
}

// ======================================================

type CallbackInputParams struct {
}

type CallbackOutputParams struct {
}

// Second activity that runs for 5 seconds precisely
func CallbackActivity(ctx context.Context, params *CallbackInputParams) (*CallbackOutputParams, error) {
	logger := activity.GetLogger(ctx)

	logger.Info("CallbackActivity starting")

	// Sleep for 5 seconds
	time.Sleep(5 * time.Second)

	logger.Info("CallbackActivity finished")

	return nil, nil
}

// ======================================================

type DefaultActivityInputParams struct {
}

type DefaultActivityOutputParams struct {
	shouldExit bool
}

// Third activity that runs for 10 seconds precisely
func DefaultActivity(ctx context.Context, params *DefaultActivityInputParams) (*DefaultActivityOutputParams, error) {
	logger := activity.GetLogger(ctx)

	logger.Info("DefaultActivity starting")

	// Sleep for 10 seconds
	time.Sleep(10 * time.Second)

	logger.Info("DefaultActivity finished")

	return &DefaultActivityOutputParams{
		shouldExit: false,
	}, nil
}

// ======================================================
