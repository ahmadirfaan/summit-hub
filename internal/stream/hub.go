package stream

import (
	"context"
	"log"
	"sync"

	"github.com/redis/go-redis/v9"
)

type Hub struct {
	redis   *redis.Client
	clients map[string]map[*Client]struct{}
	mu      sync.RWMutex
}

type Client struct {
	SessionID string
	Send      chan []byte
}

func NewHub(redisClient *redis.Client) *Hub {
	h := &Hub{
		redis:   redisClient,
		clients: map[string]map[*Client]struct{}{},
	}

	if redisClient != nil {
		go h.subscribeRedis()
	}
	return h
}

func (h *Hub) Register(sessionID string) *Client {
	client := &Client{
		SessionID: sessionID,
		Send:      make(chan []byte, 64),
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[sessionID] == nil {
		h.clients[sessionID] = map[*Client]struct{}{}
	}
	h.clients[sessionID][client] = struct{}{}
	return client
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if sessionClients, ok := h.clients[client.SessionID]; ok {
		delete(sessionClients, client)
		if len(sessionClients) == 0 {
			delete(h.clients, client.SessionID)
		}
	}
	close(client.Send)
}

func (h *Hub) Broadcast(sessionID string, payload []byte) {
	h.mu.RLock()
	clients := h.clients[sessionID]
	h.mu.RUnlock()

	for client := range clients {
		select {
		case client.Send <- payload:
		default:
		}
	}

	if h.redis != nil {
		err := h.redis.Publish(context.Background(), redisChannel(sessionID), payload).Err()
		if err != nil {
			log.Printf("redis publish error: %v", err)
		}
	}
}

func (h *Hub) subscribeRedis() {
	ctx := context.Background()
	pubsub := h.redis.Subscribe(ctx, "tracking:*:broadcast")
	defer pubsub.Close()

	for msg := range pubsub.Channel() {
		sessionID := sessionIDFromChannel(msg.Channel)
		h.mu.RLock()
		clients := h.clients[sessionID]
		h.mu.RUnlock()
		for client := range clients {
			select {
			case client.Send <- []byte(msg.Payload):
			default:
			}
		}
	}
}

func redisChannel(sessionID string) string {
	return "tracking:" + sessionID + ":broadcast"
}

func sessionIDFromChannel(ch string) string {
	// tracking:{session}:broadcast
	const prefix = "tracking:"
	const suffix = ":broadcast"
	if len(ch) <= len(prefix)+len(suffix) {
		return ""
	}
	return ch[len(prefix) : len(ch)-len(suffix)]
}
