package store

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/alimasry/go-collab-editor/ot"
)

func testFirestoreClient(t *testing.T) *firestore.Client {
	t.Helper()
	projectID := os.Getenv("FIRESTORE_PROJECT")
	if projectID == "" {
		t.Skip("FIRESTORE_PROJECT not set, skipping Firestore tests")
	}
	client, err := firestore.NewClient(context.Background(), projectID)
	if err != nil {
		t.Fatalf("failed to create Firestore client: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

// uniqueDocID returns a unique document ID for test isolation.
func uniqueDocID(t *testing.T) string {
	return fmt.Sprintf("test-%s-%d", t.Name(), time.Now().UnixNano())
}

// cleanupDoc deletes a document and its operations subcollection.
func cleanupDoc(t *testing.T, s *FirestoreStore, docID string) {
	t.Helper()
	ctx := context.Background()

	// Delete operations subcollection.
	ops := s.opsCollection(docID).Documents(ctx)
	for {
		snap, err := ops.Next()
		if err != nil {
			break
		}
		snap.Ref.Delete(ctx)
	}

	// Delete document.
	s.docRef(docID).Delete(ctx)
}

func TestFirestoreStore_CreateAndGet(t *testing.T) {
	client := testFirestoreClient(t)
	s := NewFirestoreStore(client)
	ctx := context.Background()
	docID := uniqueDocID(t)
	t.Cleanup(func() { cleanupDoc(t, s, docID) })

	if err := s.Create(ctx, docID, "hello"); err != nil {
		t.Fatal(err)
	}

	info, err := s.Get(ctx, docID)
	if err != nil {
		t.Fatal(err)
	}
	if info.Content != "hello" || info.Version != 0 || info.ID != docID {
		t.Errorf("unexpected info: %+v", info)
	}
}

func TestFirestoreStore_CreateDuplicate(t *testing.T) {
	client := testFirestoreClient(t)
	s := NewFirestoreStore(client)
	ctx := context.Background()
	docID := uniqueDocID(t)
	t.Cleanup(func() { cleanupDoc(t, s, docID) })

	s.Create(ctx, docID, "")
	if err := s.Create(ctx, docID, ""); err == nil {
		t.Error("expected error for duplicate create")
	}
}

func TestFirestoreStore_GetNotFound(t *testing.T) {
	client := testFirestoreClient(t)
	s := NewFirestoreStore(client)
	_, err := s.Get(context.Background(), "nonexistent-doc-xyz")
	if err == nil {
		t.Error("expected error for missing document")
	}
}

func TestFirestoreStore_List(t *testing.T) {
	client := testFirestoreClient(t)
	s := NewFirestoreStore(client)
	ctx := context.Background()

	ids := make([]string, 3)
	for i := range ids {
		ids[i] = uniqueDocID(t) + fmt.Sprintf("-%d", i)
		t.Cleanup(func() { cleanupDoc(t, s, ids[i]) })
		s.Create(ctx, ids[i], "")
	}

	docs, err := s.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// At least our 3 docs should be present (there may be others from parallel tests).
	found := 0
	for _, d := range docs {
		for _, id := range ids {
			if d.ID == id {
				found++
			}
		}
	}
	if found != 3 {
		t.Errorf("found %d of our 3 docs in list", found)
	}
}

func TestFirestoreStore_UpdateContent(t *testing.T) {
	client := testFirestoreClient(t)
	s := NewFirestoreStore(client)
	ctx := context.Background()
	docID := uniqueDocID(t)
	t.Cleanup(func() { cleanupDoc(t, s, docID) })

	s.Create(ctx, docID, "hello")
	if err := s.UpdateContent(ctx, docID, "hello world", 1); err != nil {
		t.Fatal(err)
	}

	info, _ := s.Get(ctx, docID)
	if info.Content != "hello world" || info.Version != 1 {
		t.Errorf("unexpected: content=%q version=%d", info.Content, info.Version)
	}
}

func TestFirestoreStore_Operations(t *testing.T) {
	client := testFirestoreClient(t)
	s := NewFirestoreStore(client)
	ctx := context.Background()
	docID := uniqueDocID(t)
	t.Cleanup(func() { cleanupDoc(t, s, docID) })

	s.Create(ctx, docID, "hello")

	op1 := ot.NewInsert(5, " world", 5)
	if err := s.AppendOperation(ctx, docID, op1, 1); err != nil {
		t.Fatal(err)
	}

	op2 := ot.NewDelete(0, 5, 11)
	if err := s.AppendOperation(ctx, docID, op2, 2); err != nil {
		t.Fatal(err)
	}

	// Get all ops.
	ops, err := s.GetOperations(ctx, docID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 2 {
		t.Fatalf("got %d ops, want 2", len(ops))
	}

	// Get ops from version 1 (skip first op).
	ops, err = s.GetOperations(ctx, docID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 1 {
		t.Fatalf("got %d ops, want 1", len(ops))
	}
}

func TestFirestoreStore_OperationsNotFound(t *testing.T) {
	client := testFirestoreClient(t)
	s := NewFirestoreStore(client)
	_, err := s.GetOperations(context.Background(), "nonexistent-doc-xyz", 0)
	if err == nil {
		t.Error("expected error for missing document")
	}
}
