package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-zoo/bone"
	"github.com/gofrs/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	// Create the client object just once per process
	c, err := client.NewClient(client.Options{})
	if err != nil {
		log.Fatalln("unable to create Temporal client", err)
	}
	defer c.Close()

	// run our web server. this isn't clean for startup or shutdown but
	// that's ok for now.
	go func() {
		mux := bone.New()

		// API to launch an workflow. could be an incoming webhook,
		// just for example.
		mux.Post("/pr", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			var details CheckDetails
			dec := json.NewDecoder(req.Body)

			if err := dec.Decode(&details); err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				rw.Write([]byte("bad"))
				return
			}

			options := client.StartWorkflowOptions{
				// This id would be the repo + pr, and possibly new sha.
				ID:        "pr-check-workflow+" + uuid.Must(uuid.NewV4()).String(),
				TaskQueue: PRCheckTaskQueue,
			}

			_, err := c.ExecuteWorkflow(context.Background(), options, CheckPR, details)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				rw.Write([]byte("couldn't enqueue"))
				return
			}

			// TODO: help get status
		}))

		// API to complete an activity.
		mux.Post("/callback/:tok", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			token, _ := base64.StdEncoding.DecodeString(bone.GetValue(req, "tok"))
			// TODO: done here could be request body
			err = c.CompleteActivity(context.TODO(), []byte(token), "done", nil)
			fmt.Println(err)
		}))

		err := http.ListenAndServe("0.0.0.0:6007", mux)
		fmt.Println(err)
	}()

	// This worker hosts both Worker and Activity functions
	w := worker.New(c, PRCheckTaskQueue, worker.Options{})
	w.RegisterWorkflow(CheckPR)
	w.RegisterActivity(Test)
	w.RegisterActivity(SetCommitStatus)
	// Start listening to the Task Queue
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("unable to start Worker", err)
	}
}
