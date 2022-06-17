package tickler

import (
	"errors"
	"github.com/goodjobtech/assert"
	"testing"
	"time"
)

func TestWaitFor(t *testing.T) {
	tl := New()
	tl.Start()

	var responses []string

	tl.Enqueue(Request{
		Name: "1",
		Job: func() error {
			responses = append(responses, "1")
			return nil
		},
	})

	tl.Enqueue(Request{
		Name: "2",
		Job: func() error {
			responses = append(responses, "2")
			return nil
		},
	}, WaitFor("1"))

	time.Sleep(time.Second * 3)
	tl.Stop()

	assert.Equal(t, "1", responses[0])
	assert.Equal(t, "2", responses[1])
}

func TestIfSuccess(t *testing.T) {
	tl := New()
	tl.Start()

	var responses []string

	tl.Enqueue(Request{
		Name: "1",
		Job: func() error {
			responses = append(responses, "1")
			return nil
		},
	})

	tl.Enqueue(Request{
		Name: "2",
		Job: func() error {
			responses = append(responses, "2")
			return nil
		},
	}, IfSuccess("1"))

	time.Sleep(time.Second * 3)

	assert.Equal(t, "1", responses[0])
	assert.Equal(t, "2", responses[1])

	tl.Enqueue(Request{
		Name: "3",
		Job: func() error {
			return errors.New("failed")
		},
	})

	tl.Enqueue(Request{
		Name: "4",
		Job: func() error {
			responses = append(responses, "4")
			return nil
		},
	}, IfSuccess("3"))

	time.Sleep(time.Second * 3)
	tl.Stop()

	assert.Equal(t, 2, len(responses))
}

func TestIfFailure(t *testing.T) {
	tl := New()
	tl.Start()

	var responses []string

	tl.Enqueue(Request{
		Name: "1",
		Job: func() error {
			responses = append(responses, "1")
			return errors.New("failed")
		},
	})

	tl.Enqueue(Request{
		Name: "2",
		Job: func() error {
			responses = append(responses, "2")
			return nil
		},
	}, IfFailure("1"))

	time.Sleep(time.Second * 5)

	assert.Equal(t, "1", responses[0])
	assert.Equal(t, "2", responses[1])

	tl.Enqueue(Request{
		Name: "3",
		Job: func() error {
			return nil
		},
	})

	tl.Enqueue(Request{
		Name: "4",
		Job: func() error {
			responses = append(responses, "4")
			return nil
		},
	}, IfFailure("3"))

	time.Sleep(time.Second * 3)
	tl.Stop()

	assert.Equal(t, 2, len(responses))
}

func TestWithRetry(t *testing.T) {
	tl := New()
	tl.Start()

	counter := 4

	tl.Enqueue(Request{
		Job: func() error {
			if counter != 0 {
				counter--
				return errors.New("need to retry")
			}

			return nil
		},
		Name: "test",
	}, WithRetry(3))

	time.Sleep(time.Second * 3)
	tl.Stop()

	assert.Equal(t, 1, counter)
}
