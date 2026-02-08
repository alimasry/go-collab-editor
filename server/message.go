package server

import (
	"encoding/json"

	"github.com/alimasry/go-collab-editor/ot"
)

// Message types exchanged over WebSocket.
const (
	MsgJoin  = "join"
	MsgLeave = "leave"
	MsgOp    = "op"
	MsgAck   = "ack"
	MsgDoc   = "doc"
	MsgError = "error"
)

// ClientMessage is a message from client to server.
type ClientMessage struct {
	Type     string       `json:"type"`
	DocID    string       `json:"docId,omitempty"`
	Revision int          `json:"revision"`
	Op       ot.Operation `json:"op,omitempty"`
}

// ServerMessage is a message from server to client.
type ServerMessage struct {
	Type     string       `json:"type"`
	DocID    string       `json:"docId,omitempty"`
	Content  string       `json:"content"`
	Revision int          `json:"revision"`
	Op       ot.Operation `json:"op,omitempty"`
	ClientID string       `json:"clientId,omitempty"`
	Name     string       `json:"name,omitempty"`
	Color    string       `json:"color,omitempty"`
	Message  string       `json:"message,omitempty"`
	Clients  []ClientInfo `json:"clients,omitempty"`
}

// ClientInfo describes a connected user.
type ClientInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Encode serializes a ServerMessage to JSON bytes.
func (m ServerMessage) Encode() []byte {
	b, _ := json.Marshal(m)
	return b
}
