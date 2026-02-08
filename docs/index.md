# go-collab-editor

A real-time collaborative text editor built with Go and the **Jupiter OT** (Operational Transformation) algorithm. Multiple users edit the same document simultaneously with automatic conflict resolution.

## Features

- **Real-time collaboration** — multiple users edit the same document simultaneously via WebSocket
- **OT conflict resolution** — concurrent edits are automatically merged using the Jupiter algorithm
- **Pluggable architecture** — swap OT engines or storage backends via clean interfaces
- **Lightweight frontend** — vanilla JavaScript with CodeMirror 5, no build step required
- **Presence awareness** — see who's connected with colored user indicators

## Quick links

- [Getting Started](getting-started.md) — clone, run, and try collaboration in under a minute
- [Architecture](architecture/overview.md) — package graph, data flow, and design decisions
- [OT Algorithm](architecture/ot-algorithm.md) — deep dive into the retain/insert/delete operation model
- [WebSocket Protocol](protocol/websocket.md) — full protocol reference for building clients
- [API Reference](api/ot.md) — auto-generated Go package documentation

## Tech stack

| Component | Technology |
|-----------|------------|
| Backend | Go stdlib `net/http` + gorilla/websocket |
| OT algorithm | Jupiter (retain/insert/delete model) |
| Frontend | Vanilla JS + CodeMirror 5 |
| Persistence | In-memory (pluggable `DocumentStore` interface) |
