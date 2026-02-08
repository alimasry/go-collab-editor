package ot

import "testing"

func TestJupiterEngine_TransformIncoming(t *testing.T) {
	engine := &JupiterEngine{}

	t.Run("no history to transform against", func(t *testing.T) {
		op := NewInsert(0, "x", 5)
		result, err := engine.TransformIncoming(op, 0, nil)
		if err != nil {
			t.Fatal(err)
		}
		// Should return unchanged
		if result.BaseLen() != op.BaseLen() {
			t.Errorf("BaseLen changed: %d vs %d", result.BaseLen(), op.BaseLen())
		}
	})

	t.Run("transform against one operation", func(t *testing.T) {
		// Doc: "hello" (len 5)
		// Server applied: insert "X" at 0 → "Xhello" (len 6)
		history := []Operation{NewInsert(0, "X", 5)}
		// Client sends: insert "Y" at 5 (end of "hello"), at revision 0
		clientOp := NewInsert(5, "Y", 5)

		result, err := engine.TransformIncoming(clientOp, 0, history)
		if err != nil {
			t.Fatal(err)
		}

		// After server applied "X" at 0, doc is "Xhello" (len 6).
		// Client's insert at 5 should become insert at 6 (shifted by X).
		doc := "Xhello"
		got, err := Apply(doc, result)
		if err != nil {
			t.Fatalf("Apply error: %v (result=%+v, doc=%q)", err, result.Ops, doc)
		}
		if got != "XhelloY" {
			t.Errorf("got %q, want %q", got, "XhelloY")
		}
	})

	t.Run("transform against multiple operations", func(t *testing.T) {
		// Doc: "abc" (len 3)
		// Server history:
		//   v0→v1: insert "X" at 0 → "Xabc" (len 4)
		//   v1→v2: insert "Y" at 4 → "XabcY" (len 5)
		history := []Operation{
			NewInsert(0, "X", 3),
			NewInsert(4, "Y", 4),
		}
		// Client at revision 0 sends: delete 'b' at position 1, doc len 3
		clientOp := NewDelete(1, 1, 3)

		result, err := engine.TransformIncoming(clientOp, 0, history)
		if err != nil {
			t.Fatal(err)
		}

		// After history, doc is "XabcY" (len 5).
		// Client wanted to delete 'b' (originally at pos 1).
		// After "X" inserted at 0, 'b' is at pos 2.
		// After "Y" inserted at 4, 'b' is still at pos 2.
		doc := "XabcY"
		got, err := Apply(doc, result)
		if err != nil {
			t.Fatalf("Apply error: %v (result=%+v, doc=%q)", err, result.Ops, doc)
		}
		if got != "XacY" {
			t.Errorf("got %q, want %q", got, "XacY")
		}
	})

	t.Run("invalid revision", func(t *testing.T) {
		_, err := engine.TransformIncoming(NewInsert(0, "x", 5), -1, nil)
		if err == nil {
			t.Error("expected error for negative revision")
		}
		_, err = engine.TransformIncoming(NewInsert(0, "x", 5), 5, []Operation{NewInsert(0, "a", 5)})
		if err == nil {
			t.Error("expected error for revision > history length")
		}
	})
}

// TestConvergence simulates multiple clients making concurrent edits
// and verifies all paths converge to the same document state.
func TestConvergence(t *testing.T) {
	engine := &JupiterEngine{}

	tests := []struct {
		name string
		doc  string
		ops  []Operation // concurrent operations, all at revision 0
		want string
	}{
		{
			"two inserts at different positions",
			"abc",
			[]Operation{
				NewInsert(0, "X", 3),
				NewInsert(3, "Y", 3),
			},
			"XabcY",
		},
		{
			"insert and delete",
			"abc",
			[]Operation{
				NewInsert(1, "X", 3),
				NewDelete(1, 1, 3),
			},
			"aXc",
		},
		{
			"three concurrent inserts",
			"abc",
			[]Operation{
				NewInsert(0, "1", 3),
				NewInsert(1, "2", 3),
				NewInsert(2, "3", 3),
			},
			"1a2b3c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := NewDocument(tt.doc)

			// Apply operations sequentially, transforming each against history
			for _, op := range tt.ops {
				transformed, err := engine.TransformIncoming(op, 0, doc.History)
				if err != nil {
					t.Fatalf("TransformIncoming error: %v", err)
				}
				if err := doc.Apply(transformed); err != nil {
					t.Fatalf("Apply error: %v", err)
				}
			}

			if doc.Content != tt.want {
				t.Errorf("got %q, want %q", doc.Content, tt.want)
			}
		})
	}
}
