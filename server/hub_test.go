package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alimasry/go-collab-editor/ot"
	"github.com/alimasry/go-collab-editor/store"
)

func TestHub_CreateSessionOnJoin(t *testing.T) {
	st := store.NewMemoryStore()
	engine := &ot.JupiterEngine{}
	hub := NewHub(st, engine)
	go hub.Run()

	c := mockClient("c1")
	c.hub = hub
	hub.joinDoc <- joinRequest{client: c, docID: "new-doc"}

	// Wait a bit for the async join to be processed
	time.Sleep(100 * time.Millisecond)

	// Client should receive a doc message
	select {
	case data := <-c.send:
		var msg ServerMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatal(err)
		}
		if msg.Type != MsgDoc {
			t.Errorf("expected doc, got %q", msg.Type)
		}
		if msg.DocID != "new-doc" {
			t.Errorf("docId = %q, want %q", msg.DocID, "new-doc")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}

	// Session should exist
	s := hub.GetSession("new-doc")
	if s == nil {
		t.Error("session not created")
	}
}

func TestHub_JoinExistingDoc(t *testing.T) {
	st := store.NewMemoryStore()
	st.Create(ctx(), "existing", "hello world")
	engine := &ot.JupiterEngine{}
	hub := NewHub(st, engine)
	go hub.Run()

	c := mockClient("c1")
	c.hub = hub
	hub.joinDoc <- joinRequest{client: c, docID: "existing"}

	time.Sleep(100 * time.Millisecond)

	select {
	case data := <-c.send:
		var msg ServerMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatal(err)
		}
		if msg.Content != "hello world" {
			t.Errorf("content = %q, want %q", msg.Content, "hello world")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}
