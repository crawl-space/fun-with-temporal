package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/go-zoo/bone"
	"github.com/gofrs/uuid"
	"go.temporal.io/api/enums/v1"
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
				ID:        "pr-check-workflow-" + uuid.Must(uuid.NewV4()).String(),
				TaskQueue: PRCheckTaskQueue,
			}

			work, err := c.ExecuteWorkflow(context.Background(), options, CheckPR, details)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				rw.Write([]byte("couldn't enqueue"))
				return
			}

			// Mark the job as accepted and point at another URL for status polling
			// TODO: use a custom id (uuid) as a search attribute
			rw.Header().Add("Location", "/jobs/"+url.PathEscape(work.GetID()))
			rw.WriteHeader(http.StatusAccepted)
			rw.Write([]byte("accepted"))
		}))

		// Get the status of a given job
		mux.Get("/jobs/:id", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			id := bone.GetValue(req, "id")
			work := c.GetWorkflow(context.TODO(), id, "")

			rw.Header().Add("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)

			desc, err := c.DescribeWorkflowExecution(context.TODO(), id, "")
			if err != nil {
				fmt.Println(err)
				return
			}

			var s map[string]interface{}

			// A query for status will use the workflow query handler to return the status value, as it is
			// in a in-flight workflow.
			// This works for finished workflows, too, provided the worker for them still exists
			// But doing this runs the worker replaying the data, and seems wrong for asking for something
			// from 90 days ago. So we return the same value as the result, by convention.
			// Thus if the job is done, we use that value.
			//
			// XXX: I haven't yet decided if query + result is better than doing a side-channel thing
			// in a distinct DB. I think it is.

			// XXX how do you determine if the work was started but not yet run anywhere?

			if desc.WorkflowExecutionInfo.Status == enums.WORKFLOW_EXECUTION_STATUS_COMPLETED {
				err := work.Get(context.TODO(), &s)
				if err != nil {
					fmt.Println(err)
					return
				}
			} else {

				// XXX: this will time out if the work isn't and can't be scheduled. not great.
				resp, err := c.QueryWorkflow(context.TODO(), work.GetID(), work.GetRunID(), "status")
				if err != nil {
					fmt.Println(err)
					return
				}

				resp.Get(&s)
			}

			var out struct {
				Status string `json:"status"`
			}

			out.Status = desc.WorkflowExecutionInfo.Status.String()

			enc := json.NewEncoder(rw)
			enc.Encode(s)
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
	w.RegisterActivity(DiffResults)
	w.RegisterActivity(SetCommitStatus)
	// Start listening to the Task Queue
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("unable to start Worker", err)
	}
}
