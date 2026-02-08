# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Workflow

- Always run `go test ./...` after writing or modifying code. Do not consider a task complete until tests pass.
- Commit early and often — don't wait until the end of a task.
- Never include `Co-Authored-By` lines or Anthropic email addresses in commit messages.
- When changing Go code that affects exported symbols (adding, removing, or modifying exported types/functions/methods), regenerate the API reference docs with `make docs-api` and include the updated `docs/api/` files in the commit.
- When changing architecture, protocols, or behavior documented in `docs/`, update the relevant doc pages to stay in sync.

## Build & Run Commands

```bash
go run main.go                  # Start server on :8080
go run main.go -addr :3000      # Custom port
go test ./...                   # Run all tests
go test -v ./ot                 # Run one package's tests
go test -v ./ot -run TestTransform_InsertDelete  # Run a single test
make docs-serve                 # Live preview docs at http://localhost:8000
make docs-build                 # Build docs site (strict mode)
make docs-api                   # Regenerate API reference from Go doc comments
```

## Architecture

Real-time collaborative text editor using the **Jupiter OT** (Operational Transformation) algorithm. Go backend with WebSocket transport, vanilla JS + CodeMirror 5 frontend.

### Package dependency graph

```
main.go → server/ → ot/
                   → store/
```

`ot/` is a pure algorithm library with zero dependencies on other packages.

### OT operation model

Operations use the **retain/insert/delete** sequential model (not position-based). An operation is a list of components that walk the entire document left-to-right:

```
Insert "X" at pos 2 in "hello":  [Retain(2), Insert("X"), Retain(3)]
Delete 2 chars at pos 1 in "hello":  [Retain(1), Delete(2), Retain(2)]
```

`Transform(a, b)` returns `(aPrime, bPrime)` satisfying:
`Apply(Apply(doc, a), bPrime) == Apply(Apply(doc, b), aPrime)`

Tie-break rule: when both operations insert at the same position, operation `a` (first argument) wins.

### Goroutine architecture

```
Hub (1 goroutine — routes clients to sessions)
└── Session per document (1 goroutine — serializes ALL ops, no mutexes needed)
    └── Client per connection (2 goroutines: ReadPump + WritePump)
```

The session's single-goroutine design is intentional — it eliminates race conditions on document state. All OT transform/apply/broadcast happens in that one goroutine via channel receives.

### Key interfaces

- **`ot.Engine`** — abstracts the OT algorithm. `JupiterEngine` is the current implementation. Future algorithms (Wave, CRDT adapter) implement this same interface without changing server code.
- **`store.DocumentStore`** — abstracts persistence. `MemoryStore` is the current implementation. Designed for a future `FirestoreStore` drop-in replacement (all methods take `context.Context`, map to Firestore operations).

### WebSocket protocol

All messages are JSON over `/ws`. Key flow:
1. Client sends `join` with `docId` → receives `doc` (full content + revision + client list)
2. Client sends `op` with `revision` (last known server version) + operation
3. Server transforms op against `history[revision:]`, applies, sends `ack` to sender + broadcasts `op` to others
4. `join`/`leave` messages broadcast presence changes

### Frontend OT client (static/js/main.js)

The JS implements the same transform algorithm as Go, plus a 3-state machine:
- **Synchronized** — send ops immediately
- **AwaitingAck** — one op in flight, buffer new edits
- **AwaitingAckWithBuffer** — op in flight + buffered edits; uses `compose()` to merge buffer

Document identity is the URL hash (`#abc123`). Same URL = same document.
