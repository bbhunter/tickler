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
	mu     sync.Mutex
	fnOpts *ticklerFunctionOptions
	Job    JobName
	f      BackgroundFunction
	ctx    context.Context

	ch chan struct{}
}

type Request struct {
	F   BackgroundFunction
	Job JobName
}

func newTicklerEvent(ctx context.Context, request Request, opts ...TicklerFunctionOption) *ticklerEvent {
	t := &ticklerEvent{
		fnOpts: &ticklerFunctionOptions{},
		f:      request.F,
		Job:    request.Job,
		ctx:    ctx,
		ch:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt.apply(t.fnOpts)
	}

	return t
}

type ticklerFunctionOptions struct {
	waitFor []string
	timeout time.Duration
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

func WithTimeout(duration time.Duration) TicklerFunctionOption {
	return newTicklerFunctionOption(func(t *ticklerFunctionOptions) {
		t.timeout = duration
	})
}

type ticklerOptions struct {
	sema chan int
}

type Tickler struct {
	mu         sync.Mutex
	ctx        context.Context
	queue      *list.List
	options    ticklerOptions
	loopSignal chan struct{}

	currentJobs map[JobName]bool
	jobsWaitFor map[JobName][]chan struct{}
}

func (s *Tickler) EnqueueWithContext(ctx context.Context, request Request, opts ...TicklerFunctionOption) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.tickleLoop()

	ticklerEvent := newTicklerEvent(ctx, request, opts...)

	s.currentJobs[request.Job] = true

	for _, v := range ticklerEvent.fnOpts.waitFor {
		s.jobsWaitFor[v] = append(s.jobsWaitFor[v], ticklerEvent.ch)
	}

	s.queue.PushBack(ticklerEvent)
	log.Printf("Added request to queue with length %d\n", s.queue.Len())
}

func (s *Tickler) Enqueue(request Request, opts ...TicklerFunctionOption) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.tickleLoop()

	ticklerEvent := newTicklerEvent(context.Background(), request, opts...)

	s.currentJobs[request.Job] = true

	for _, v := range ticklerEvent.fnOpts.waitFor {
		s.jobsWaitFor[v] = append(s.jobsWaitFor[v], ticklerEvent.ch)
	}

	s.queue.PushBack(ticklerEvent)
	log.Printf("Added request to queue with length %d\n", s.queue.Len())
}

func (s *Tickler) loop() {
	log.Println("Starting service loop")
	for {
		select {
		case <-s.loopSignal:
			s.tryDequeue()
		case <-s.ctx.Done():
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

	for _, v := range s.jobsWaitFor[event.Job] {
		v <- struct{}{}
	}

	delete(s.jobsWaitFor, event.Job)
	delete(s.currentJobs, event.Job)
}

func (s *Tickler) process(event *ticklerEvent) {
	defer s.replenish()
	defer s.removeJob(event)

	cnt := len(event.fnOpts.waitFor)

	ch := make(chan struct{})
	// Wait for other jobs to be done
	for {
		if cnt < 1 {
			break
		}

		select {
		case <-event.ch:
			cnt--
		default:
		}
	}

	go func() {
		if err := event.f(); err != nil {
			log.Printf("background task got error: %v", err)
		}

		ch <- struct{}{}
	}()

	if event.fnOpts.timeout > 0 {
		ctx, cancel := context.WithTimeout(event.ctx, event.fnOpts.timeout)
		event.mu.Lock()
		event.ctx = ctx
		event.mu.Unlock()
		defer cancel()
	}

	select {
	case <-event.ctx.Done():
		log.Printf("event context cancalled for %v", event.Job)
		return
	case <-ch:
		return
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

func (s *Tickler) Start() {
	go s.loop()
}

func (s *Tickler) Stop() {
	s.ctx.Done()
}

// New creates a new Tickler with default settings.
func New() *Tickler {
	service := &Tickler{
		queue: list.New(),
		options: ticklerOptions{
			sema: make(chan int, defaultRequestLimit),
		},
		ctx:         context.Background(),
		loopSignal:  make(chan struct{}, defaultRequestLimit),
		currentJobs: make(map[string]bool),
		jobsWaitFor: make(map[string][]chan struct{}),
	}

	return service
}
