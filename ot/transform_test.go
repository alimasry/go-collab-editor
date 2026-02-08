package ot

import "testing"

// verifyTransform checks the OT invariant: Apply(Apply(doc,a),bPrime) == Apply(Apply(doc,b),aPrime)
func verifyTransform(t *testing.T, doc string, a, b Operation) {
	t.Helper()

	aPrime, bPrime, err := Transform(a, b)
	if err != nil {
		t.Fatalf("Transform error: %v", err)
	}

	// Path 1: apply a, then bPrime
	afterA, err := Apply(doc, a)
	if err != nil {
		t.Fatalf("Apply(doc, a) error: %v", err)
	}
	path1, err := Apply(afterA, bPrime)
	if err != nil {
		t.Fatalf("Apply(afterA, bPrime) error: %v\nafterA=%q, bPrime=%+v", err, afterA, bPrime)
	}

	// Path 2: apply b, then aPrime
	afterB, err := Apply(doc, b)
	if err != nil {
		t.Fatalf("Apply(doc, b) error: %v", err)
	}
	path2, err := Apply(afterB, aPrime)
	if err != nil {
		t.Fatalf("Apply(afterB, aPrime) error: %v\nafterB=%q, aPrime=%+v", err, afterB, aPrime)
	}

	if path1 != path2 {
		t.Errorf("convergence failed:\n  doc=%q\n  a=%+v → %q\n  b=%+v → %q\n  path1(a,bP)=%q\n  path2(b,aP)=%q\n  aPrime=%+v\n  bPrime=%+v",
			doc, a.Ops, afterA, b.Ops, afterB, path1, path2, aPrime.Ops, bPrime.Ops)
	}
}

func TestTransform_InsertInsert(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		a, b Operation
		want string // expected converged result
	}{
		{
			"both insert at different positions",
			"hello",
			NewInsert(1, "X", 5), // "hXello"
			NewInsert(3, "Y", 5), // "helYlo"
			"hXelYlo",
		},
		{
			"both insert at same position (a wins tie-break)",
			"hello",
			NewInsert(2, "A", 5),
			NewInsert(2, "B", 5),
			"heABllo",
		},
		{
			"insert at start and end",
			"abc",
			NewInsert(0, "X", 3),
			NewInsert(3, "Y", 3),
			"XabcY",
		},
		{
			"both insert at start",
			"abc",
			NewInsert(0, "X", 3),
			NewInsert(0, "Y", 3),
			"XYabc",
		},
		{
			"multi-char inserts",
			"ab",
			NewInsert(1, "XY", 2),
			NewInsert(1, "ZW", 2),
			"aXYZWb",
		},
		{
			"insert into empty doc",
			"",
			Operation{[]Component{{Insert: "A"}}},
			Operation{[]Component{{Insert: "B"}}},
			"AB",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifyTransform(t, tt.doc, tt.a, tt.b)
			// Also verify we get the expected result
			aPrime, bPrime, _ := Transform(tt.a, tt.b)
			afterA, _ := Apply(tt.doc, tt.a)
			result, _ := Apply(afterA, bPrime)
			if result != tt.want {
				t.Errorf("got %q, want %q (aPrime=%+v, bPrime=%+v)", result, tt.want, aPrime.Ops, bPrime.Ops)
			}
		})
	}
}

func TestTransform_InsertDelete(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		a, b Operation
		want string
	}{
		{
			"insert before delete",
			"abcde",
			NewInsert(1, "X", 5), // "aXbcde"
			NewDelete(3, 1, 5),   // "abce" (delete 'd')
			"aXbce",
		},
		{
			"insert after delete",
			"abcde",
			NewInsert(4, "X", 5), // "abcdXe"
			NewDelete(1, 1, 5),   // "acde" (delete 'b')
			"acdXe",
		},
		{
			"insert at delete position",
			"abcde",
			NewInsert(2, "X", 5), // "abXcde"
			NewDelete(2, 1, 5),   // "abde" (delete 'c')
			"abXde",
		},
		{
			"insert inside delete range",
			"abcde",
			NewInsert(2, "X", 5), // "abXcde"
			NewDelete(1, 3, 5),   // "ae" (delete 'bcd')
			"aXe",
		},
		{
			"delete all, insert in middle",
			"abc",
			NewInsert(1, "X", 3),
			NewDelete(0, 3, 3),
			"X",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifyTransform(t, tt.doc, tt.a, tt.b)
			aPrime, bPrime, _ := Transform(tt.a, tt.b)
			afterA, _ := Apply(tt.doc, tt.a)
			result, _ := Apply(afterA, bPrime)
			if result != tt.want {
				t.Errorf("got %q, want %q (aPrime=%+v, bPrime=%+v)", result, tt.want, aPrime.Ops, bPrime.Ops)
			}
		})
	}
}

