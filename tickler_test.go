package tickler

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestTicklerIntegration(t *testing.T) {
	ctx := context.Background()
	tl := New()
	tl.Start(ctx)
	tl.EnqueueRequest(Request{Job: "1", F: func() error {
		fmt.Println("1")
		time.Sleep(time.Second * 3)
		return nil
	}})

	tl.EnqueueRequest(Request{Job: "2", F: func() error {
		fmt.Println("2")
		time.Sleep(time.Second * 3)
		return nil
	}})

	tl.EnqueueRequest(Request{Job: "3", F: func() error {
		fmt.Println("3")
		return nil
	}}, WaitForJobs("1", "2"))

	tl.EnqueueRequest(Request{Job: "4", F: func() error {
		fmt.Println("4")
		return nil
	}}, WaitForJobs("3"))

	tl.EnqueueRequest(Request{Job: "5", F: func() error {
		fmt.Println("5")
		return nil
	}}, WaitForJobs("3", "2", "4"))

	time.Sleep(time.Second * 10)
}
