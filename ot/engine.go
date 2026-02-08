package ot

import "fmt"

// Engine abstracts the OT collaboration algorithm.
// Different algorithms (Jupiter, Wave, etc.) implement this interface.
type Engine interface {
	// TransformIncoming transforms a client operation (created at the given
	// revision) against all operations in the history since that revision.
	// Returns the operation transformed to apply at the current server state.
	TransformIncoming(op Operation, revision int, history []Operation) (Operation, error)
}

// JupiterEngine implements the Jupiter OT algorithm.
// It sequentially transforms the incoming operation against each
// server operation the client hasn't seen.
type JupiterEngine struct{}

func (e *JupiterEngine) TransformIncoming(op Operation, revision int, history []Operation) (Operation, error) {
	if revision < 0 || revision > len(history) {
		return Operation{}, fmt.Errorf("invalid revision %d (history len %d)", revision, len(history))
	}

	transformed := op
	for i := revision; i < len(history); i++ {
		var err error
		transformed, _, err = Transform(transformed, history[i])
		if err != nil {
			return Operation{}, fmt.Errorf("transform against history[%d]: %w", i, err)
		}
	}
	return transformed, nil
}
