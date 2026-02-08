package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/alimasry/go-collab-editor/ot"
	"github.com/alimasry/go-collab-editor/store"
)

func setupTestServer(t *testing.T) (*httptest.Server, *Hub) {
	t.Helper()
	st := store.NewMemoryStore()
	engine := &ot.JupiterEngine{}
	hub := NewHub(st, engine)
	go hub.Run()
	handler := NewHandler(hub)
	return httptest.NewServer(handler), hub
}

func wsConnect(t *testing.T, server *httptest.Server) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	return conn
}

func readWsMsg(t *testing.T, conn *websocket.Conn) ServerMessage {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var msg ServerMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return msg
}

func TestHandler_WebSocketConnect(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	conn := wsConnect(t, server)
	defer conn.Close()

	// Send join message
	msg := ClientMessage{Type: MsgJoin, DocID: "test-doc"}
	if err := conn.WriteJSON(msg); err != nil {
		t.Fatal(err)
	}

	// Read doc response
	resp := readWsMsg(t, conn)
	if resp.Type != MsgDoc {
		t.Errorf("expected doc, got %q", resp.Type)
	}
}

func TestHandler_TwoClientsCollaborate(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	conn1 := wsConnect(t, server)
	defer conn1.Close()
	conn2 := wsConnect(t, server)
	defer conn2.Close()

	// c1 joins
	conn1.WriteJSON(ClientMessage{Type: MsgJoin, DocID: "collab"})
	doc1 := readWsMsg(t, conn1) // doc
	if doc1.Type != MsgDoc {
		t.Fatalf("c1 expected doc, got %q", doc1.Type)
	}

	// c2 joins
	conn2.WriteJSON(ClientMessage{Type: MsgJoin, DocID: "collab"})
	doc2 := readWsMsg(t, conn2) // doc
	if doc2.Type != MsgDoc {
		t.Fatalf("c2 expected doc, got %q", doc2.Type)
	}

	// c1 gets join notification for c2
	joinNotif := readWsMsg(t, conn1)
	if joinNotif.Type != MsgJoin {
		t.Fatalf("c1 expected join notification, got %q", joinNotif.Type)
	}

	// c1 sends an insert
	op := ot.NewInsert(0, "hello", 0)
	conn1.WriteJSON(ClientMessage{Type: MsgOp, DocID: "collab", Revision: 0, Op: op})

	// c1 gets ack
	ack := readWsMsg(t, conn1)
	if ack.Type != MsgAck {
		t.Fatalf("expected ack, got %q", ack.Type)
	}

	// c2 gets the broadcast op
	broadcast := readWsMsg(t, conn2)
	if broadcast.Type != MsgOp {
		t.Fatalf("expected op broadcast, got %q", broadcast.Type)
	}
}
