package mapper

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/shiks2/charlie-gateway/internal/domain"
)

func MapStartTransaction(chargePointId string, req *core.StartTransactionRequest, transactionId int) domain.CharlieEvent {
	connectorID := req.ConnectorId

	payload := map[string]interface{}{
		"connector_id":   req.ConnectorId,
		"id_tag":         req.IdTag,
		"meter_start":    req.MeterStart,
		"transaction_id": transactionId,
	}

	if req.Timestamp != nil {
		payload["charger_timestamp"] = req.Timestamp.Time
	}

	return domain.CharlieEvent{
		EventID:       uuid.NewString(),
		EventType:     domain.EventTypeStartTransaction,
		Version:       domain.EventAPIVersion,
		ChargePointID: chargePointId,
		CustomerID:    "",
		ConnectionID:  fmt.Sprintf("%d", connectorID),
		Timestamp:     time.Now().UTC(),
		Payload:       payload,
	}
}

func MapStopTransaction(chargePointId string, req *core.StopTransactionRequest) domain.CharlieEvent {
	payload := map[string]interface{}{
		"transaction_id": req.TransactionId,
		"id_tag":         req.IdTag,
		"meter_stop":     req.MeterStop,
		"reason":         string(req.Reason),
	}

	if req.Timestamp != nil {
		payload["charger_timestamp"] = req.Timestamp.Time
	}

	if len(req.TransactionData) > 0 {
		payload["transaction_data_count"] = len(req.TransactionData)
	}

	return domain.CharlieEvent{
		EventID:       uuid.NewString(),
		EventType:     domain.EventTypeStopTransaction,
		Version:       domain.EventAPIVersion,
		ChargePointID: chargePointId,
		CustomerID:    "",
		ConnectionID:  "",
		Timestamp:     time.Now().UTC(),
		Payload:       payload,
	}
}
