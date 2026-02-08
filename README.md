# go-collab-editor

Real-time collaborative text editor built with Go. Uses the Jupiter OT (Operational Transformation) algorithm to synchronize edits across multiple clients over WebSockets.

## Quick Start

```bash
go run main.go
```

Open `http://localhost:8080` in multiple browser tabs — edits sync in real time. The URL hash (e.g. `#abc123`) identifies the document; same URL = same document.

## Usage

```bash
go run main.go                          # Start on :8080 with in-memory storage
go run main.go -addr :3000              # Custom port
go run main.go -store firestore -project my-gcp-project  # Firestore persistence
```

## Testing

```bash
go test ./...
```

## Architecture

```
main.go → server/ → ot/
                   → store/
```

- **`ot/`** — Pure OT algorithm library (retain/insert/delete model, transform, compose, apply)
- **`server/`** — WebSocket hub, per-document sessions, client read/write pumps
- **`store/`** — Document persistence (`MemoryStore`, `FirestoreStore` with write-behind cache)
- **`static/`** — Vanilla JS + CodeMirror 5 frontend
