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
	"go.temporal.io/sdk/client"
)

func Api(c client.Client, r *Repository) http.Handler {
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

		work, err := c.ExecuteWorkflow(context.Background(), options, (&CheckPR{}).CheckPR, details)
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

		rw.Header().Add("Content-Type", "application/json")
		enc := json.NewEncoder(rw)

		var s PRStatus
		err := r.Get(req.Context(), id, &s)
		switch err {
		case nil:
			rw.WriteHeader(http.StatusOK)
			enc.Encode(&s)
		case ErrNotFound:
			// TODO: ask temporal here, in case it is PENDING.
			rw.WriteHeader(http.StatusNotFound)
			enc.Encode("not found")
		default:
			rw.WriteHeader(http.StatusInternalServerError)
			enc.Encode("error")
			fmt.Println(err)
		}
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
