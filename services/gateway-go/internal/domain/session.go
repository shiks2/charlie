package domain

import "time"

type Session struct {
	TransactionID int
	ChargePointID string
	ConnectorID   int
	IdTag         string
	MeterStart    int
	MeterStop     *int
	StartedAt     time.Time
	StoppedAt     *time.Time
}
