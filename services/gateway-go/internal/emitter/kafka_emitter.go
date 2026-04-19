package emitter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/segmentio/kafka-go"
	"github.com/shiks2/charlie-gateway/internal/domain"
)

// KafkaEmitter sends events to Kafka topics for production message queuing
type KafkaEmitter struct {
	mu      sync.Mutex
	writers map[string]*kafka.Writer
	broker  string
}

// NewKafkaEmitter creates a new Kafka emitter that writes to specified broker
// brokerURL should be in format "localhost:9092"
func NewKafkaEmitter(brokerURL string) *KafkaEmitter {
	return &KafkaEmitter{
		writers: make(map[string]*kafka.Writer),
		broker:  brokerURL,
	}
}

// Emit sends a CharlieEvent to the appropriate Kafka topic based on event type
func (k *KafkaEmitter) Emit(event domain.CharlieEvent) error {
	topic := k.topicForEventType(event.EventType)

	writer, err := k.getWriter(topic)
	if err != nil {
		return fmt.Errorf("failed to get kafka writer for topic %s: %w", topic, err)
	}

	// Marshal event to JSON
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create message with event ID as key for partitioning
	message := kafka.Message{
		Key:   []byte(event.EventID),
		Value: payload,
	}

	// Send message with context
	ctx := context.Background()
	err = writer.WriteMessages(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}

	log.Printf("Event %s emitted to Kafka topic %s", event.EventID, topic)
	return nil
}

// getWriter returns or creates a Kafka writer for the given topic
func (k *KafkaEmitter) getWriter(topic string) (*kafka.Writer, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	if writer, exists := k.writers[topic]; exists {
		return writer, nil
	}

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{k.broker},
		Topic:   topic,
	})

	k.writers[topic] = writer
	return writer, nil
}

// topicForEventType returns the appropriate Kafka topic for an event type
func (k *KafkaEmitter) topicForEventType(eventType string) string {
	// Map event types to Kafka topics for organization
	topicMap := map[string]string{
		domain.EventTypeBootNotification:   "charger-events.boot",
		domain.EventTypeStatusNotification: "charger-events.status",
		domain.EventTypeMeterValues:        "charging-events.meter",
		domain.EventTypeStartTransaction:   "charging-events.start",
		domain.EventTypeStopTransaction:    "charging-events.stop",
		domain.EventTypeHeartbeatOffline:   "charger-events.offline",
		domain.EventTypeHeartbeatStale:     "charger-events.stale",
	}

	if topic, exists := topicMap[eventType]; exists {
		return topic
	}

	// Default fallback topic
	return "events.default"
}

// Close gracefully closes all Kafka writers
func (k *KafkaEmitter) Close() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	for topic, writer := range k.writers {
		if err := writer.Close(); err != nil {
			log.Printf("failed to close kafka writer for topic %s: %v", topic, err)
			return err
		}
	}
	return nil
}
