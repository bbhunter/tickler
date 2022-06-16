package tickler

import (
	"container/list"
	"context"
	"log"
	"sync"
	"time"
)

// Tickler is a service that can be used to schedule and run the background tasks/jobs
type Tickler struct {
	mu         sync.Mutex
	ctx        context.Context
	queue      *list.List
	options    Options
	loopSignal chan struct{}

	currentJobs map[JobName]bool
	jobsWaitFor map[JobName][]chan struct{}
	resultCh    map[JobName][]chan status
}

type event struct {
	fnOpts *eventOptions
	Job    JobName
	f      BackgroundFunction
	ctx    context.Context
	result status

	ch       chan struct{}
	resultCh chan status
}

// Request is a request to be executed
type Request struct {
	Job  BackgroundFunction
	Name JobName
}

func (s *Tickler) EnqueueWithContext(ctx context.Context, request Request, opts ...EventOption) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.tickleLoop()

	ticklerEvent := newEvent(ctx, request, opts...)

	s.currentJobs[request.Name] = true

	for _, v := range ticklerEvent.fnOpts.waitFor {
		s.jobsWaitFor[v] = append(s.jobsWaitFor[v], ticklerEvent.ch)
	}

	s.queue.PushBack(ticklerEvent)
	log.Printf("Added request to queue with length %d\n", s.queue.Len())
}

func (s *Tickler) Enqueue(request Request, opts ...EventOption) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.tickleLoop()

	ticklerEvent := newEvent(context.Background(), request, opts...)

	s.currentJobs[request.Name] = true

	for _, v := range ticklerEvent.fnOpts.waitFor {
		s.jobsWaitFor[v] = append(s.jobsWaitFor[v], ticklerEvent.ch)
	}

	for _, v := range ticklerEvent.fnOpts.ifSuccess {
		s.resultCh[v] = append(s.resultCh[v], ticklerEvent.resultCh)
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

func (s *Tickler) dequeue() *event {
	element := s.queue.Front()
	s.queue.Remove(element)
	return element.Value.(*event)
}

func (s *Tickler) removeJob(event *event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, v := range s.jobsWaitFor[event.Job] {
		v <- struct{}{}
	}

	for _, v := range s.resultCh[event.Job] {
		v <- event.result
	}

	delete(s.jobsWaitFor, event.Job)
	delete(s.currentJobs, event.Job)
}

type eventResults struct {
	SucceededEvents int
	FailedEvents    int
	TotalEvents     int
}

func (s *Tickler) process(event *event) {
	defer s.replenish()
	defer s.removeJob(event)

	cnt := len(event.fnOpts.waitFor)
	eventRes := eventResults{
		SucceededEvents: len(event.fnOpts.ifSuccess),
		FailedEvents:    len(event.fnOpts.ifFailure),
		TotalEvents:     len(event.fnOpts.ifSuccess) + len(event.fnOpts.ifFailure),
	}

	// Wait for other jobs to be done
	for {
		if cnt < 1 && eventRes.TotalEvents == 0 {
			break
		}

		select {
		case <-event.ch:
			cnt--
		case r := <-event.resultCh:
			eventRes.TotalEvents--
			if r == statusSuccess {
				eventRes.SucceededEvents--
			} else {
				eventRes.FailedEvents--
			}
		}
	}

	// If all jobs are done, then we can proceed
	if eventRes.SucceededEvents != 0 || eventRes.FailedEvents != 0 {
		event.result = statusFailure
		return
	}

	select {
	case <-event.ctx.Done():
		log.Printf("event context cancelled for %v", event.Job)
		return
	default:
		for i := 0; i < event.fnOpts.retryOpts.maxRetries; i++ {
			if err := event.f(); err != nil {
				log.Printf("background task got error: %v", err)
				event.result = statusFailure
			} else {
				event.result = statusSuccess
			}

			if event.result == statusSuccess {
				break
			}

			backoff := event.fnOpts.retryOpts.backoff.Duration()
			log.Printf("Retrying in %v seconds", backoff)
			time.Sleep(backoff)
		}
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

// Start the tickler service
func (s *Tickler) Start() {
	go s.loop()
}

func (s *Tickler) Stop() {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	s.ctx = ctx
}

// SetContext sets the context for the tickler
func (s *Tickler) SetContext(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ctx = ctx
}

// GetContext returns the context of the tickler
func (s *Tickler) GetContext() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.ctx
}

// GetQueueLength returns the length of the queue
func (s *Tickler) GetQueueLength() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.queue.Len()
}

// GetCurrentJobs returns a list of jobs that are currently running
func (s *Tickler) GetCurrentJobs() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.currentJobs
}

// Limit the number of jobs that can be run at the same time
// the default size is 100.
func (s *Tickler) Limit(newLimit int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Increase the size of the semaphore
	s.options.sema = make(chan int, newLimit)
	s.loopSignal = make(chan struct{}, newLimit)
}

// New creates a new Tickler with default settings.
// The Tickler will not start until the Start method is called.
func New() *Tickler {
	service := &Tickler{
		queue: list.New(),
		options: Options{
			sema: make(chan int, defaultRequestLimit),
		},
		ctx:         context.Background(),
		loopSignal:  make(chan struct{}, defaultRequestLimit),
		currentJobs: make(map[string]bool),
		jobsWaitFor: make(map[string][]chan struct{}),
		resultCh:    make(map[string][]chan status),
	}

	return service
}
