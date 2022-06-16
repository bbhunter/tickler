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
	tl.Enqueue(Request{Name: "1", Job: func() error {
		fmt.Println("1")
		time.Sleep(time.Second * 3)
		return nil
	}})

	tl.Enqueue(Request{Name: "2", Job: func() error {
		fmt.Println("2")
		time.Sleep(time.Second * 3)
		return nil
	}})

	tl.Enqueue(Request{Name: "3", Job: func() error {
		fmt.Println("3")
		return nil
	}}, WaitFor("1", "2"))

	tl.Enqueue(Request{Name: "4", Job: func() error {
		fmt.Println("4")
		return nil
	}}, WaitFor("3"))

	tl.Enqueue(Request{Name: "5", Job: func() error {
		fmt.Println("5")
		return nil
	}}, WaitFor("3", "2", "4"))

	time.Sleep(time.Second * 6)

	tl.Enqueue(Request{Name: "6", Job: func() error {
		fmt.Println("6")
		time.Sleep(time.Second * 3)
		// This should not be executed
		fmt.Println("6.1")
		return nil
	}})

	time.Sleep(time.Second * 4)

	tl.Enqueue(Request{Name: "7", Job: func() error {
		fmt.Println("7")
		return nil
	}})

	time.Sleep(time.Second * 1)
}

func TestTicklerIfSuccess(t *testing.T) {
	tl := New()
	tl.Start()

	tl.Enqueue(Request{Name: "1", Job: func() error {
		fmt.Println("1")
		return nil
	}})

	tl.Enqueue(Request{Name: "2", Job: func() error {
		fmt.Println("2")
		return errors.New("error")
	}}, IfSuccess("1"))

	tl.Enqueue(Request{Name: "3", Job: func() error {
		fmt.Println("3")
		return nil
	}}, IfSuccess("1", "2"))

	tl.Enqueue(Request{Name: "4", Job: func() error {
		fmt.Println("4")
		return nil
	}}, IfSuccess("3"))

	time.Sleep(time.Second * 5)
}

func TestTicklerRetry(t *testing.T) {
	tl := New()
	tl.Start()

	tl.Enqueue(Request{Name: "1", Job: func() error {
		fmt.Println("1")
		return errors.New("error")
	}}, WithRetry(10))

	tl.Enqueue(Request{Name: "2", Job: func() error {
		fmt.Println("2")
		return nil
	}}, WithRetry(10))

	time.Sleep(time.Second * 5)
}
