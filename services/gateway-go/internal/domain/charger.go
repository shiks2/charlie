package domain

import "time"

type Charger struct {
	ID              string
	Vendor          string
	Model           string
	LastHeartbeatAt time.Time
	Connected       bool
	Status          string
}
