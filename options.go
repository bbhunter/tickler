package tickler

import (
	"context"
	"time"
)

type (
	JobName            = string
	BackgroundFunction = func() error
)

type status = int

const (
	statusUndefined status = iota
	statusSuccess
	statusFailure
)

const (
	defaultRequestLimit = 100
)

type Options struct {
	Limit int64
	Ctx   context.Context

	sema chan int
}

func newEvent(ctx context.Context, request Request, opts ...EventOption) *event {
	t := &event{
		fnOpts: &eventOptions{
			retryOpts: retryOptions{
				maxRetries: 1,
			},
		},
		f:        request.Job,
		Job:      request.Name,
		ctx:      ctx,
		ch:       make(chan struct{}),
		result:   statusSuccess,
		resultCh: make(chan status),
	}

	for _, opt := range opts {
		opt.apply(t.fnOpts)
	}

	return t
}

type retryOptions struct {
	maxRetries int
	backoff    *backoff
}

type eventOptions struct {
	waitFor   []JobName
	ifSuccess []JobName
	ifFailure []JobName
	retryOpts retryOptions
}

type EventOption interface {
	apply(*eventOptions)
}

type eventOption struct {
	f func(*eventOptions)
}

func (o *eventOption) apply(opts *eventOptions) {
	o.f(opts)
}

func newEventOption(f func(*eventOptions)) *eventOption {
	return &eventOption{
		f: f,
	}
}

func WaitFor(jobNames ...JobName) EventOption {
	return newEventOption(func(t *eventOptions) {
		t.waitFor = jobNames
	})
}

// IfSuccess runs the given function if the job(s) succeeds.
func IfSuccess(jobNames ...JobName) EventOption {
	return newEventOption(func(t *eventOptions) {
		t.ifSuccess = jobNames
	})
}

// IfFailure runs the given function if the job(s) fails.
func IfFailure(jobNames ...JobName) EventOption {
	return newEventOption(func(t *eventOptions) {
		t.ifFailure = jobNames
	})
}

// WithRetry runs the given function with the given number of retries.
// It uses the truncated exponential backoff algorithm to determine the delay between retries.
func WithRetry(maxRetries int) EventOption {
	return newEventOption(func(t *eventOptions) {
		t.retryOpts = retryOptions{
			maxRetries: maxRetries,
			backoff: &backoff{
				Min:     time.Millisecond * 100,
				Max:     time.Second * 10,
				Factor:  2,
				attempt: 0,
			},
		}
	})
}
