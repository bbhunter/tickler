package tickler

import (
	"container/list"
	"context"
	"log"
	"sync"
	"time"
)

type (
	JobName            = string
	BackgroundFunction = func() error
)

const (
	defaultRequestLimit = 100
)

type ticklerEvent struct {
	fnOpts *ticklerFunctionOptions
	Job    JobName
	f      BackgroundFunction
}

type Request struct {
	F   BackgroundFunction
	Job JobName
}

func newTicklerFunction(request Request, opts ...TicklerFunctionOption) *ticklerEvent {
	t := &ticklerEvent{
		fnOpts: &ticklerFunctionOptions{},
		f:      request.F,
		Job:    request.Job,
	}

	for _, opt := range opts {
		opt.apply(t.fnOpts)
	}

	return t
}

type ticklerFunctionOptions struct {
	waitFor []JobName
}

type TicklerFunctionOption interface {
	apply(*ticklerFunctionOptions)
}

type ticklerFunctionOption struct {
	f func(*ticklerFunctionOptions)
}

func (tfo *ticklerFunctionOption) apply(opts *ticklerFunctionOptions) {
	tfo.f(opts)
}

func newTicklerFunctionOption(f func(*ticklerFunctionOptions)) *ticklerFunctionOption {
	return &ticklerFunctionOption{
		f: f,
	}
}

func WaitForJobs(jobNames ...JobName) TicklerFunctionOption {
	return newTicklerFunctionOption(func(t *ticklerFunctionOptions) {
		t.waitFor = jobNames
	})
}

type TicklerOptions struct {
	sema chan int
}

type Tickler struct {
	mu          sync.Mutex
	queue       *list.List
	options     TicklerOptions
	loopSignal  chan struct{}
	currentJobs map[JobName]bool
}

func (s *Tickler) EnqueueRequest(request Request, opts ...TicklerFunctionOption) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ticklerFn := newTicklerFunction(request, opts...)

	s.currentJobs[request.Job] = true

	s.queue.PushBack(ticklerFn)
	log.Printf("Added request to queue with length %d\n", s.queue.Len())
	s.tickleLoop()
}

func (s *Tickler) loop(ctx context.Context) {
	log.Println("Starting service loop")
	for {
		select {
		case <-s.loopSignal:
			s.tryDequeue()
		case <-ctx.Done():
			log.Printf("Loop context cancelled")
			return
		}
	}
}

func (s *Tickler) tryDequeue() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.queue.Len() == 0 {
		return
	}

	select {
	case s.options.sema <- 1:
		request := s.dequeue()
		log.Printf("Dequeued request %v\n", request)
		go s.process(request)
	default:
		log.Printf("Received loop signal, but request limit is reached")
	}
}

func (s *Tickler) dequeue() *ticklerEvent {
	element := s.queue.Front()
	s.queue.Remove(element)
	return element.Value.(*ticklerEvent)
}

func (s *Tickler) removeJob(event *ticklerEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.currentJobs, event.Job)
}

func (s *Tickler) process(event *ticklerEvent) {
	defer s.replenish()
	defer s.removeJob(event)

	cnt := len(event.fnOpts.waitFor)

	for {
		if cnt == 0 {
			break
		}

		for idx, j := range event.fnOpts.waitFor {
			if !s.currentJobs[j] {
				event.fnOpts.waitFor = append(event.fnOpts.waitFor[:idx], event.fnOpts.waitFor[idx+1:]...)
				cnt--
				break

			}
		}

		time.Sleep(time.Second * 5)
	}

	if err := event.f(); err != nil {
		log.Printf("background task got error: %v", err)
	}
}

func (s *Tickler) replenish() {
	<-s.options.sema
	log.Printf("Replenishing semaphore, now %d/%d slots in use\n", len(s.options.sema), cap(s.options.sema))
	s.tickleLoop()
}

func (s *Tickler) tickleLoop() {
	select {
	case s.loopSignal <- struct{}{}:
	default:
	}
}

func (s *Tickler) Start(ctx context.Context) {
	go s.loop(ctx)
}

// New creates a new Tickler with default settings.
func New(ctx context.Context) *Tickler {
	service := &Tickler{
		queue: list.New(),
		options: TicklerOptions{
			sema: make(chan int, defaultRequestLimit),
		},
		loopSignal:  make(chan struct{}, 1),
		currentJobs: make(map[string]bool),
	}

	return service
}
