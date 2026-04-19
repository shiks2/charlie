package store

import (
	"sync"
	"time"

	"github.com/shiks2/charlie-gateway/internal/domain"
)

type MemoryStore struct {
	mu       sync.RWMutex
	chargers map[string]*domain.Charger
	sessions map[int]*domain.Session
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		chargers: make(map[string]*domain.Charger),
		sessions: make(map[int]*domain.Session),
	}
}

func (s *MemoryStore) UpsertCharger(id, vendor, model string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch, ok := s.chargers[id]
	if !ok {
		ch = &domain.Charger{ID: id}
		s.chargers[id] = ch
	}
	ch.Vendor = vendor
	ch.Model = model
	ch.Connected = true
	ch.LastHeartbeatAt = time.Now().UTC()
}

func (s *MemoryStore) GetAllChargers() []*domain.Charger {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chargers := make([]*domain.Charger, 0, len(s.chargers))
	for _, ch := range s.chargers {
		chargers = append(chargers, ch)
	}
	return chargers
}

func (s *MemoryStore) UpdateChargerHeartbeat(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ch, ok := s.chargers[id]; ok {
		ch.LastHeartbeatAt = time.Now().UTC()
		ch.Connected = true
	}
}
