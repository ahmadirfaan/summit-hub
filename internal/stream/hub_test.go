package stream

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestHubBroadcast(t *testing.T) {
	hub := NewHub(nil)
	client := hub.Register("session-1")
	defer hub.Unregister(client)

	payload := []byte("hello")
	hub.Broadcast("session-1", payload)

	select {
	case msg := <-client.Send:
		if string(msg) != "hello" {
			t.Fatalf("unexpected message")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout waiting for message")
	}
}

func TestHubHelpers(t *testing.T) {
	ch := redisChannel("abc")
	if ch == "" {
		t.Fatalf("expected channel")
	}
	if sessionIDFromChannel(ch) != "abc" {
		t.Fatalf("unexpected session id")
	}
	if sessionIDFromChannel("bad") != "" {
		t.Fatalf("expected empty session id")
	}
}

func TestUnregisterCloses(t *testing.T) {
	hub := NewHub(nil)
	client := hub.Register("session-2")
	hub.Unregister(client)
	_, ok := <-client.Send
	if ok {
		t.Fatalf("expected channel closed")
	}
}

func TestHubRedisBroadcastAndSubscribe(t *testing.T) {
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer client.Close()

	hub := NewHub(client)
	ws := hub.Register("session-redis")
	defer hub.Unregister(ws)

	hub.Broadcast("session-redis", []byte("ping"))

	select {
	case msg := <-ws.Send:
		if string(msg) != "ping" {
			t.Fatalf("unexpected message")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timeout waiting for broadcast")
	}

	// ensure subscribeRedis forwards redis publish (subscribe uses literal channel string)
	starClient := hub.Register("*")
	defer hub.Unregister(starClient)

	time.Sleep(20 * time.Millisecond)
	if err := client.Publish(context.Background(), "tracking:*:broadcast", "pong").Err(); err != nil {
		t.Fatalf("publish error: %v", err)
	}

	select {
	case msg := <-starClient.Send:
		if string(msg) != "pong" {
			t.Fatalf("unexpected message from redis")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timeout waiting for redis message")
	}
}

func TestHubRedisPublishError(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	server.Close()
	defer client.Close()

	hub := NewHub(client)
	clientNode := hub.Register("session-bad")
	defer hub.Unregister(clientNode)

	hub.Broadcast("session-bad", []byte("ping"))
}
