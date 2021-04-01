package app

import (
	"context"
	"encoding/base64"
	"fmt"
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

func CheckPR(ctx workflow.Context, details CheckDetails) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute, // Max 1 minute before quitting
	})

	// Start both tests. We get back futures. They transparently handle
	// retries and persisting the results.
	old := workflow.ExecuteActivity(ctx, Test, details.Repo, details.Old)
	new := workflow.ExecuteActivity(ctx, Test, details.Repo, details.New)

	// we need both results, so its fine to wait to resolve in either order.
	var oldRes, newRes string

	if err := old.Get(ctx, &oldRes); err != nil {
		return err
	}

	if err := new.Get(ctx, &newRes); err != nil {
		return err
	}

	// fast to do and deterministic. run inside the workflow.
	diff := diffResults(oldRes, newRes)

	// Resolve the final task and finish.
	err := workflow.ExecuteActivity(ctx, SetCommitStatus, details.Repo, details.PR, diff).Get(ctx, nil)
	return err
}

func Test(ctx context.Context, repo, sha string) (string, error) {
	// fully asnyk
	token := base64.StdEncoding.EncodeToString(activity.GetInfo(ctx).TaskToken)
	fmt.Println("task token is", token)

	// magic return. pass the zero value of your result, and a sentinel error.
	return "", activity.ErrResultPending
}

func diffResults(old, new string) string {
	return "new bad code added. :( :( :("
}

func SetCommitStatus(ctx context.Context, repo, sha, diff string) error {
	return nil
}
