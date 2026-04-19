package ocpp

import (
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

func NewBootConfirmation(interval int) *core.BootNotificationConfirmation {
	return core.NewBootNotificationConfirmation(
		types.NewDateTime(time.Now().UTC()),
		interval,
		core.RegistrationStatusAccepted,
	)
}

func NewHeartbeatConfirmation() *core.HeartbeatConfirmation {
	return core.NewHeartbeatConfirmation(types.NewDateTime(time.Now().UTC()))
}

func NewStatusConfirmation() *core.StatusNotificationConfirmation {
	return core.NewStatusNotificationConfirmation()
}

func NewMeterValuesConfirmation() *core.MeterValuesConfirmation {
	return core.NewMeterValuesConfirmation()
}

func NewStartTransactionConfirmation(transactionID int) *core.StartTransactionConfirmation {
	return core.NewStartTransactionConfirmation(
		types.NewIdTagInfo(types.AuthorizationStatusAccepted),
		transactionID,
	)
}

func NewStopTransactionConfirmation() *core.StopTransactionConfirmation {
	return core.NewStopTransactionConfirmation()
}
