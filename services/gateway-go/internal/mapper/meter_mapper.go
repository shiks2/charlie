package mapper

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/shiks2/charlie-gateway/internal/domain"
)

func MapMeterValues(chargePointId string, req *core.MeterValuesRequest) domain.CharlieEvent {
	connectorID := req.ConnectorId

	meterValues := make([]map[string]interface{}, 0)

	for _, mv := range req.MeterValue {
		item := map[string]interface{}{}

		if mv.Timestamp != nil {
			item["timestamp"] = mv.Timestamp.Time
		}

		sampled := make([]map[string]interface{}, 0)
		for _, sv := range mv.SampledValue {
			s := map[string]interface{}{
				"value": sv.Value,
			}

			if sv.Context != "" {
				s["context"] = string(sv.Context)
			}
			if sv.Format != "" {
				s["format"] = string(sv.Format)
			}
			if sv.Measurand != "" {
				s["measurand"] = string(sv.Measurand)
			}
			if sv.Phase != "" {
				s["phase"] = string(sv.Phase)
			}
			if sv.Location != "" {
				s["location"] = string(sv.Location)
			}
			if sv.Unit != "" {
				s["unit"] = string(sv.Unit)
			}

			sampled = append(sampled, s)
		}

		item["sampled_values"] = sampled
		meterValues = append(meterValues, item)
	}

	payload := map[string]interface{}{
		"connector_id":   req.ConnectorId,
		"transaction_id": req.TransactionId,
		"meter_values":   meterValues,
	}

	return domain.CharlieEvent{
		EventID:       uuid.NewString(),
		EventType:     domain.EventTypeMeterValues,
		Version:       domain.EventAPIVersion,
		ChargePointID: chargePointId,
		CustomerID:    "",
		ConnectionID:  fmt.Sprintf("%d", connectorID),
		Timestamp:     time.Now().UTC(),
		Payload:       payload,
	}
}
