package emitter

import (
	"fmt"

	"github.com/shiks2/charlie-gateway/internal/domain"
)

// MultiEmitter sends events to multiple emitters simultaneously
// Useful for sending to both a Kafka producer and local logger
type MultiEmitter struct {
	emitters []EventEmitter
}

// NewMultiEmitter creates a new multi-emitter that sends to all provided emitters
func NewMultiEmitter(emitters ...EventEmitter) *MultiEmitter {
	return &MultiEmitter{
		emitters: emitters,
	}
}

// Emit sends the event to all configured emitters
// Returns an error if any emitter fails, but continues attempting all emitters
func (m *MultiEmitter) Emit(event domain.CharlieEvent) error {
	var errors []error

	for _, emitter := range m.emitters {
		if err := emitter.Emit(event); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("one or more emitters failed: %v", errors)
	}

	return nil
}
