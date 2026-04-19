package emitter

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/shiks2/charlie-gateway/internal/domain"
)

// ManualMockEmitter is a manual mock implementation of EventEmitter
type ManualMockEmitter struct {
	mu      sync.Mutex
	emitted []domain.CharlieEvent
	err     error
	delay   time.Duration
}

func (m *ManualMockEmitter) Emit(event domain.CharlieEvent) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.emitted = append(m.emitted, event)
	return nil
}

func (m *ManualMockEmitter) getEmittedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.emitted)
}

func TestAsyncEmitter_Emit(t *testing.T) {
	mockWrapped := &ManualMockEmitter{}
	async := NewAsyncEmitter(mockWrapped, 10, 2)
	async.Start()
	defer async.Close()

	event := domain.CharlieEvent{EventID: "test-1"}
	
	err := async.Emit(event)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Wait for worker to process
	timeout := time.After(1 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("timed out waiting for event to be emitted")
		case <-ticker.C:
			if mockWrapped.getEmittedCount() == 1 {
				return
			}
		}
	}
}

func TestAsyncEmitter_GracefulShutdown(t *testing.T) {
	mockWrapped := &ManualMockEmitter{delay: 10 * time.Millisecond}
	async := NewAsyncEmitter(mockWrapped, 10, 1)
	async.Start()

	events := []domain.CharlieEvent{
		{EventID: "e1"},
		{EventID: "e2"},
		{EventID: "e3"},
	}

	for _, e := range events {
		_ = async.Emit(e)
	}

	// Close should wait for all events to be processed
	err := async.Close()
	if err != nil {
		t.Fatalf("expected no error on close, got %v", err)
	}

	if mockWrapped.getEmittedCount() != 3 {
		t.Errorf("expected 3 events emitted, got %d", mockWrapped.getEmittedCount())
	}
}

func TestAsyncEmitter_WorkerPool(t *testing.T) {
	// Use 5 workers and a delay to test concurrency
	mockWrapped := &ManualMockEmitter{delay: 20 * time.Millisecond}
	async := NewAsyncEmitter(mockWrapped, 20, 5)
	async.Start()

	start := time.Now()
	for i := 0; i < 10; i++ {
		_ = async.Emit(domain.CharlieEvent{EventID: "event"})
	}

	err := async.Close()
	if err != nil {
		t.Fatalf("expected no error on close, got %v", err)
	}
	duration := time.Since(start)

	count := mockWrapped.getEmittedCount()
	if count != 10 {
		t.Errorf("expected 10 events, got %d", count)
	}

	// 10 events with 20ms delay on 5 workers should take around 40ms + overhead
	// If it was sequential, it would take 200ms.
	if duration > 150*time.Millisecond {
		t.Errorf("execution took too long (%v), workers might not be running in parallel", duration)
	}
}

func TestAsyncEmitter_ErrorHandling(t *testing.T) {
	mockWrapped := &ManualMockEmitter{err: errors.New("kafka down")}
	async := NewAsyncEmitter(mockWrapped, 10, 1)
	async.Start()
	defer async.Close()

	_ = async.Emit(domain.CharlieEvent{EventID: "failed-event"})
	
	// Error is logged but doesn't crash the worker
	time.Sleep(50 * time.Millisecond)
}
