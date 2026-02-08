package store

import (
	"context"
	"testing"
	"time"

	"github.com/alimasry/go-collab-editor/ot"
)

func TestCachedStore_ReadThrough(t *testing.T) {
	backing := NewMemoryStore()
	ctx := context.Background()

	// Pre-populate backing store.
	if err := backing.Create(ctx, "doc1", "hello"); err != nil {
		t.Fatal(err)
	}
	op := ot.NewInsert(5, " world", 5)
	if err := backing.AppendOperation(ctx, "doc1", op, 1); err != nil {
		t.Fatal(err)
	}

	cs := NewCachedStore(backing, time.Hour) // long interval — no auto flush
	defer cs.Close()

	// Get should load from backing.
	info, err := cs.Get(ctx, "doc1")
	if err != nil {
		t.Fatal(err)
	}
	if info.Content != "hello" || info.Version != 1 {
		t.Errorf("unexpected info: %+v", info)
	}

	// Operations should also be available.
	ops, err := cs.GetOperations(ctx, "doc1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 1 {
		t.Fatalf("got %d ops, want 1", len(ops))
	}
}

func TestCachedStore_WriteBehind(t *testing.T) {
	backing := NewMemoryStore()
	ctx := context.Background()

	cs := NewCachedStore(backing, 50*time.Millisecond)
	defer cs.Close()

	// Create doc in cache.
	if err := cs.Create(ctx, "doc1", "hello"); err != nil {
		t.Fatal(err)
	}

	// Backing should NOT have it yet.
	if _, err := backing.Get(ctx, "doc1"); err == nil {
		t.Error("expected backing to not have doc yet")
	}

	// Wait for flush.
	time.Sleep(150 * time.Millisecond)

	// Now backing should have it.
	info, err := backing.Get(ctx, "doc1")
	if err != nil {
		t.Fatal(err)
	}
	if info.ID != "doc1" {
		t.Errorf("unexpected doc ID: %s", info.ID)
	}
}

func TestCachedStore_OperationFlushTracking(t *testing.T) {
	backing := NewMemoryStore()
	ctx := context.Background()

	cs := NewCachedStore(backing, 50*time.Millisecond)
	defer cs.Close()

	if err := cs.Create(ctx, "doc1", "hello"); err != nil {
		t.Fatal(err)
	}

	// Append 3 ops.
	for i := 1; i <= 3; i++ {
		op := ot.NewInsert(0, "x", 4+i)
		if err := cs.AppendOperation(ctx, "doc1", op, i); err != nil {
			t.Fatal(err)
		}
	}

	// Wait for first flush.
	time.Sleep(150 * time.Millisecond)

	ops, err := backing.GetOperations(ctx, "doc1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 3 {
		t.Fatalf("after first flush: got %d ops, want 3", len(ops))
	}

	// Append 2 more.
	for i := 4; i <= 5; i++ {
		op := ot.NewInsert(0, "y", 4+i)
		if err := cs.AppendOperation(ctx, "doc1", op, i); err != nil {
			t.Fatal(err)
		}
	}

	// Wait for second flush.
	time.Sleep(150 * time.Millisecond)

	ops, err = backing.GetOperations(ctx, "doc1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 5 {
		t.Fatalf("after second flush: got %d ops, want 5", len(ops))
	}
}

func TestCachedStore_CloseFlushes(t *testing.T) {
	backing := NewMemoryStore()
	ctx := context.Background()

	cs := NewCachedStore(backing, time.Hour) // very long interval

	if err := cs.Create(ctx, "doc1", "hello"); err != nil {
		t.Fatal(err)
	}
	if err := cs.UpdateContent(ctx, "doc1", "hello world", 1); err != nil {
		t.Fatal(err)
	}
	op := ot.NewInsert(5, " world", 5)
	if err := cs.AppendOperation(ctx, "doc1", op, 1); err != nil {
		t.Fatal(err)
	}

	// Close triggers final flush.
	cs.Close()

	// Backing should have everything.
	info, err := backing.Get(ctx, "doc1")
	if err != nil {
		t.Fatal(err)
	}
	if info.Content != "hello world" || info.Version != 1 {
		t.Errorf("unexpected info: content=%q version=%d", info.Content, info.Version)
	}

	ops, err := backing.GetOperations(ctx, "doc1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 1 {
		t.Fatalf("got %d ops, want 1", len(ops))
	}
}

func TestCachedStore_PreLoadedDoc(t *testing.T) {
	backing := NewMemoryStore()
	ctx := context.Background()

	// Pre-populate backing with doc and 2 ops.
	if err := backing.Create(ctx, "doc1", "ab"); err != nil {
		t.Fatal(err)
	}
	op1 := ot.NewInsert(2, "c", 2)
	if err := backing.AppendOperation(ctx, "doc1", op1, 1); err != nil {
		t.Fatal(err)
	}
	op2 := ot.NewInsert(3, "d", 3)
	if err := backing.AppendOperation(ctx, "doc1", op2, 2); err != nil {
		t.Fatal(err)
	}

	cs := NewCachedStore(backing, time.Hour)

	// Load into cache via Get.
	if _, err := cs.Get(ctx, "doc1"); err != nil {
		t.Fatal(err)
	}

	// Append a new op via cache.
	op3 := ot.NewInsert(4, "e", 4)
	if err := cs.AppendOperation(ctx, "doc1", op3, 3); err != nil {
		t.Fatal(err)
	}

	// Close to flush.
	cs.Close()

	// Backing should have exactly 3 ops (no duplicates).
	ops, err := backing.GetOperations(ctx, "doc1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 3 {
		t.Fatalf("got %d ops, want 3", len(ops))
	}
}

func TestCachedStore_ListDelegatesToBacking(t *testing.T) {
	backing := NewMemoryStore()
	ctx := context.Background()

	backing.Create(ctx, "a", "")
	backing.Create(ctx, "b", "")

	cs := NewCachedStore(backing, time.Hour)
	defer cs.Close()

	docs, err := cs.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Errorf("got %d docs, want 2", len(docs))
	}
}
