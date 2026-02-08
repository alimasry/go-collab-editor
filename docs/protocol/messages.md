# Message Reference

All messages are JSON objects with a `type` field. This page documents every message type.

## Client to server

### `join`

Join or create a document session.

```json
{
  "type": "join",
  "docId": "abc123"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"join"` |
| `docId` | string | Document identifier |

If the document doesn't exist, the server creates it with empty content.

### `op`

Send an editing operation.

```json
{
  "type": "op",
  "docId": "abc123",
  "revision": 5,
  "op": {
    "ops": [
      {"retain": 3},
      {"insert": "hello"},
      {"retain": 7}
    ]
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"op"` |
| `docId` | string | Document identifier |
| `revision` | int | Client's last known server revision |
| `op` | Operation | The editing operation |

## Server to client

### `doc`

Full document state, sent in response to `join`.

```json
{
  "type": "doc",
  "docId": "abc123",
  "content": "hello world",
  "revision": 5,
  "clients": [
    {"id": "a1b2c3d4", "name": "Blue Fox", "color": "#3498db"},
    {"id": "e5f6g7h8", "name": "Red Owl", "color": "#e74c3c"}
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"doc"` |
| `docId` | string | Document identifier |
| `content` | string | Full document text |
| `revision` | int | Current server revision |
| `clients` | ClientInfo[] | List of connected users |

### `ack`

Acknowledges a successfully applied operation.

```json
{
  "type": "ack",
  "revision": 6
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"ack"` |
| `revision` | int | New server revision after applying the operation |

### `op` (broadcast)

A remote operation from another client.

```json
{
  "type": "op",
  "docId": "abc123",
  "revision": 6,
  "op": {
    "ops": [
      {"retain": 5},
      {"delete": 3},
      {"retain": 2}
    ]
  },
  "clientId": "a1b2c3d4"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"op"` |
| `docId` | string | Document identifier |
| `revision` | int | Server revision after this operation |
| `op` | Operation | The transformed operation |
| `clientId` | string | ID of the client that authored the operation |

### `join` (presence)

A new user connected to the document.

```json
{
  "type": "join",
  "clientId": "a1b2c3d4",
  "name": "Blue Fox",
  "color": "#3498db"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"join"` |
| `clientId` | string | New client's ID |
| `name` | string | Display name (randomly generated) |
| `color` | string | Hex color for presence indicators |

### `leave`

A user disconnected from the document.

```json
{
  "type": "leave",
  "clientId": "a1b2c3d4"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"leave"` |
| `clientId` | string | Departing client's ID |

### `error`

An error occurred processing a client message.

```json
{
  "type": "error",
  "message": "not joined to a document"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"error"` |
| `message` | string | Human-readable error description |

## Data types

### Operation

```json
{
  "ops": [
    {"retain": 5},
    {"insert": "hello"},
    {"delete": 3}
  ]
}
```

An array of components. Each component has exactly one field set:

| Field | Type | Description |
|-------|------|-------------|
| `retain` | int | Number of characters to keep unchanged |
| `insert` | string | Text to insert at the current position |
| `delete` | int | Number of characters to remove |

### ClientInfo

```json
{
  "id": "a1b2c3d4",
  "name": "Blue Fox",
  "color": "#3498db"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | 8-character alphanumeric client ID |
| `name` | string | Random name (adjective + animal, e.g. "Blue Fox") |
| `color` | string | Hex color from a predefined palette |
