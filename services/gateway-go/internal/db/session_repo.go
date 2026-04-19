package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/shiks2/charlie-gateway/internal/domain"
)

func (db *DB) CreateSession(ctx context.Context, orgID string, chargerUUID string, s *domain.Session) (int64, error) {
	if db == nil || db.pool == nil {
		return 0, errors.New("database is not initialized")
	}
	if s == nil {
		return 0, errors.New("session is nil")
	}

	tx, err := db.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin create session transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const insertQuery = `
INSERT INTO charging_sessions (
	charger_id,
	organisation_id,
	connector_id,
	ocpp_transaction_id,
	id_tag,
	meter_start,
	started_at,
	status
) VALUES ($1, $2, $3, $4, $5, $6, $7, 'active')
RETURNING id`

	var sessionID int64
	if err := tx.QueryRow(ctx, insertQuery, chargerUUID, orgID, s.ConnectorID, nil, s.IdTag, s.MeterStart, s.StartedAt).Scan(&sessionID); err != nil {
		return 0, fmt.Errorf("insert charging session: %w", err)
	}

	const updateQuery = `
UPDATE charging_sessions
SET ocpp_transaction_id = $2,
	updated_at = NOW()
WHERE id = $1`

	if _, err := tx.Exec(ctx, updateQuery, sessionID, sessionID); err != nil {
		return 0, fmt.Errorf("backfill charging session transaction id: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit create session transaction: %w", err)
	}

	return sessionID, nil
}

func (db *DB) StopSession(ctx context.Context, ocppTransactionID int, meterStop int, stopReason string) error {
	if db == nil || db.pool == nil {
		return errors.New("database is not initialized")
	}

	const query = `
UPDATE charging_sessions
SET meter_stop = $2,
	stopped_at = NOW(),
	status = 'completed',
	stop_reason = $3,
	updated_at = NOW()
WHERE ocpp_transaction_id = $1 AND status = 'active'`

	result, err := db.pool.Exec(ctx, query, ocppTransactionID, meterStop, stopReason)
	if err != nil {
		return fmt.Errorf("stop session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("stop session: no active session found for transaction %d", ocppTransactionID)
	}

	return nil
}
