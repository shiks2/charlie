package domain

import "time"

// Event type constants to prevent magic strings
const (
	EventTypeBootNotification   = "charger.booted"
	EventTypeStatusNotification = "charger.status.changed"
	EventTypeMeterValues        = "charging.session.updated"
	EventTypeStartTransaction   = "charging.session.started"
	EventTypeStopTransaction    = "charging.session.ended"
	EventTypeHeartbeatOffline   = "charger.offline"
	EventTypeHeartbeatStale     = "charger.stale"
)

// Current API version for event schema evolution
const EventAPIVersion = "1.0"

type CharlieEvent struct {
	EventID       string                 `json:"event_id"`
	EventType     string                 `json:"event_type"`
	Version       string                 `json:"version"`
	ChargePointID string                 `json:"charge_point_id"`
	CustomerID    string                 `json:"customer_id"`
	ConnectionID  string                 `json:"connection_id"`
	Timestamp     time.Time              `json:"timestamp"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Payload       map[string]interface{} `json:"payload"`
}
