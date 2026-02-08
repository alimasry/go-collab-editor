package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alimasry/go-collab-editor/ot"
	"github.com/alimasry/go-collab-editor/store"
)

func ctx() context.Context { return context.Background() }

// mockClient creates a client without a real WebSocket connection, for testing.
func mockClient(id string) *Client {
	return &Client{
		ID:    id,
		Name:  "Test " + id,
		Color: "#000000",
		send:  make(chan []byte, 256),
	}
}

// recvMsg reads one message from a mock client's send channel with timeout.
func recvMsg(t *testing.T, c *Client) ServerMessage {
	t.Helper()
	select {
	case data := <-c.send:
		var msg ServerMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		return msg
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
		return ServerMessage{}
	}
}

func TestSession_JoinAndReceiveDoc(t *testing.T) {
	st := store.NewMemoryStore()
	st.Create(ctx(), "doc1", "hello")
	engine := &ot.JupiterEngine{}
	s := newSession("doc1", "hello", engine, st)
	go s.Run()
	defer close(s.stop)

	c := mockClient("c1")
	s.join <- c
	msg := recvMsg(t, c)

	if msg.Type != MsgDoc {
		t.Fatalf("expected doc message, got %q", msg.Type)
	}
	if msg.Content != "hello" {
		t.Errorf("content = %q, want %q", msg.Content, "hello")
	}
	if msg.Revision != 0 {
		t.Errorf("revision = %d, want 0", msg.Revision)
	}
}

func TestSession_OpTransformAndBroadcast(t *testing.T) {
	st := store.NewMemoryStore()
	st.Create(ctx(), "doc1", "abc")
	engine := &ot.JupiterEngine{}
	s := newSession("doc1", "abc", engine, st)
	go s.Run()
	defer close(s.stop)

	c1 := mockClient("c1")
	c2 := mockClient("c2")
	s.join <- c1
	s.join <- c2
	recvMsg(t, c1) // doc
	recvMsg(t, c2) // doc
	recvMsg(t, c1) // c2 join notification

	// c1 sends an insert at position 0
	op := ot.NewInsert(0, "X", 3)
	s.incoming <- opMessage{client: c1, msg: ClientMessage{Type: MsgOp, DocID: "doc1", Revision: 0, Op: op}}

	// c1 should get ack
	ack := recvMsg(t, c1)
	if ack.Type != MsgAck {
		t.Fatalf("expected ack, got %q", ack.Type)
	}
	if ack.Revision != 1 {
		t.Errorf("ack revision = %d, want 1", ack.Revision)
	}

	// c2 should get the op
	broadcast := recvMsg(t, c2)
	if broadcast.Type != MsgOp {
		t.Fatalf("expected op, got %q", broadcast.Type)
	}
	if broadcast.Revision != 1 {
		t.Errorf("broadcast revision = %d, want 1", broadcast.Revision)
	}
	if broadcast.ClientID != "c1" {
		t.Errorf("broadcast clientId = %q, want %q", broadcast.ClientID, "c1")
	}

	// Verify document state
	if s.doc.Content != "Xabc" {
		t.Errorf("doc content = %q, want %q", s.doc.Content, "Xabc")
	}
}

func TestSession_ConcurrentOps(t *testing.T) {
	st := store.NewMemoryStore()
	st.Create(ctx(), "doc1", "abc")
	engine := &ot.JupiterEngine{}
	s := newSession("doc1", "abc", engine, st)
	go s.Run()
	defer close(s.stop)

	c1 := mockClient("c1")
	c2 := mockClient("c2")
	s.join <- c1
	s.join <- c2
	recvMsg(t, c1) // doc
	recvMsg(t, c2) // doc
	recvMsg(t, c1) // c2 join notification

	// Both at revision 0:
	// c1 inserts "X" at pos 0: "Xabc"
	// c2 inserts "Y" at pos 3: "abcY"
	s.incoming <- opMessage{
		client: c1,
		msg:    ClientMessage{Type: MsgOp, DocID: "doc1", Revision: 0, Op: ot.NewInsert(0, "X", 3)},
	}
	recvMsg(t, c1) // ack
	recvMsg(t, c2) // broadcast

	s.incoming <- opMessage{
		client: c2,
		msg:    ClientMessage{Type: MsgOp, DocID: "doc1", Revision: 0, Op: ot.NewInsert(3, "Y", 3)},
	}
	recvMsg(t, c2) // ack
	recvMsg(t, c1) // broadcast

	// After both ops, doc should be "XabcY"
	if s.doc.Content != "XabcY" {
		t.Errorf("doc content = %q, want %q", s.doc.Content, "XabcY")
	}
}

func TestSession_LeaveNotification(t *testing.T) {
	st := store.NewMemoryStore()
	st.Create(ctx(), "doc1", "")
	engine := &ot.JupiterEngine{}
	s := newSession("doc1", "", engine, st)
	go s.Run()
	defer close(s.stop)

	c1 := mockClient("c1")
	c2 := mockClient("c2")
	s.join <- c1
	s.join <- c2
	recvMsg(t, c1) // doc
	recvMsg(t, c2) // doc
	recvMsg(t, c1) // c2 join

	s.leave <- c2
	msg := recvMsg(t, c1)
	if msg.Type != MsgLeave {
		t.Fatalf("expected leave, got %q", msg.Type)
	}
	if msg.ClientID != "c2" {
		t.Errorf("leave clientId = %q, want %q", msg.ClientID, "c2")
	}
}
