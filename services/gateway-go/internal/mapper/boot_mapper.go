package mapper

import (
	"time"

	"github.com/google/uuid"
	"github.com/shiks2/charlie-gateway/internal/domain"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
)

func MapBootNotification(chargePointId string, req *core.BootNotificationRequest) domain.CharlieEvent {
	return domain.CharlieEvent{
		EventID:       uuid.NewString(),
		EventType:     domain.EventTypeBootNotification,
		Version:       domain.EventAPIVersion,
		ChargePointID: chargePointId,
		CustomerID:    "",
		ConnectionID:  "",
		Timestamp:     time.Now().UTC(),
		Payload: map[string]interface{}{
			"vendor": req.ChargePointVendor,
			"model":  req.ChargePointModel,
		},
	}
}
