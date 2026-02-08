# OT Algorithm

The editor uses a **retain/insert/delete** sequential operation model. This page explains how operations work, how concurrent edits are merged, and how the Jupiter engine ties it all together.

## Operation model

An `Operation` is a list of `Component` values that walk the document left-to-right. Every component does one of three things:

| Component | Effect |
|-----------|--------|
| `Retain(n)` | Keep the next `n` characters unchanged |
| `Insert(s)` | Insert string `s` at the current cursor position |
| `Delete(n)` | Remove the next `n` characters |

An operation must account for the entire input document — the sum of `Retain` and `Delete` values must equal the document length.

### Examples

Insert `"X"` at position 2 in `"hello"` (length 5):

```
[Retain(2), Insert("X"), Retain(3)]
```

Result: `"heXllo"`

Delete 2 characters at position 1 in `"hello"` (length 5):

```
[Retain(1), Delete(2), Retain(2)]
```

Result: `"hlo"`

### Convenience constructors

```go
// Insert "world" at position 5 in a 5-char document
op := ot.NewInsert(5, "world", 5)
// Produces: [Retain(5), Insert("world")]

// Delete 3 chars at position 0 in a 10-char document
op := ot.NewDelete(0, 3, 10)
// Produces: [Delete(3), Retain(7)]
```

## Length invariants

Every operation has two length properties:

- **`BaseLen()`** — the expected input document length (sum of `Retain` + `Delete`)
- **`TargetLen()`** — the output document length after applying (sum of `Retain` + `Insert`)

`Apply` verifies that `len(doc) == op.BaseLen()` before executing.

## Apply

`Apply(doc, op)` walks the components left-to-right, building a result string:

- **Retain**: copy `n` characters from the input
- **Insert**: append the insert text
- **Delete**: skip `n` characters in the input

## Transform

`Transform(a, b)` takes two concurrent operations (both applied to the same document state) and returns `aPrime` and `bPrime` such that:

```
Apply(Apply(doc, a), bPrime) == Apply(Apply(doc, b), aPrime)
```

This is the **convergence property** — regardless of which operation is applied first, the final document state is the same.

### Algorithm

The transform uses an iterator-based approach. Two iterators walk through operations `a` and `b` simultaneously:

1. **Both insert**: operation `a` wins (tie-break rule). `a`'s insert goes into `aPrime`, and `bPrime` gets a `Retain` to skip over the inserted text.
2. **One inserts**: the insert goes into its prime, the other gets a `Retain`.
3. **Both consume input**: take the shorter chunk of the two:
    - **Retain vs Retain**: both primes retain
    - **Delete vs Retain**: the delete goes into its prime, the other drops the component
    - **Delete vs Delete**: both delete the same characters, nothing to emit

### Tie-break rule

When both operations insert at the same position, **operation `a` (the first argument) wins** — its insert is placed first. This provides deterministic ordering for concurrent inserts at the same position.

### Compact

After transform, `compact()` merges adjacent components of the same type. For example, `[Retain(2), Retain(3)]` becomes `[Retain(5)]`.

## Jupiter Engine

`JupiterEngine` implements the `Engine` interface. When a client sends an operation created at revision `r`, the engine sequentially transforms it against every server operation from `history[r:]`:

```go
func (e *JupiterEngine) TransformIncoming(op Operation, revision int, history []Operation) (Operation, error) {
    transformed := op
    for i := revision; i < len(history); i++ {
        transformed, _, _ = Transform(transformed, history[i])
    }
    return transformed, nil
}
```

After transformation, the operation is safe to apply at the current server state.
