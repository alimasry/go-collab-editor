package store

import (
	"context"
	"testing"

	"github.com/alielmasry/go-collab-editor/ot"
)

func TestMemoryStore_CreateAndGet(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	if err := s.Create(ctx, "doc1", "hello"); err != nil {
		t.Fatal(err)
	}

	info, err := s.Get(ctx, "doc1")
	if err != nil {
		t.Fatal(err)
	}
	if info.Content != "hello" || info.Version != 0 || info.ID != "doc1" {
		t.Errorf("unexpected info: %+v", info)
	}
}

func TestMemoryStore_CreateDuplicate(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	s.Create(ctx, "doc1", "")
	if err := s.Create(ctx, "doc1", ""); err == nil {
		t.Error("expected error for duplicate create")
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.Get(context.Background(), "nope")
	if err == nil {
		t.Error("expected error for missing document")
	}
}

func TestMemoryStore_List(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	s.Create(ctx, "a", "")
	s.Create(ctx, "b", "")
	s.Create(ctx, "c", "")

	docs, err := s.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 3 {
		t.Errorf("got %d docs, want 3", len(docs))
	}
}

func TestMemoryStore_UpdateContent(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	s.Create(ctx, "doc1", "hello")
	if err := s.UpdateContent(ctx, "doc1", "hello world", 1); err != nil {
		t.Fatal(err)
	}

	info, _ := s.Get(ctx, "doc1")
	if info.Content != "hello world" || info.Version != 1 {
		t.Errorf("unexpected: content=%q version=%d", info.Content, info.Version)
	}
}

func TestMemoryStore_Operations(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	s.Create(ctx, "doc1", "hello")

	op1 := ot.NewInsert(5, " world", 5)
	if err := s.AppendOperation(ctx, "doc1", op1, 1); err != nil {
		t.Fatal(err)
	}

	op2 := ot.NewDelete(0, 5, 11)
	if err := s.AppendOperation(ctx, "doc1", op2, 2); err != nil {
		t.Fatal(err)
	}

	// Get all ops
	ops, err := s.GetOperations(ctx, "doc1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 2 {
		t.Fatalf("got %d ops, want 2", len(ops))
	}

	// Get ops from version 1
	ops, err = s.GetOperations(ctx, "doc1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 1 {
		t.Fatalf("got %d ops, want 1", len(ops))
	}
}

func TestMemoryStore_OperationsNotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.GetOperations(context.Background(), "nope", 0)
	if err == nil {
		t.Error("expected error for missing document")
	}
}
