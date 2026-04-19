package ocpp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	dbpkg "github.com/shiks2/charlie-gateway/internal/db"
	"github.com/shiks2/charlie-gateway/internal/domain"
	"github.com/shiks2/charlie-gateway/internal/emitter"
	"github.com/shiks2/charlie-gateway/internal/mapper"
	"github.com/shiks2/charlie-gateway/internal/store"
)

type CharlieCoreHandler struct {
	store   *store.MemoryStore
	emitter emitter.EventEmitter
	db      *dbpkg.DB
	orgID   string

	mu           sync.RWMutex
	chargerUUIDs map[string]string
}

func NewCharlieCoreHandler(store *store.MemoryStore, emitter emitter.EventEmitter, database *dbpkg.DB, orgID string) *CharlieCoreHandler {
	return &CharlieCoreHandler{
		store:        store,
		emitter:      emitter,
		db:           database,
		orgID:        orgID,
		chargerUUIDs: make(map[string]string),
	}
}

func (h *CharlieCoreHandler) rememberChargerUUID(chargePointID string, chargerUUID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.chargerUUIDs[chargePointID] = chargerUUID
}

func (h *CharlieCoreHandler) chargerUUID(chargePointID string) (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	chargerUUID, ok := h.chargerUUIDs[chargePointID]
	return chargerUUID, ok
}

func (h *CharlieCoreHandler) getOrFetchChargerUUID(ctx context.Context, chargePointID string) (string, error) {
	if chargerUUID, ok := h.chargerUUID(chargePointID); ok {
		return chargerUUID, nil
	}

	if h.db != nil {
		chargerUUID, err := h.db.LookupChargerUUID(ctx, h.orgID, chargePointID)
		if err != nil {
			return "", fmt.Errorf("db lookup: %w", err)
		}
		if chargerUUID != "" {
			h.rememberChargerUUID(chargePointID, chargerUUID)
			return chargerUUID, nil
		}
	}

	return "", fmt.Errorf("charger %s not registered - BootNotification not yet received", chargePointID)
}

func (h *CharlieCoreHandler) OnBootNotification(chargePointId string, request *core.BootNotificationRequest) (confirmation *core.BootNotificationConfirmation, err error) {
	fmt.Printf("[BOOT] charger %s (%s)\n", chargePointId, request.ChargePointModel)

	// 1. Update the Memory Store (B2B tracking)
	h.store.UpsertCharger(chargePointId, request.ChargePointVendor, request.ChargePointModel)

	if h.db != nil {
		chargerUUID, dbErr := h.db.UpsertCharger(context.Background(), h.orgID, &domain.Charger{
			ID:     chargePointId,
			Vendor: request.ChargePointVendor,
			Model:  request.ChargePointModel,
		})
		if dbErr != nil {
			log.Printf("db upsert charger failed for %s: %v", chargePointId, dbErr)
		} else {
			h.rememberChargerUUID(chargePointId, chargerUUID)
		}
	}

	// 2. Use the Mapper to create the CharlieEvent
	event := mapper.MapBootNotification(chargePointId, request)

	// 3. Emit the Event (The Bridge to Kafka/JSON)
	_ = h.emitter.Emit(event)

	return NewBootConfirmation(60), nil
}
func (h *CharlieCoreHandler) OnHeartbeat(chargePointId string, request *core.HeartbeatRequest) (confirmation *core.HeartbeatConfirmation, err error) {
	fmt.Printf(" [HEARTBEAT] %s is still alive\n", chargePointId)

	// Update charger's last heartbeat timestamp for watchdog monitoring
	h.store.UpdateChargerHeartbeat(chargePointId)

	if h.db != nil {
		if dbErr := h.db.UpdateChargerHeartbeat(context.Background(), h.orgID, chargePointId); dbErr != nil {
			log.Printf("db update charger heartbeat failed for %s: %v", chargePointId, dbErr)
		}
	}

	return NewHeartbeatConfirmation(), nil
}

