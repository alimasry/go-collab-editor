package server

import (
	"context"
	"log"

	"github.com/alimasry/go-collab-editor/ot"
	"github.com/alimasry/go-collab-editor/store"
)

type opMessage struct {
	client *Client
	msg    ClientMessage
}

// Session manages collaboration for a single document.
// All operations are serialized through a single goroutine.
type Session struct {
	docID   string
	doc     *ot.Document
	engine  ot.Engine
	store   store.DocumentStore
	clients map[*Client]bool

	incoming chan opMessage
	join     chan *Client
	leave    chan *Client
	stop     chan struct{}
}

func newSession(docID, content string, version int, history []ot.Operation, engine ot.Engine, st store.DocumentStore) *Session {
	doc := ot.NewDocument(content)
	doc.Version = version
	doc.History = history
	return &Session{
		docID:    docID,
		doc:      doc,
		engine:   engine,
		store:    st,
		clients:  make(map[*Client]bool),
		incoming: make(chan opMessage, 64),
		join:     make(chan *Client, 16),
		leave:    make(chan *Client, 16),
		stop:     make(chan struct{}),
	}
}

// Run is the session's main loop. It serializes all operations.
func (s *Session) Run() {
	for {
		select {
		case c := <-s.join:
			s.handleJoin(c)
		case c := <-s.leave:
			s.handleLeave(c)
		case om := <-s.incoming:
			s.handleOp(om)
		case <-s.stop:
			return
		}
	}
}

func (s *Session) handleJoin(c *Client) {
	s.clients[c] = true
	c.mu.Lock()
	c.session = s
	c.mu.Unlock()

	// Send current document state to the joining client.
	clients := s.clientInfos()
	c.sendMsg(ServerMessage{
		Type:     MsgDoc,
		DocID:    s.docID,
		Content:  s.doc.Content,
		Revision: s.doc.Version,
		Clients:  clients,
	})

	// Notify other clients about the new user.
	for other := range s.clients {
		if other != c {
			other.sendMsg(ServerMessage{
				Type:     MsgJoin,
				ClientID: c.ID,
				Name:     c.Name,
				Color:    c.Color,
			})
		}
	}
}

func (s *Session) handleLeave(c *Client) {
	if _, ok := s.clients[c]; !ok {
		return
	}
	delete(s.clients, c)
	c.mu.Lock()
	c.session = nil
	c.mu.Unlock()
	close(c.send)

	// Notify others.
	for other := range s.clients {
		other.sendMsg(ServerMessage{
			Type:     MsgLeave,
			ClientID: c.ID,
		})
	}
}

func (s *Session) handleOp(om opMessage) {
	// Transform the client's operation against server history.
	transformed, err := s.engine.TransformIncoming(om.msg.Op, om.msg.Revision, s.doc.History)
	if err != nil {
		log.Printf("session %s: transform error: %v", s.docID, err)
		om.client.sendError("transform error: " + err.Error())
		return
	}

	// Apply to the document.
	if err := s.doc.Apply(transformed); err != nil {
		log.Printf("session %s: apply error: %v", s.docID, err)
		om.client.sendError("apply error: " + err.Error())
		return
	}

	// Persist.
	ctx := context.Background()
	s.store.UpdateContent(ctx, s.docID, s.doc.Content, s.doc.Version)
	s.store.AppendOperation(ctx, s.docID, transformed, s.doc.Version)

	// Ack the sender.
	om.client.sendMsg(ServerMessage{
		Type:     MsgAck,
		Revision: s.doc.Version,
	})

	// Broadcast to other clients.
	for c := range s.clients {
		if c != om.client {
			c.sendMsg(ServerMessage{
				Type:     MsgOp,
				DocID:    s.docID,
				Revision: s.doc.Version,
				Op:       transformed,
				ClientID: om.client.ID,
			})
		}
	}
}

func (s *Session) clientInfos() []ClientInfo {
	infos := make([]ClientInfo, 0, len(s.clients))
	for c := range s.clients {
		infos = append(infos, c.Info())
	}
	return infos
}
