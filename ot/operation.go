package ot

import (
	"fmt"
	"strings"
)

// Component is a single step in an OT operation.
// Exactly one field should be set.
type Component struct {
	Retain int    `json:"retain,omitempty"` // keep N chars unchanged
	Insert string `json:"insert,omitempty"` // insert text at cursor
	Delete int    `json:"delete,omitempty"` // remove N chars at cursor
}

func (c Component) IsRetain() bool { return c.Retain > 0 && c.Insert == "" && c.Delete == 0 }
func (c Component) IsInsert() bool { return c.Insert != "" }
func (c Component) IsDelete() bool { return c.Delete > 0 && c.Insert == "" }

// Operation is a sequence of components that transforms a document.
// Components are applied left-to-right, advancing a cursor through the input.
type Operation struct {
	Ops []Component `json:"ops"`
}

// BaseLen returns the expected input document length.
func (op Operation) BaseLen() int {
	n := 0
	for _, c := range op.Ops {
		if c.IsRetain() {
			n += c.Retain
		} else if c.IsDelete() {
			n += c.Delete
		}
	}
	return n
}

// TargetLen returns the document length after the operation is applied.
func (op Operation) TargetLen() int {
	n := 0
	for _, c := range op.Ops {
		if c.IsRetain() {
			n += c.Retain
		} else if c.IsInsert() {
			n += len(c.Insert)
		}
	}
	return n
}

// IsNoop returns true if the operation makes no changes.
func (op Operation) IsNoop() bool {
	for _, c := range op.Ops {
		if c.IsInsert() || c.IsDelete() {
			return false
		}
	}
	return true
}

// Apply applies the operation to a document string.
func Apply(doc string, op Operation) (string, error) {
	if len(doc) != op.BaseLen() {
		return "", fmt.Errorf("document length %d != operation base length %d", len(doc), op.BaseLen())
	}
	var b strings.Builder
	pos := 0
	for _, c := range op.Ops {
		switch {
		case c.IsRetain():
			b.WriteString(doc[pos : pos+c.Retain])
			pos += c.Retain
		case c.IsInsert():
			b.WriteString(c.Insert)
		case c.IsDelete():
			pos += c.Delete
		}
	}
	return b.String(), nil
}

// NewInsert creates an operation that inserts text at pos in a document of docLen.
func NewInsert(pos int, text string, docLen int) Operation {
	var ops []Component
	if pos > 0 {
		ops = append(ops, Component{Retain: pos})
	}
	ops = append(ops, Component{Insert: text})
	if remaining := docLen - pos; remaining > 0 {
		ops = append(ops, Component{Retain: remaining})
	}
	return Operation{Ops: ops}
}

// NewDelete creates an operation that deletes count chars at pos in a document of docLen.
func NewDelete(pos, count, docLen int) Operation {
	var ops []Component
	if pos > 0 {
		ops = append(ops, Component{Retain: pos})
	}
	ops = append(ops, Component{Delete: count})
	if remaining := docLen - pos - count; remaining > 0 {
		ops = append(ops, Component{Retain: remaining})
	}
	return Operation{Ops: ops}
}