func (h *CharlieCoreHandler) OnAuthorize(chargePointId string, request *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error) {
	fmt.Printf("[AUTHORIZE] charger %s - idTag: %s\n", chargePointId, request.IdTag)

	// Accept all authorizations (in production, verify against your user database)
	return core.NewAuthorizationConfirmation(types.NewIdTagInfo(types.AuthorizationStatusAccepted)), nil
}
func (h *CharlieCoreHandler) OnDataTransfer(chargePointId string, request *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	fmt.Printf("[DATA TRANSFER] charger %s - vendor: %s, messageID: %s\n", chargePointId, request.VendorId, request.MessageId)

	// Log the data transfer but don't block on processing
	return core.NewDataTransferConfirmation(core.DataTransferStatusAccepted), nil
}
func (h *CharlieCoreHandler) OnMeterValues(chargePointId string, request *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	fmt.Printf("[METER] charger %s - connector %d with %d meter readings\n", chargePointId, request.ConnectorId, len(request.MeterValue))

	// 1. Use the Mapper to create the CharlieEvent
	event := mapper.MapMeterValues(chargePointId, request)

	// 2. Emit the Event (The Bridge to Kafka/JSON)
	_ = h.emitter.Emit(event)

	return NewMeterValuesConfirmation(), nil
}
func (h *CharlieCoreHandler) OnStatusNotification(chargePointId string, request *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	fmt.Printf("[STATUS] charger %s - connector %d status: %s (error: %s)\n", chargePointId, request.ConnectorId, request.Status, request.ErrorCode)

	// 1. Use the Mapper to create the CharlieEvent
	event := mapper.MapStatusNotification(chargePointId, request)

	// 2. Emit the Event (The Bridge to Kafka/JSON)
	_ = h.emitter.Emit(event)

	if h.db != nil {
		chargerUUID, err := h.getOrFetchChargerUUID(context.Background(), chargePointId)
		if err != nil {
			log.Printf("db upsert connector status skipped for %s: %v", chargePointId, err)
		} else if dbErr := h.db.UpsertConnectorStatus(context.Background(), chargerUUID, request.ConnectorId, string(request.Status), string(request.ErrorCode)); dbErr != nil {
			log.Printf("db upsert connector status failed for %s connector %d: %v", chargePointId, request.ConnectorId, dbErr)
		}
	}

	return NewStatusConfirmation(), nil
}
func (h *CharlieCoreHandler) OnStartTransaction(chargePointId string, request *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	fmt.Printf("[START TXN] charger %s - idTag: %s, meter: %d\n", chargePointId, request.IdTag, request.MeterStart)

	// 0 means unresolved transaction ID; downstream billing should skip unresolved sessions.
	transactionID := 0

	if h.db != nil {
		chargerUUID, err := h.getOrFetchChargerUUID(context.Background(), chargePointId)
		if err != nil {
			log.Printf("db create session skipped for %s: %v", chargePointId, err)
		} else {
			sessionID, dbErr := h.db.CreateSession(context.Background(), h.orgID, chargerUUID, &domain.Session{
				ChargePointID: chargePointId,
				ConnectorID:   request.ConnectorId,
				IdTag:         request.IdTag,
				MeterStart:    request.MeterStart,
				StartedAt:     time.Now().UTC(),
			})
			if dbErr != nil {
				log.Printf("db create session failed for %s: %v", chargePointId, dbErr)
			} else {
				transactionID = int(sessionID)
			}
		}
	}

	// 1. Use the Mapper to create the CharlieEvent
	event := mapper.MapStartTransaction(chargePointId, request, transactionID)

	// 2. Emit the Event (The Bridge to Kafka/JSON)
	_ = h.emitter.Emit(event)

	return NewStartTransactionConfirmation(transactionID), nil
}
func (h *CharlieCoreHandler) OnStopTransaction(chargePointId string, request *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	fmt.Printf("[STOP TXN] charger %s - txnID: %d, meter: %d, reason: %s\n", chargePointId, request.TransactionId, request.MeterStop, request.Reason)

	// 1. Use the Mapper to create the CharlieEvent
	event := mapper.MapStopTransaction(chargePointId, request)

	// 2. Emit the Event (The Bridge to Kafka/JSON)
	_ = h.emitter.Emit(event)

	if h.db != nil {
		if dbErr := h.db.StopSession(context.Background(), request.TransactionId, request.MeterStop, string(request.Reason)); dbErr != nil {
			log.Printf("db stop session failed for %s txn %d: %v", chargePointId, request.TransactionId, dbErr)
		}
	}

	return NewStopTransactionConfirmation(), nil
}
