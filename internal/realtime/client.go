package realtime

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"nhooyr.io/websocket"
)

// Client represents a WebSocket connection
type Client struct {
	conn *websocket.Conn
	send chan []byte
}

// NewClient creates a new WebSocket client
func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		conn: conn,
		send: make(chan []byte, 64),
	}
}

// WriteLoop pumps messages from the hub to the WebSocket connection
func (c *Client) WriteLoop(ctx context.Context) {
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := c.conn.Write(writeCtx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				log.Debugf("realtime: write error: %v", err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// ReadLoop reads messages from the WebSocket (discards them, detects close)
func (c *Client) ReadLoop(ctx context.Context) {
	for {
		_, _, err := c.conn.Read(ctx)
		if err != nil {
			return
		}
	}
}

// Close closes the WebSocket connection
func (c *Client) Close() {
	c.conn.Close(websocket.StatusNormalClosure, "")
}
