package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

const PRCheckTaskQueue = "PR_CHECK_TASK_QUEUE"

type CheckDetails struct {
	Repo string
	PR   string
	Old  string
	New  string
}

type ResponseStatus struct {
	State       string    `json:"state"` // should be enum
	TimeStamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
}

type Response struct {
	Status []ResponseStatus `json:"status"`
}

func CheckPR(ctx workflow.Context, details CheckDetails) (*Response, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute, // Max 1 minute before quitting
	})

	// our result object, which we update as we go, so we can respond to queries
	var resp Response
	workflow.SetQueryHandler(ctx, "status", func() (*Response, error) {
		return &resp, nil
	})

	// Start both tests. We get back futures. They transparently handle
	// retries and persisting the results.
	resp.Status = append([]ResponseStatus{{State: "testing", TimeStamp: workflow.Now(ctx), Description: "tests are running"}}, resp.Status...)
	old := workflow.ExecuteActivity(ctx, Test, details.Repo, details.Old)
	new := workflow.ExecuteActivity(ctx, Test, details.Repo, details.New)

	// we need both results, so its fine to wait to resolve in either order.
	var oldRes, newRes string

	if err := old.Get(ctx, &oldRes); err != nil {
		return nil, err
	}

	if err := new.Get(ctx, &newRes); err != nil {
		return nil, err
	}

	// fast to do and deterministic. run inside the workflow.
	var diff string
	resp.Status = append([]ResponseStatus{{State: "diffing", TimeStamp: workflow.Now(ctx), Description: "test results are diffing"}}, resp.Status...)
	err := workflow.ExecuteActivity(ctx, DiffResults, oldRes, newRes).Get(ctx, &diff)
	if err != nil {
		return nil, err
	}

	// Resolve the final task and finish.
	resp.Status = append([]ResponseStatus{{State: "reporting", TimeStamp: workflow.Now(ctx), Description: "PR check results are being posted"}}, resp.Status...)
	if err := workflow.ExecuteActivity(ctx, SetCommitStatus, details.Repo, details.PR, diff).Get(ctx, nil); err != nil {
		return nil, err
	}

	// XXX: Does setting this final status make sense? how will query interact with the actual workflow completion status?
	// is there a race condition? probably doesn't matter.
	resp.Status = append([]ResponseStatus{{State: "complete", TimeStamp: workflow.Now(ctx), Description: "All done"}}, resp.Status...)
	return &resp, nil
}

func Test(ctx context.Context, repo, sha string) (string, error) {
	// fully asnyk ;)
	token := base64.StdEncoding.EncodeToString(activity.GetInfo(ctx).TaskToken)
	fmt.Println("task token is", token)

	var dat = struct {
		Callback string `json:"callback"`
	}{Callback: "http://workflow:6007/callback/" + token}
	body, _ := json.Marshal(dat)

	// Send the request to the leaf work processor
	resp, err := http.DefaultClient.Post("http://leaf:6008/", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close() // just close it really. we don't use it.

	// magic return. pass the zero value of your result, and a sentinel error.
	return "", activity.ErrResultPending
}

func DiffResults(old, new string) (string, error) {
	var dat = struct {
		Old string `json:"old"`
		New string `json:"new"`
	}{Old: old, New: new}
	body, _ := json.Marshal(dat)

	resp, err := http.DefaultClient.Post("http://leaf:6008/diff", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// for more fun read the response body
	return "new bad code added. :( :( :(", nil
}

func SetCommitStatus(ctx context.Context, repo, sha, diff string) error {
	fmt.Println("I set a commit status")
	return nil
}