func TestTransform_DeleteInsert(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		a, b Operation
		want string
	}{
		{
			"delete before insert",
			"abcde",
			NewDelete(0, 2, 5),   // "cde"
			NewInsert(3, "X", 5), // "abcXde"
			"cXde",
		},
		{
			"delete after insert",
			"abcde",
			NewDelete(3, 2, 5),   // "abc"
			NewInsert(1, "X", 5), // "aXbcde"
			"aXbc",
		},
		{
			"delete around insert position",
			"abcde",
			NewDelete(1, 3, 5),   // "ae" (delete 'bcd')
			NewInsert(2, "X", 5), // "abXcde"
			"aXe",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifyTransform(t, tt.doc, tt.a, tt.b)
			_, bPrime, _ := Transform(tt.a, tt.b)
			afterA, _ := Apply(tt.doc, tt.a)
			result, _ := Apply(afterA, bPrime)
			if result != tt.want {
				t.Errorf("got %q, want %q (bPrime=%+v)", result, tt.want, bPrime.Ops)
			}
		})
	}
}

func TestTransform_DeleteDelete(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		a, b Operation
		want string
	}{
		{
			"disjoint deletes (a before b)",
			"abcdef",
			NewDelete(0, 2, 6), // "cdef"
			NewDelete(4, 2, 6), // "abcd"
			"cd",
		},
		{
			"disjoint deletes (b before a)",
			"abcdef",
			NewDelete(4, 2, 6), // "abcd"
			NewDelete(0, 2, 6), // "cdef"
			"cd",
		},
		{
			"same range deleted",
			"abcdef",
			NewDelete(1, 3, 6), // "aef"
			NewDelete(1, 3, 6), // "aef"
			"aef",
		},
		{
			"overlapping deletes",
			"abcdef",
			NewDelete(1, 3, 6), // "aef" (delete 'bcd')
			NewDelete(2, 3, 6), // "abf" (delete 'cde')
			"af",
		},
		{
			"a contains b",
			"abcdef",
			NewDelete(1, 4, 6), // "af" (delete 'bcde')
			NewDelete(2, 2, 6), // "abef" (delete 'cd')
			"af",
		},
		{
			"delete entire doc twice",
			"abc",
			NewDelete(0, 3, 3),
			NewDelete(0, 3, 3),
			"",
		},
		{
			"adjacent deletes",
			"abcdef",
			NewDelete(0, 3, 6), // "def"
			NewDelete(3, 3, 6), // "abc"
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifyTransform(t, tt.doc, tt.a, tt.b)
			_, bPrime, _ := Transform(tt.a, tt.b)
			afterA, _ := Apply(tt.doc, tt.a)
			result, _ := Apply(afterA, bPrime)
			if result != tt.want {
				t.Errorf("got %q, want %q (bPrime=%+v)", result, tt.want, bPrime.Ops)
			}
		})
	}
}

func TestTransform_ErrorOnMismatchedBaseLens(t *testing.T) {
	a := NewInsert(0, "x", 5)
	b := NewInsert(0, "y", 3)
	_, _, err := Transform(a, b)
	if err == nil {
		t.Error("expected error for mismatched base lengths")
	}
}

func TestTransform_Noop(t *testing.T) {
	doc := "hello"
	a := Operation{[]Component{{Retain: 5}}}
	b := NewInsert(2, "X", 5)
	verifyTransform(t, doc, a, b)
}

func TestTransform_Unicode(t *testing.T) {
	// Note: Go strings are byte-indexed, so we work with byte positions.
	doc := "hello"
	a := NewInsert(5, " world", 5)
	b := NewInsert(0, ">>> ", 5)
	verifyTransform(t, doc, a, b)

	_, bPrime, _ := Transform(a, b)
	afterA, _ := Apply(doc, a)
	result, _ := Apply(afterA, bPrime)
	if result != ">>> hello world" {
		t.Errorf("got %q, want %q", result, ">>> hello world")
	}
}
