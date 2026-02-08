package ot

import "testing"

func TestBaseLen(t *testing.T) {
	tests := []struct {
		name string
		op   Operation
		want int
	}{
		{"retain only", Operation{[]Component{{Retain: 5}}}, 5},
		{"insert only", Operation{[]Component{{Insert: "hi"}}}, 0},
		{"delete only", Operation{[]Component{{Delete: 3}}}, 3},
		{"mixed", Operation{[]Component{{Retain: 2}, {Insert: "x"}, {Delete: 1}, {Retain: 3}}}, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.BaseLen(); got != tt.want {
				t.Errorf("BaseLen() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTargetLen(t *testing.T) {
	tests := []struct {
		name string
		op   Operation
		want int
	}{
		{"retain only", Operation{[]Component{{Retain: 5}}}, 5},
		{"insert only", Operation{[]Component{{Insert: "hi"}}}, 2},
		{"delete only", Operation{[]Component{{Delete: 3}}}, 0},
		{"mixed", Operation{[]Component{{Retain: 2}, {Insert: "x"}, {Delete: 1}, {Retain: 3}}}, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.TargetLen(); got != tt.want {
				t.Errorf("TargetLen() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestIsNoop(t *testing.T) {
	tests := []struct {
		name string
		op   Operation
		want bool
	}{
		{"empty", Operation{}, true},
		{"retain only", Operation{[]Component{{Retain: 5}}}, true},
		{"has insert", Operation{[]Component{{Retain: 2}, {Insert: "x"}}}, false},
		{"has delete", Operation{[]Component{{Delete: 1}}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.IsNoop(); got != tt.want {
				t.Errorf("IsNoop() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApply(t *testing.T) {
	tests := []struct {
		name    string
		doc     string
		op      Operation
		want    string
		wantErr bool
	}{
		{
			"insert at start",
			"hello",
			NewInsert(0, "X", 5),
			"Xhello",
			false,
		},
		{
			"insert at end",
			"hello",
			NewInsert(5, "!", 5),
			"hello!",
			false,
		},
		{
			"insert in middle",
			"hello",
			NewInsert(2, "XY", 5),
			"heXYllo",
			false,
		},
		{
			"delete at start",
			"hello",
			NewDelete(0, 2, 5),
			"llo",
			false,
		},
		{
			"delete at end",
			"hello",
			NewDelete(3, 2, 5),
			"hel",
			false,
		},
		{
			"delete in middle",
			"hello",
			NewDelete(1, 3, 5),
			"ho",
			false,
		},
		{
			"length mismatch",
			"hi",
			NewInsert(0, "x", 5),
			"",
			true,
		},
		{
			"empty doc insert",
			"",
			Operation{[]Component{{Insert: "hi"}}},
			"hi",
			false,
		},
		{
			"retain all",
			"hello",
			Operation{[]Component{{Retain: 5}}},
			"hello",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Apply(tt.doc, tt.op)
			if (err != nil) != tt.wantErr {
				t.Errorf("Apply() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Apply() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewInsert(t *testing.T) {
	op := NewInsert(3, "abc", 10)
	if op.BaseLen() != 10 {
		t.Errorf("BaseLen() = %d, want 10", op.BaseLen())
	}
	if op.TargetLen() != 13 {
		t.Errorf("TargetLen() = %d, want 13", op.TargetLen())
	}
}

func TestNewDelete(t *testing.T) {
	op := NewDelete(2, 3, 10)
	if op.BaseLen() != 10 {
		t.Errorf("BaseLen() = %d, want 10", op.BaseLen())
	}
	if op.TargetLen() != 7 {
		t.Errorf("TargetLen() = %d, want 7", op.TargetLen())
	}
}
