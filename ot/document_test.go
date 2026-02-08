package ot

import "testing"

func TestDocument_Apply(t *testing.T) {
	doc := NewDocument("hello")
	if doc.Content != "hello" || doc.Version != 0 {
		t.Fatalf("initial state: content=%q version=%d", doc.Content, doc.Version)
	}

	// Insert " world"
	err := doc.Apply(NewInsert(5, " world", 5))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Content != "hello world" {
		t.Errorf("after insert: %q", doc.Content)
	}
	if doc.Version != 1 {
		t.Errorf("version = %d, want 1", doc.Version)
	}

	// Delete "world"
	err = doc.Apply(NewDelete(6, 5, 11))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Content != "hello " {
		t.Errorf("after delete: %q", doc.Content)
	}
	if doc.Version != 2 {
		t.Errorf("version = %d, want 2", doc.Version)
	}

	// History should have 2 operations
	if len(doc.History) != 2 {
		t.Errorf("history length = %d, want 2", len(doc.History))
	}
}

func TestDocument_ApplyNoop(t *testing.T) {
	doc := NewDocument("test")
	err := doc.Apply(Operation{[]Component{{Retain: 4}}})
	if err != nil {
		t.Fatal(err)
	}
	// Noop should not change version
	if doc.Version != 0 {
		t.Errorf("version = %d, want 0 after noop", doc.Version)
	}
}

func TestDocument_ApplyError(t *testing.T) {
	doc := NewDocument("hi")
	err := doc.Apply(NewInsert(0, "x", 10)) // wrong base length
	if err == nil {
		t.Error("expected error for length mismatch")
	}
	// Document should be unchanged
	if doc.Content != "hi" || doc.Version != 0 {
		t.Errorf("document modified after error: %q v%d", doc.Content, doc.Version)
	}
}
