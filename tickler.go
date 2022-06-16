package tickler

import (
	"container/list"
	"context"
	"log"
	"sync"
	"time"
)

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

type Event struct {
	fnOpts *eventOptions
	Job    JobName
	f      BackgroundFunction
	ctx    context.Context
	result status

	ch       chan struct{}
	resultCh chan status
}

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

func (s *Tickler) dequeue() *Event {
	element := s.queue.Front()
	s.queue.Remove(element)
	return element.Value.(*Event)
}

func (s *Tickler) removeJob(event *Event) {
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

func (s *Tickler) process(event *Event) {
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

func (s *Tickler) Start() {
	go s.loop()
}

func (s *Tickler) Stop() {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	s.ctx = ctx
}

func (s *Tickler) SetContext(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ctx = ctx
}

// New creates a new Tickler with default settings.
func New(ctx context.Context, limit int) *Tickler {
	if ctx == nil {
		ctx = context.Background()
	}

	if limit < 1 {
		limit = DefaultRequestLimit
	}

	service := &Tickler{
		queue: list.New(),
		options: Options{
			sema: make(chan int, limit),
		},
		ctx:         ctx,
		loopSignal:  make(chan struct{}, limit),
		currentJobs: make(map[string]bool),
		jobsWaitFor: make(map[string][]chan struct{}),
		resultCh:    make(map[string][]chan status),
	}

	return service
}
