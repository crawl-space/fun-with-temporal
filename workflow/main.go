package main

import (
	"log"
	"net/http"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	// Create the client object just once per process
	c, err := client.NewClient(client.Options{
		HostPort: "temporal:7233",
	})
	if err != nil {
		log.Fatalln("unable to create Temporal client", err)
	}
	defer c.Close()

	// run our web server. this isn't clean for startup or shutdown but
	// that's ok for now.
	go func() {
		err := http.ListenAndServe("0.0.0.0:6007", Api(c))
		log.Fatalln(err)
	}()

	time.Sleep(2 * time.Minute)

	// This worker hosts both Worker and Activity functions
	w := worker.New(c, PRCheckTaskQueue, worker.Options{})

	w.RegisterWorkflow(CheckPR)

	w.RegisterActivity(Test)
	w.RegisterActivity(DiffResults)
	w.RegisterActivity(SetCommitStatus)
	// Start listening to the Task Queue
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("unable to start Worker", err)
	}
}
