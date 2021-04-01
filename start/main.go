package main

import (
	"context"
	"log"

	"github.com/gofrs/uuid"
	"go.temporal.io/sdk/client"

	"fun-with-temporal/app"
)

func main() {
	// Create the client object just once per process
	c, err := client.NewClient(client.Options{})
	if err != nil {
		log.Fatalln("unable to create Temporal client", err)
	}
	defer c.Close()
	options := client.StartWorkflowOptions{
		// This id would be the repo + pr, and possibly new sha.
		ID:        "pr-check-workflow+" + uuid.Must(uuid.NewV4()).String(),
		TaskQueue: app.PRCheckTaskQueue,
	}
	prDetails := app.CheckDetails{
		Repo: "jbowes/repl",
		PR:   "123",
		Old:  "old-sha",
		New:  "new-sha",
	}
	we, err := c.ExecuteWorkflow(context.Background(), options, app.CheckPR, prDetails)
	if err != nil {
		log.Fatalln("error starting PR Check workflow", err)
	}
	printResults(we.GetID(), we.GetRunID())
	// wait for finish
	we.Get(context.TODO(), nil)
	log.Printf("all done!")
}

func printResults(workflowID, runID string) {
	log.Printf("started done %s - %s", workflowID, runID)
}
