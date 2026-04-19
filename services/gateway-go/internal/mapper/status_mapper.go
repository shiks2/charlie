package mapper

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/shiks2/charlie-gateway/internal/domain"
)

func MapStatusNotification(chargePointId string, req *core.StatusNotificationRequest) domain.CharlieEvent {
	connectorID := req.ConnectorId

	payload := map[string]interface{}{
		"connector_id":      req.ConnectorId,
		"status":            string(req.Status),
		"error_code":        string(req.ErrorCode),
		"info":              req.Info,
		"vendor_id":         req.VendorId,
		"vendor_error_code": req.VendorErrorCode,
	}

	if req.Timestamp != nil {
		payload["charger_timestamp"] = req.Timestamp.Time
	}

	return domain.CharlieEvent{
		EventID:       uuid.NewString(),
		EventType:     domain.EventTypeStatusNotification,
		Version:       domain.EventAPIVersion,
		ChargePointID: chargePointId,
		CustomerID:    "",
		ConnectionID:  fmt.Sprintf("%d", connectorID),
		Timestamp:     time.Now().UTC(),
		Payload:       payload,
	}
}
