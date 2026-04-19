package emitter

import (
	"encoding/json"
	"log"

	"github.com/shiks2/charlie-gateway/internal/domain"
)

type EventEmitter interface {
	Emit(event domain.CharlieEvent) error
}

type JSONEmitter struct{}

func NewJSONEmitter() *JSONEmitter {
	return &JSONEmitter{}
}

func (e *JSONEmitter) Emit(event domain.CharlieEvent) error {
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	log.Println(string(b))
	return nil
}
