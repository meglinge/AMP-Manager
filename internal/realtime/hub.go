package realtime

import (
	"encoding/json"
	"sync"

	log "github.com/sirupsen/logrus"
)

// LogFetcher fetches a single request log for broadcasting
type LogFetcher func(id string) (interface{}, error)

// Hub manages WebSocket clients and broadcasts log events
type Hub struct {
	clients    map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	notify     chan string
	fetcher    LogFetcher
	mu         sync.RWMutex
}

var (
	globalHub *Hub
	hubOnce   sync.Once
)

// InitHub initializes the global hub
func InitHub(fetcher LogFetcher) {
	hubOnce.Do(func() {
		globalHub = &Hub{
			clients:    make(map[*Client]struct{}),
			register:   make(chan *Client, 16),
			unregister: make(chan *Client, 16),
			notify:     make(chan string, 256),
			fetcher:    fetcher,
		}
		go globalHub.run()
		log.Info("realtime: hub initialized")
	})
}

// GetHub returns the global hub
func GetHub() *Hub {
	return globalHub
}

// NotifyLogCompleted notifies the hub a log was completed (non-blocking)
func NotifyLogCompleted(id string) {
	h := globalHub
	if h == nil {
		return
	}
	select {
	case h.notify <- id:
	default:
		// hub busy, drop notification
	}
}

// Register adds a client to the hub
func (h *Hub) Register(c *Client) {
	h.register <- c
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = struct{}{}
			h.mu.Unlock()
			log.Debugf("realtime: client registered, total=%d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Debugf("realtime: client unregistered, total=%d", len(h.clients))

		case id := <-h.notify:
			h.mu.RLock()
			count := len(h.clients)
			h.mu.RUnlock()
			if count == 0 {
				continue
			}

			logEntry, err := h.fetcher(id)
			if err != nil || logEntry == nil {
				log.Debugf("realtime: failed to fetch log %s for broadcast: %v", id, err)
				continue
			}

			msg, err := json.Marshal(map[string]interface{}{
				"type": "request_log_completed",
				"data": logEntry,
			})
			if err != nil {
				continue
			}

			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- msg:
				default:
					// slow client, skip
				}
			}
			h.mu.RUnlock()
		}
	}
}
