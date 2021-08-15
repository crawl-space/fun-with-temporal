package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-zoo/bone"
	"github.com/gofrs/uuid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc/codes"
)

func Api(c client.Client) http.Handler {
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
		ctx := req.Context()

		id := bone.GetValue(req, "id")

		rw.Header().Add("Content-Type", "application/json")

		// Does the requested workflow execution exist at all?
		desc, err := c.DescribeWorkflowExecution(ctx, id, "")
		switch {
		case err == nil:
		case serviceerror.ToStatus(err).Code() == codes.NotFound:
			rw.WriteHeader(http.StatusNotFound)
			fmt.Fprint(rw, "{}")
			return
		default:
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(rw, "{}")
			return
		}

		// You could check the execution "memo" here for
		// thinks like an org id for AuthZ.

		var out struct {
			Status string `json:"status"`
		}

		// Figure out what our status is.
		// As soon as the work is added to the queue, its status
		// will be running (so "pending" won't work here).
		switch desc.WorkflowExecutionInfo.Status {
		case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
			out.Status = "completed"
		case enums.WORKFLOW_EXECUTION_STATUS_RUNNING, enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW:
			out.Status = "running"
		case enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
			out.Status = "timed_out"
		default: // cancelled, errored, etc
			out.Status = "errored"
		}

		rw.WriteHeader(http.StatusOK)
		enc := json.NewEncoder(rw)
		enc.Encode(out)
	}))

	// API to complete an activity.
	mux.Post("/callback/:tok", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		token, _ := base64.StdEncoding.DecodeString(bone.GetValue(req, "tok"))
		// TODO: done here could be request body
		err := c.CompleteActivity(context.TODO(), []byte(token), "done", nil)
		fmt.Println(err)
	}))

	return mux
}
