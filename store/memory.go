package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alimasry/go-collab-editor/ot"
)

type docRecord struct {
	info    DocumentInfo
	history []ot.Operation
}

// MemoryStore is an in-memory implementation of DocumentStore.
type MemoryStore struct {
	mu   sync.RWMutex
	docs map[string]*docRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{docs: make(map[string]*docRecord)}
}

func (s *MemoryStore) Create(_ context.Context, id, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.docs[id]; exists {
		return fmt.Errorf("document %q already exists", id)
	}
	now := time.Now()
	s.docs[id] = &docRecord{
		info: DocumentInfo{
			ID:        id,
			Content:   content,
			Version:   0,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	return nil
}

func (s *MemoryStore) Get(_ context.Context, id string) (*DocumentInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rec, ok := s.docs[id]
	if !ok {
		return nil, fmt.Errorf("document %q not found", id)
	}
	info := rec.info
	return &info, nil
}

func (s *MemoryStore) List(_ context.Context) ([]DocumentInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]DocumentInfo, 0, len(s.docs))
	for _, rec := range s.docs {
		result = append(result, rec.info)
	}
	return result, nil
}

func (s *MemoryStore) UpdateContent(_ context.Context, id, content string, version int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.docs[id]
	if !ok {
		return fmt.Errorf("document %q not found", id)
	}
	rec.info.Content = content
	rec.info.Version = version
	rec.info.UpdatedAt = time.Now()
	return nil
}

func (s *MemoryStore) AppendOperation(_ context.Context, id string, op ot.Operation, version int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.docs[id]
	if !ok {
		return fmt.Errorf("document %q not found", id)
	}
	rec.history = append(rec.history, op)
	rec.info.Version = version
	rec.info.UpdatedAt = time.Now()
	return nil
}

func (s *MemoryStore) GetOperations(_ context.Context, id string, fromVersion int) ([]ot.Operation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rec, ok := s.docs[id]
	if !ok {
		return nil, fmt.Errorf("document %q not found", id)
	}
	if fromVersion < 0 || fromVersion > len(rec.history) {
		return nil, fmt.Errorf("invalid version %d", fromVersion)
	}
	ops := make([]ot.Operation, len(rec.history)-fromVersion)
	copy(ops, rec.history[fromVersion:])
	return ops, nil
}
