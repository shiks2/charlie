# Event Emitters Guide

This package provides multiple event emitter implementations for the Charlie Gateway, supporting both development and production use cases.

## Overview

The `EventEmitter` interface defines a contract for sending `CharlieEvent` instances to various backends:

```go
type EventEmitter interface {
	Emit(event domain.CharlieEvent) error
}
```

## Available Emitters

### 1. JSONEmitter (Development/Logging)

Logs events as JSON to stdout. Useful for local development and debugging.

```go
emitter := emitter.NewJSONEmitter()
err := emitter.Emit(event)
```

**Use Case**: Development environments, debugging, local testing

---

### 2. KafkaEmitter (Production)

Sends events to Apache Kafka for distributed processing and persistence.

```go
kafkaEmitter := emitter.NewKafkaEmitter("localhost:9092")
err := kafkaEmitter.Emit(event)
defer kafkaEmitter.Close()
```

**Features**:
- Topic-based routing based on event type
- Automatic writer creation and pooling
- Event ID used as partition key for ordering guarantees
- Context-aware message sending

**Topic Mapping**:
- `charger-events.boot` - Boot notifications
- `charger-events.status` - Status notifications
- `charger-events.offline` - Charger offline events
- `charger-events.stale` - Stale heartbeat events
- `charging-events.meter` - Meter value updates
- `charging-events.start` - Transaction start events
- `charging-events.stop` - Transaction stop events

---

### 3. MultiEmitter (Composite)

Sends events to multiple emitters simultaneously. Useful for scenarios where you need redundancy or multi-destination routing.

```go
multiEmitter := emitter.NewMultiEmitter(
    emitter.NewJSONEmitter(),  // Also log locally
    emitter.NewKafkaEmitter("localhost:9092"),  // Send to Kafka
)
err := multiEmitter.Emit(event)
```

**Use Case**: Production environments with both logging and Kafka, failover scenarios

---

## Usage Examples

### Local Development
```go
em := emitter.NewJSONEmitter()

event := domain.CharlieEvent{
    EventID:       "evt-123",
    EventType:     domain.EventTypeBootNotification,
    Version:       domain.EventAPIVersion,
    ChargePointID: "CP-001",
    Timestamp:     time.Now().UTC(),
    Payload:       map[string]interface{}{"vendor": "Tesla"},
}

if err := em.Emit(event); err != nil {
    log.Printf("Failed to emit event: %v", err)
}
```

### Production with Kafka
```go
em := emitter.NewKafkaEmitter("kafka-broker:9092")
defer em.Close()

// Emit your events
if err := em.Emit(event); err != nil {
    log.Printf("Failed to emit event to Kafka: %v", err)
}
```

### Dual Emission (Dev + Production)
```go
multiEm := emitter.NewMultiEmitter(
    emitter.NewJSONEmitter(),
    emitter.NewKafkaEmitter("kafka-broker:9092"),
)

// Events are sent to both stdout and Kafka
if err := multiEm.Emit(event); err != nil {
    log.Printf("One or more emitters failed: %v", err)
}
```

---

## Event Evolution

The `CharlieEvent` struct includes a `Version` field to support API evolution:

```go
type CharlieEvent struct {
    Version string `json:"version"` // Tracks schema version
    Payload map[string]interface{} `json:"payload"` // Event-specific data
    Metadata map[string]interface{} `json:"metadata,omitempty"` // Additional context
}
```

Future versions of the event schema can increment the `EventAPIVersion` constant, allowing downstream consumers to handle schema migrations gracefully.

---

## Metadata Usage

The `Metadata` field provides a flexible way to attach context without modifying the core event structure:

```go
event := domain.CharlieEvent{
    // ... other fields ...
    Metadata: map[string]interface{}{
        "correlation_id": "corr-456",
        "trace_id": "trace-789",
        "source": "simulator",
    },
}
```

---

## Error Handling

Each emitter may fail for different reasons:

- **JSONEmitter**: Failures are rare (JSON marshaling errors)
- **KafkaEmitter**: Network issues, broker unavailability, message size limits
- **MultiEmitter**: Reports errors from any failing sub-emitter

Implement appropriate retry logic based on your use case:

```go
if err := em.Emit(event); err != nil {
    // Log the error
    log.Printf("Emit failed: %v", err)
    
    // Implement retry logic if needed
    // Or fallback to alternative emitter
}
```

---

## Testing

For unit tests, create a mock emitter:

```go
type MockEmitter struct {
    Events []domain.CharlieEvent
    Error  error
}

func (m *MockEmitter) Emit(event domain.CharlieEvent) error {
    if m.Error != nil {
        return m.Error
    }
    m.Events = append(m.Events, event)
    return nil
}
```

---

## Future Enhancements

Potential emitter implementations:
- **CloudEventsEmitter**: CloudEvents standard format
- **HTTPEmitter**: HTTP/REST webhook delivery
- **S3Emitter**: Direct archival to AWS S3
- **PubSubEmitter**: Google Cloud Pub/Sub
- **SQSEmitter**: AWS SQS for decoupled processing
- **ElasticsearchEmitter**: Direct indexing for analytics
