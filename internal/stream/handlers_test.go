package stream

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gorilla/websocket"
)

func TestStreamHandlersUpgradeRequired(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/stream"), NewHub(nil))

	req := httptest.NewRequest(http.MethodGet, "/stream/ws/session-1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected non-200 for non-websocket request")
	}
}

func TestStreamHandlersWebsocketBroadcast(t *testing.T) {
	hub := NewHub(nil)
	app := fiber.New()
	RegisterRoutes(app.Group("/stream"), hub)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	defer ln.Close()

	go func() {
		_ = app.Listener(ln)
	}()
	defer func() { _ = app.Shutdown() }()

	wsURL := "ws://" + ln.Addr().String() + "/stream/ws/session-1"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	hub.Broadcast("session-1", []byte("hello"))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(msg) != "hello" {
		t.Fatalf("unexpected message")
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte("client")); err != nil {
		t.Fatalf("write error: %v", err)
	}

	conn.Close()
	hub.Broadcast("session-1", []byte("bye"))
	_ = conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
}

func TestStreamHandlersWebsocketWriteError(t *testing.T) {
	hub := NewHub(nil)
	app := fiber.New()
	RegisterRoutes(app.Group("/stream"), hub)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	defer ln.Close()

	go func() {
		_ = app.Listener(ln)
	}()
	defer func() { _ = app.Shutdown() }()

	wsURL := "ws://" + ln.Addr().String() + "/stream/ws/session-2"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	conn.Close()

	hub.Broadcast("session-2", []byte("ping"))
	time.Sleep(20 * time.Millisecond)
}

func TestStreamHandlersWebsocketCloseMessage(t *testing.T) {
	hub := NewHub(nil)
	app := fiber.New()
	RegisterRoutes(app.Group("/stream"), hub)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	defer ln.Close()

	go func() {
		_ = app.Listener(ln)
	}()
	defer func() { _ = app.Shutdown() }()

	wsURL := "ws://" + ln.Addr().String() + "/stream/ws/session-3"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}

	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
	conn.Close()

	hub.Broadcast("session-3", []byte("ping"))
	time.Sleep(20 * time.Millisecond)
}
