package main

import (
	"context"
	"log"
	"time"

	"github.com/pborman/uuid"
	"go.temporal.io/sdk/client"

	signallost "github.com/temporalio/samples-go/signal-lost"
)

func main() {
	// The client is a heavyweight object that should be created once per process.
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	workflowOptions := client.StartWorkflowOptions{
		ID:        "signal-lost_" + uuid.New(),
		TaskQueue: "signal-lost",
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, signallost.WorkflowDefinition)
	if err != nil {
		log.Fatalln("Unable to execute workflow", err)
	}
	log.Println("Started workflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID())

	// After 8 seconds, send a callback signal to the workflow
	// to simulate a signal lost scenario.

	// Wait for 8 seconds
	<-time.After(8 * time.Second)

	// Send a signal to the workflow
	err = c.SignalWorkflow(context.Background(), we.GetID(), we.GetRunID(), signallost.CallbackSignalName, "callback-signal-input")
	if err != nil {
		log.Fatalln("Unable to signal workflow", err)
	}

	// After 20 seconds, send an exit signal to the workflow

	// Wait for 20 seconds
	<-time.After(20 * time.Second)

	// Send a signal to the workflow
	err = c.SignalWorkflow(context.Background(), we.GetID(), we.GetRunID(), signallost.ExitSignalName, "exit-signal-input")
	if err != nil {
		log.Fatalln("Unable to signal workflow", err)
	}
}
