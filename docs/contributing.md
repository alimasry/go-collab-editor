# Contributing

## Development setup

```bash
git clone https://github.com/alimasry/go-collab-editor.git
cd go-collab-editor
go run main.go
```

Open [http://localhost:8080](http://localhost:8080) to verify the server works.

## Testing

Always run tests after modifying code:

```bash
# All tests
go test ./...

# Single package
go test -v ./ot

# Single test
go test -v ./ot -run TestTransform_InsertDelete
```

## Code organization

| Package | Responsibility |
|---------|---------------|
| `ot/` | Pure OT algorithm — zero external dependencies |
| `server/` | HTTP/WebSocket transport, hub, sessions, clients |
| `store/` | Document persistence abstraction |
| `static/` | Frontend (vanilla JS + CodeMirror 5) |

## Extending the system

### Adding a new OT engine

Implement the `Engine` interface in `ot/engine.go`:

```go
type Engine interface {
    TransformIncoming(op Operation, revision int, history []Operation) (Operation, error)
}
```

Then pass your engine to `server.NewHub()` in `main.go`.

### Adding a new storage backend

Implement the `DocumentStore` interface in `store/store.go`:

```go
type DocumentStore interface {
    Create(ctx context.Context, id, content string) error
    Get(ctx context.Context, id string) (*DocumentInfo, error)
    List(ctx context.Context) ([]DocumentInfo, error)
    UpdateContent(ctx context.Context, id, content string, version int) error
    AppendOperation(ctx context.Context, id string, op ot.Operation, version int) error
    GetOperations(ctx context.Context, id string, fromVersion int) ([]ot.Operation, error)
}
```

All methods take `context.Context` — this maps directly to Firestore, PostgreSQL, or other async backends.

## Documentation

To work on the docs site:

```bash
# Install tooling (first time)
make docs-setup

# Live preview at http://localhost:8000
make docs-serve

# Build and check for broken links
make docs-build

# Regenerate API reference after changing Go code
make docs-api
```

## Commit conventions

- Commit early and often
- Keep commits focused on a single change
