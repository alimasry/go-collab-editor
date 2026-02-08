# Architecture Overview

## Package dependency graph

```mermaid
graph TD
    A[main.go] --> B[server/]
    B --> C[ot/]
    B --> D[store/]
    E[static/] -.->|served by| B
```

- **`ot/`** — Pure algorithm library with zero dependencies on other packages. Contains the operation model, transform function, and engine interface.
- **`server/`** — HTTP handler, WebSocket hub, per-document sessions, and client connection management. Depends on `ot/` and `store/`.
- **`store/`** — Document persistence abstraction. `MemoryStore` is the current implementation; the interface is designed for a future `FirestoreStore` drop-in.
- **`static/`** — Vanilla JS frontend with CodeMirror 5. Implements the same OT transform algorithm as the Go backend.

## Data flow

```mermaid
sequenceDiagram
    participant Client as Browser
    participant WS as WebSocket Handler
    participant Hub
    participant Session
    participant Engine as OT Engine
    participant Store as DocumentStore

    Client->>WS: Connect to /ws
    WS->>WS: Upgrade to WebSocket
    WS-->>Client: Connection established

    Client->>Hub: join {docId}
    Hub->>Store: Get or Create document
    Hub->>Session: Create session (if new)
    Session-->>Client: doc {content, revision, clients}

    Client->>Session: op {revision, operation}
    Session->>Engine: TransformIncoming(op, revision, history)
    Engine-->>Session: transformed operation
    Session->>Store: UpdateContent + AppendOperation
    Session-->>Client: ack {revision}
    Session-->>Client: op broadcast to other clients
```

## Key design decisions

**Single-goroutine-per-session**: Each document session runs in exactly one goroutine. All OT transforms, document mutations, and client broadcasts happen sequentially in that goroutine via channel receives. This eliminates the need for mutexes on document state and makes the concurrency model simple to reason about.

**Retain/insert/delete model**: Operations are sequences of components that walk the entire document left-to-right, rather than position-based point mutations. This makes transform and compose operations well-defined and composable.

**Interface-driven extensibility**: `ot.Engine` and `store.DocumentStore` are interfaces. New OT algorithms (Wave, CRDT adapters) or storage backends (Firestore, PostgreSQL) can be swapped in without changing server code.
