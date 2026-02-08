package server

import (
	"encoding/json"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMsgSize = 64 * 1024
)

// Client represents a single WebSocket connection.
type Client struct {
	ID    string
	Name  string
	Color string

	hub  *Hub
	conn *websocket.Conn
	send chan []byte

	// The session this client is currently in (nil if not joined).
	mu      sync.Mutex
	session *Session
}

var (
	adjectives = []string{"Red", "Blue", "Green", "Gold", "Silver", "Purple", "Orange", "Teal", "Coral", "Jade"}
	animals    = []string{"Fox", "Owl", "Bear", "Wolf", "Hawk", "Deer", "Lynx", "Crow", "Dove", "Seal"}
	colors     = []string{"#e74c3c", "#3498db", "#2ecc71", "#f39c12", "#9b59b6", "#1abc9c", "#e67e22", "#00bcd4", "#ff5722", "#8bc34a"}
)

func newClient(hub *Hub, conn *websocket.Conn) *Client {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &Client{
		ID:    generateID(),
		Name:  adjectives[r.Intn(len(adjectives))] + " " + animals[r.Intn(len(animals))],
		Color: colors[r.Intn(len(colors))],
		hub:   hub,
		conn:  conn,
		send:  make(chan []byte, 256),
	}
}

func generateID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[r.Intn(len(chars))]
	}
	return string(b)
}

// ReadPump reads messages from the WebSocket and routes them.
func (c *Client) ReadPump() {
	defer func() {
		c.mu.Lock()
		s := c.session
		c.mu.Unlock()
		if s != nil {
			s.leave <- c
		}
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMsgSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("client %s read error: %v", c.ID, err)
			}
			return
		}

		var msg ClientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			c.sendError("invalid message format")
			continue
		}

		switch msg.Type {
		case MsgJoin:
			c.hub.joinDoc <- joinRequest{client: c, docID: msg.DocID}
		case MsgOp:
			c.mu.Lock()
			s := c.session
			c.mu.Unlock()
			if s == nil {
				c.sendError("not joined to a document")
				continue
			}
			s.incoming <- opMessage{client: c, msg: msg}
		default:
			c.sendError("unknown message type: " + msg.Type)
		}
	}
}

// WritePump writes messages from the send channel to the WebSocket.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case data, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) sendMsg(msg ServerMessage) {
	select {
	case c.send <- msg.Encode():
	default:
		// Client too slow, drop message.
	}
}

func (c *Client) sendError(message string) {
	c.sendMsg(ServerMessage{Type: MsgError, Message: message})
}

func (c *Client) Info() ClientInfo {
	return ClientInfo{ID: c.ID, Name: c.Name, Color: c.Color}
}
