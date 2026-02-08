package store

import (
	"context"
	"time"

	"github.com/alimasry/go-collab-editor/ot"
)

// DocumentInfo holds document metadata and content.
type DocumentInfo struct {
	ID        string
	Content   string
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DocumentStore abstracts document persistence.
// Implementations: MemoryStore (in-memory), FirestoreStore (Google Cloud Firestore).
type DocumentStore interface {
	Create(ctx context.Context, id, content string) error
	Get(ctx context.Context, id string) (*DocumentInfo, error)
	List(ctx context.Context) ([]DocumentInfo, error)
	UpdateContent(ctx context.Context, id, content string, version int) error
	AppendOperation(ctx context.Context, id string, op ot.Operation, version int) error
	GetOperations(ctx context.Context, id string, fromVersion int) ([]ot.Operation, error)
}
