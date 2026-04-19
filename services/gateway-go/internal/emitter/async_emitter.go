package emitter

import (
	"log"
	"sync"

	"github.com/shiks2/charlie-gateway/internal/domain"
)

// AsyncEmitter wraps an existing EventEmitter and offloads Emit operations to a background worker pool
type AsyncEmitter struct {
	wrapped   EventEmitter
	ch        chan domain.CharlieEvent
	workers   int
	wg        sync.WaitGroup
	quit      chan struct{}
	onceStart sync.Once
	onceStop  sync.Once
}

// NewAsyncEmitter creates a new AsyncEmitter with the specified buffer size and worker count
func NewAsyncEmitter(wrapped EventEmitter, bufferSize int, workers int) *AsyncEmitter {
	if bufferSize <= 0 {
		bufferSize = 1024
	}
	if workers <= 0 {
		workers = 5
	}

	return &AsyncEmitter{
		wrapped: wrapped,
		ch:      make(chan domain.CharlieEvent, bufferSize),
		workers: workers,
		quit:    make(chan struct{}),
	}
}

// Start initializes the worker pool. This method should be called once after creation.
func (a *AsyncEmitter) Start() {
	a.onceStart.Do(func() {
		for i := 0; i < a.workers; i++ {
			a.wg.Add(1)
			go a.worker(i)
		}
		log.Printf("AsyncEmitter started with %d workers", a.workers)
	})
}

// Emit pushes the event to the internal channel. This is non-blocking unless the buffer is full.
func (a *AsyncEmitter) Emit(event domain.CharlieEvent) error {
	select {
	case <-a.quit:
		log.Printf("AsyncEmitter already closed, dropping event %s", event.EventID)
		return nil
	case a.ch <- event:
		return nil
	}
}

// Close gracefully shuts down the AsyncEmitter, waiting for all workers to finish processing the buffer.
func (a *AsyncEmitter) Close() error {
	a.onceStop.Do(func() {
		close(a.quit) // Signal workers to stop receiving new events (if we wanted immediate stop)
		// However, for graceful shutdown, we close the channel to drain it.
		close(a.ch)
		a.wg.Wait()
		log.Println("AsyncEmitter shut down gracefully")
	})
	return nil
}

func (a *AsyncEmitter) worker(id int) {
	defer a.wg.Done()
	log.Printf("AsyncEmitter worker %d started", id)

	for event := range a.ch {
		if err := a.wrapped.Emit(event); err != nil {
			log.Printf("AsyncEmitter worker %d: failed to emit event %s: %v", id, event.EventID, err)
		}
	}
	log.Printf("AsyncEmitter worker %d stopped", id)
}
