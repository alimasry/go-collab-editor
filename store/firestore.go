package store

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/alimasry/go-collab-editor/ot"
)

// FirestoreStore is a Firestore-backed implementation of DocumentStore.
type FirestoreStore struct {
	client     *firestore.Client
	collection string
}

// NewFirestoreStore creates a new FirestoreStore using the given Firestore client.
func NewFirestoreStore(client *firestore.Client) *FirestoreStore {
	return &FirestoreStore{
		client:     client,
		collection: "documents",
	}
}

func (s *FirestoreStore) docRef(id string) *firestore.DocumentRef {
	return s.client.Collection(s.collection).Doc(id)
}

func (s *FirestoreStore) opsCollection(docID string) *firestore.CollectionRef {
	return s.docRef(docID).Collection("operations")
}

func zeroPad(version int) string {
	return fmt.Sprintf("%010d", version)
}

func (s *FirestoreStore) Create(ctx context.Context, id, content string) error {
	now := time.Now()
	_, err := s.docRef(id).Create(ctx, map[string]interface{}{
		"content":   content,
		"version":   0,
		"createdAt": now,
		"updatedAt": now,
	})
	if status.Code(err) == codes.AlreadyExists {
		return fmt.Errorf("document %q already exists", id)
	}
	return err
}

func (s *FirestoreStore) Get(ctx context.Context, id string) (*DocumentInfo, error) {
	snap, err := s.docRef(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, fmt.Errorf("document %q not found", id)
	}
	if err != nil {
		return nil, err
	}
	return snapshotToDocInfo(id, snap)
}

func snapshotToDocInfo(id string, snap *firestore.DocumentSnapshot) (*DocumentInfo, error) {
	data := snap.Data()
	content, _ := data["content"].(string)
	version, _ := data["version"].(int64)
	createdAt, _ := data["createdAt"].(time.Time)
	updatedAt, _ := data["updatedAt"].(time.Time)
	return &DocumentInfo{
		ID:        id,
		Content:   content,
		Version:   int(version),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func (s *FirestoreStore) List(ctx context.Context) ([]DocumentInfo, error) {
	iter := s.client.Collection(s.collection).Documents(ctx)
	defer iter.Stop()

	var result []DocumentInfo
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		info, err := snapshotToDocInfo(snap.Ref.ID, snap)
		if err != nil {
			return nil, err
		}
		result = append(result, *info)
	}
	return result, nil
}

func (s *FirestoreStore) UpdateContent(ctx context.Context, id, content string, version int) error {
	_, err := s.docRef(id).Update(ctx, []firestore.Update{
		{Path: "content", Value: content},
		{Path: "version", Value: version},
		{Path: "updatedAt", Value: time.Now()},
	})
	if status.Code(err) == codes.NotFound {
		return fmt.Errorf("document %q not found", id)
	}
	return err
}

func (s *FirestoreStore) AppendOperation(ctx context.Context, id string, op ot.Operation, version int) error {
	components := make([]map[string]interface{}, len(op.Ops))
	for i, c := range op.Ops {
		m := make(map[string]interface{})
		if c.Retain > 0 {
			m["retain"] = c.Retain
		}
		if c.Insert != "" {
			m["insert"] = c.Insert
		}
		if c.Delete > 0 {
			m["delete"] = c.Delete
		}
		components[i] = m
	}

	// Store with 0-based index: version 1 → index 0, matching MemoryStore's
	// history slice semantics where GetOperations(fromVersion) returns history[fromVersion:].
	index := version - 1
	_, err := s.opsCollection(id).Doc(zeroPad(index)).Set(ctx, map[string]interface{}{
		"ops":     components,
		"version": version,
	})
	return err
}

func (s *FirestoreStore) GetOperations(ctx context.Context, id string, fromVersion int) ([]ot.Operation, error) {
	// Verify document exists.
	_, err := s.docRef(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, fmt.Errorf("document %q not found", id)
	}
	if err != nil {
		return nil, err
	}

	iter := s.opsCollection(id).
		OrderBy(firestore.DocumentID, firestore.Asc).
		StartAt(zeroPad(fromVersion)).
		Documents(ctx)
	defer iter.Stop()

	var ops []ot.Operation
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		op, err := snapshotToOperation(snap)
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	return ops, nil
}

func snapshotToOperation(snap *firestore.DocumentSnapshot) (ot.Operation, error) {
	data := snap.Data()
	rawOps, ok := data["ops"].([]interface{})
	if !ok {
		return ot.Operation{}, fmt.Errorf("invalid ops field in operation %s", snap.Ref.ID)
	}

	components := make([]ot.Component, len(rawOps))
	for i, raw := range rawOps {
		m, ok := raw.(map[string]interface{})
		if !ok {
			return ot.Operation{}, fmt.Errorf("invalid component %d in operation %s", i, snap.Ref.ID)
		}
		var c ot.Component
		if v, ok := m["retain"].(int64); ok {
			c.Retain = int(v)
		}
		if v, ok := m["insert"].(string); ok {
			c.Insert = v
		}
		if v, ok := m["delete"].(int64); ok {
			c.Delete = int(v)
		}
		components[i] = c
	}
	return ot.Operation{Ops: components}, nil
}
