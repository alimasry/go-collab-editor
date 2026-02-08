package server

import (
	"context"
	"log"
	"sync"

	"github.com/alimasry/go-collab-editor/ot"
	"github.com/alimasry/go-collab-editor/store"
)

type joinRequest struct {
	client *Client
	docID  string
}

// Hub manages document sessions and routes clients to the right session.
type Hub struct {
	store    store.DocumentStore
	engine   ot.Engine
	sessions map[string]*Session
	mu       sync.RWMutex

	joinDoc chan joinRequest
}

func NewHub(st store.DocumentStore, engine ot.Engine) *Hub {
	return &Hub{
		store:    st,
		engine:   engine,
		sessions: make(map[string]*Session),
		joinDoc:  make(chan joinRequest, 64),
	}
}

// Run is the hub's main loop.
func (h *Hub) Run() {
	for req := range h.joinDoc {
		h.handleJoinDoc(req)
	}
}

func (h *Hub) handleJoinDoc(req joinRequest) {
	h.mu.Lock()
	s, ok := h.sessions[req.docID]
	if !ok {
		// Create document in store if it doesn't exist.
		ctx := context.Background()
		if _, err := h.store.Get(ctx, req.docID); err != nil {
			if err := h.store.Create(ctx, req.docID, ""); err != nil {
				log.Printf("hub: failed to create doc %q: %v", req.docID, err)
				h.mu.Unlock()
				req.client.sendError("failed to create document")
				return
			}
		}

		info, err := h.store.Get(ctx, req.docID)
		if err != nil {
			log.Printf("hub: failed to get doc %q: %v", req.docID, err)
			h.mu.Unlock()
			req.client.sendError("failed to load document")
			return
		}

		s = newSession(req.docID, info.Content, h.engine, h.store)
		h.sessions[req.docID] = s
		go s.Run()
	}
	h.mu.Unlock()

	s.join <- req.client
}

// GetSession returns the session for a document, if active.
func (h *Hub) GetSession(docID string) *Session {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.sessions[docID]
}
