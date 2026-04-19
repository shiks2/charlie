package watchdog

import (
	"time"

	"github.com/shiks2/charlie-gateway/internal/domain"
	"github.com/shiks2/charlie-gateway/internal/emitter"
	"github.com/shiks2/charlie-gateway/internal/store"

	"github.com/google/uuid"
)

func StartHeartbeatWatchdog(st *store.MemoryStore, em emitter.EventEmitter, expectedInterval time.Duration) {
	ticker := time.NewTicker(15 * time.Second)

	go func() {
		for range ticker.C {
			chargers := st.GetAllChargers()
			now := time.Now().UTC()

			for _, ch := range chargers {
				age := now.Sub(ch.LastHeartbeatAt)

				if age > 3*expectedInterval {
					event := domain.CharlieEvent{
						EventID:       uuid.NewString(),
						EventType:     domain.EventTypeHeartbeatOffline,
						Version:       domain.EventAPIVersion,
						ChargePointID: ch.ID,
						CustomerID:    "",
						ConnectionID:  "",
						Timestamp:     now,
						Payload: map[string]interface{}{
							"last_heartbeat_at": ch.LastHeartbeatAt,
							"heartbeat_age_sec": int(age.Seconds()),
						},
					}
					_ = em.Emit(event)
				} else if age > 2*expectedInterval {
					event := domain.CharlieEvent{
						EventID:       uuid.NewString(),
						EventType:     domain.EventTypeHeartbeatStale,
						Version:       domain.EventAPIVersion,
						ChargePointID: ch.ID,
						CustomerID:    "",
						ConnectionID:  "",
						Timestamp:     now,
						Payload: map[string]interface{}{
							"last_heartbeat_at": ch.LastHeartbeatAt,
							"heartbeat_age_sec": int(age.Seconds()),
						},
					}
					_ = em.Emit(event)
				}
			}
		}
	}()
}
