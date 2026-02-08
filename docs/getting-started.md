# Getting Started

## Prerequisites

- Go 1.25 or later
- A web browser

## Clone and run

```bash
git clone https://github.com/alimasry/go-collab-editor.git
cd go-collab-editor
go run main.go
```

The server starts on [http://localhost:8080](http://localhost:8080).

To use a custom port:

```bash
go run main.go -addr :3000
```

## Try collaboration

1. Open [http://localhost:8080](http://localhost:8080) in your browser
2. Note the URL hash (e.g. `#abc123`) — this identifies the document
3. Open the same URL in a second browser tab
4. Type in both tabs and watch edits appear in real-time

Share the URL with others to collaborate on the same document.

## Run tests

```bash
# Run all tests
go test ./...

# Run a specific package
go test -v ./ot

# Run a single test
go test -v ./ot -run TestTransform_InsertDelete
```

## Project structure

| Directory | Purpose |
|-----------|---------|
| `ot/` | Pure OT algorithm library (zero external dependencies) |
| `server/` | HTTP handler, WebSocket hub, sessions, and client management |
| `store/` | Document persistence (`MemoryStore` and `FirestoreStore` implementations) |
| `static/` | Frontend: HTML, CSS, and JavaScript (CodeMirror 5) |
| `main.go` | Server entry point — wires everything together |

## Deploy to Cloud Run

Build and deploy with Firestore as the storage backend:

```bash
# Build the container image
docker build -t go-collab-editor .

# Run locally with Firestore
docker run -p 8080:8080 \
  -e GCP_PROJECT=your-project-id \
  go-collab-editor -store firestore

# Deploy to Cloud Run
gcloud run deploy go-collab-editor \
  --source . \
  --set-env-vars GCP_PROJECT=your-project-id \
  --args="-store=firestore"
```

The server reads the `PORT` environment variable automatically (set by Cloud Run).

### Storage backends

| Flag | Description |
|------|-------------|
| `-store memory` | In-memory (default) — data lost on restart |
| `-store firestore` | Google Cloud Firestore — persistent across restarts |

When using Firestore, set the project ID via `-project` flag or `GCP_PROJECT` env var.
