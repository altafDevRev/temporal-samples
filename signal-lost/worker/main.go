package main

import (
	"log"

	signallost "github.com/temporalio/samples-go/signal-lost"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	// The client and worker are heavyweight objects that should be created once per process.
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	w := worker.New(c, "signal-lost", worker.Options{})

	// Register the workflow
	w.RegisterWorkflow(signallost.WorkflowDefinition)

	// Register the activity
	w.RegisterActivity(signallost.ActivityOneDefinition)
	w.RegisterActivity(signallost.ActivityTwoDefinition)
	w.RegisterActivity(signallost.ActivityThreeDefinition)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}
