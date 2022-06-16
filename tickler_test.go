package tickler

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestTicklerIntegration(t *testing.T) {
	tl := New()
	tl.Start()
	tl.Enqueue(Request{Job: "1", F: func() error {
		fmt.Println("1")
		time.Sleep(time.Second * 3)
		return nil
	}})

	tl.Enqueue(Request{Job: "2", F: func() error {
		fmt.Println("2")
		time.Sleep(time.Second * 3)
		return nil
	}})

	tl.Enqueue(Request{Job: "3", F: func() error {
		fmt.Println("3")
		return nil
	}}, WaitForJobs("1", "2"))

	tl.Enqueue(Request{Job: "4", F: func() error {
		fmt.Println("4")
		return nil
	}}, WaitForJobs("3"))

	tl.Enqueue(Request{Job: "5", F: func() error {
		fmt.Println("5")
		return nil
	}}, WaitForJobs("3", "2", "4"))

	time.Sleep(time.Second * 6)

	tl.Enqueue(Request{Job: "6", F: func() error {
		fmt.Println("6")
		time.Sleep(time.Second * 3)
		// This should not be executed
		fmt.Println("6.1")
		return nil
	}})

	time.Sleep(time.Second * 4)

	tl.Enqueue(Request{Job: "7", F: func() error {
		fmt.Println("7")
		return nil
	}})

	time.Sleep(time.Second * 1)
}

func TestTicklerIfSuccess(t *testing.T) {
	tl := New()
	tl.Start()

	tl.Enqueue(Request{Job: "1", F: func() error {
		fmt.Println("1")
		return nil
	}})

	tl.Enqueue(Request{Job: "2", F: func() error {
		fmt.Println("2")
		return errors.New("error")
	}}, IfSuccess("1"))

	tl.Enqueue(Request{Job: "3", F: func() error {
		fmt.Println("3")
		return nil
	}}, IfSuccess("1", "2"))

	tl.Enqueue(Request{Job: "4", F: func() error {
		fmt.Println("4")
		return nil
	}}, IfSuccess("3"))

	time.Sleep(time.Second * 5)
}
