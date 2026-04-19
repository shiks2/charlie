package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/shiks2/charlie-gateway/internal/domain"
)

func (db *DB) LookupChargerUUID(ctx context.Context, orgID string, chargePointID string) (string, error) {
	if db == nil || db.pool == nil {
		return "", errors.New("database is not initialized")
	}

	const query = `
SELECT id
FROM chargers
WHERE organisation_id = $1 AND charge_point_id = $2`

	var chargerUUID string
	err := db.pool.QueryRow(ctx, query, orgID, chargePointID).Scan(&chargerUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("lookup charger uuid: %w", err)
	}

	return chargerUUID, nil
}

func (db *DB) UpsertCharger(ctx context.Context, orgID string, c *domain.Charger) (string, error) {
	if db == nil || db.pool == nil {
		return "", errors.New("database is not initialized")
	}
	if c == nil {
		return "", errors.New("charger is nil")
	}

	const query = `
INSERT INTO chargers (
	organisation_id,
	charge_point_id,
	vendor,
	model,
	is_online,
	last_heartbeat_at,
	updated_at
) VALUES ($1, $2, $3, $4, TRUE, NOW(), NOW())
ON CONFLICT (organisation_id, charge_point_id) DO UPDATE
SET vendor = EXCLUDED.vendor,
	model = EXCLUDED.model,
	is_online = TRUE,
	last_heartbeat_at = NOW(),
	updated_at = NOW()
RETURNING id`

	var chargerUUID string
	if err := db.pool.QueryRow(ctx, query, orgID, c.ID, c.Vendor, c.Model).Scan(&chargerUUID); err != nil {
		return "", fmt.Errorf("upsert charger: %w", err)
	}

	return chargerUUID, nil
}

func (db *DB) UpdateChargerHeartbeat(ctx context.Context, orgID string, chargePointID string) error {
	if db == nil || db.pool == nil {
		return errors.New("database is not initialized")
	}

	const query = `
UPDATE chargers
SET last_heartbeat_at = NOW(),
	is_online = TRUE,
	updated_at = NOW()
WHERE organisation_id = $1 AND charge_point_id = $2`

	result, err := db.pool.Exec(ctx, query, orgID, chargePointID)
	if err != nil {
		return fmt.Errorf("update charger heartbeat: %w", err)
	}

	if result.RowsAffected() == 0 {
		// A heartbeat can arrive before BootNotification persists the charger row.
		return nil
	}

	return nil
}

func (db *DB) UpdateChargerOffline(ctx context.Context, orgID string, chargePointID string) error {
	if db == nil || db.pool == nil {
		return errors.New("database is not initialized")
	}

	const query = `
UPDATE chargers
SET is_online = FALSE,
	status = 'offline',
	updated_at = NOW()
WHERE organisation_id = $1 AND charge_point_id = $2`

	result, err := db.pool.Exec(ctx, query, orgID, chargePointID)
	if err != nil {
		return fmt.Errorf("update charger offline: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("update charger offline: charger not found for org %s charge point %s", orgID, chargePointID)
	}

	return nil
}

func (db *DB) UpsertConnectorStatus(ctx context.Context, chargerUUID string, connectorID int, status string, errorCode string) error {
	if db == nil || db.pool == nil {
		return errors.New("database is not initialized")
	}

	normalizedStatus := normalizeConnectorStatus(status)

	const query = `
INSERT INTO connectors (
	charger_id,
	connector_id,
	status,
	error_code,
	updated_at
) VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (charger_id, connector_id) DO UPDATE
SET status = $3,
	error_code = $4,
	updated_at = NOW()`

	if _, err := db.pool.Exec(ctx, query, chargerUUID, connectorID, normalizedStatus, errorCode); err != nil {
		return fmt.Errorf("upsert connector status: %w", err)
	}

	return nil
}

func normalizeConnectorStatus(status string) string {
	normalized := strings.ToLower(strings.TrimSpace(status))

	switch normalized {
	case "suspendedev":
		return "suspended_ev"
	case "suspendedevse":
		return "suspended_evse"
	default:
		return normalized
	}
}
