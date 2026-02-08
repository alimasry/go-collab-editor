package ot

import "fmt"

// Document represents a collaborative document with its full operation history.
type Document struct {
	Content string
	Version int
	History []Operation
}

// NewDocument creates a new document with the given initial content.
func NewDocument(content string) *Document {
	return &Document{Content: content}
}

// Apply applies an operation to the document, appending it to history.
func (d *Document) Apply(op Operation) error {
	if op.IsNoop() {
		return nil
	}
	result, err := Apply(d.Content, op)
	if err != nil {
		return fmt.Errorf("apply to document v%d: %w", d.Version, err)
	}
	d.Content = result
	d.Version++
	d.History = append(d.History, op)
	return nil
}
