package stream

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	fws "github.com/gofiber/websocket/v2"
	gws "github.com/gorilla/websocket"
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
	conn, _, err := gws.DefaultDialer.Dial(wsURL, nil)
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

	if err := conn.WriteMessage(gws.TextMessage, []byte("client")); err != nil {
		t.Fatalf("write error: %v", err)
	}

	conn.Close()
	hub.Broadcast("session-1", []byte("bye"))
	_ = conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
}

func TestStreamHandlersCloseHook(t *testing.T) {
	closed := make(chan struct{})
	oldHook := onStreamClosed
	onStreamClosed = func(_ string) { close(closed) }
	defer func() { onStreamClosed = oldHook }()

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

	wsURL := "ws://" + ln.Addr().String() + "/stream/ws/session-hook"
	conn, _, err := gws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	conn.Close()

	select {
	case <-closed:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected close hook")
	}
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
	conn, _, err := gws.DefaultDialer.Dial(wsURL, nil)
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
	conn, _, err := gws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}

	_ = conn.WriteMessage(gws.CloseMessage, gws.FormatCloseMessage(gws.CloseNormalClosure, "bye"))
	conn.Close()

	hub.Broadcast("session-3", []byte("ping"))
	time.Sleep(20 * time.Millisecond)
}

func TestStreamHandlersWriteAndReadErrorHooks(t *testing.T) {
	oldWrite := writeMessageFn
	oldRead := readMessageFn
	oldHook := onStreamClosed
	writeCalled := make(chan struct{})
	readRelease := make(chan struct{})
	writeMessageFn = func(_ *fws.Conn, _ []byte) error {
		close(writeCalled)
		return fiber.ErrInternalServerError
	}
	readMessageFn = func(_ *fws.Conn) error {
		<-readRelease
		return fiber.ErrInternalServerError
	}
	closed := make(chan struct{})
	onStreamClosed = func(_ string) { close(closed) }
	defer func() {
		writeMessageFn = oldWrite
		readMessageFn = oldRead
		onStreamClosed = oldHook
	}()

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

	wsURL := "ws://" + ln.Addr().String() + "/stream/ws/session-err"
	conn, _, err := gws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	hub.Broadcast("session-err", []byte("ping"))
	select {
	case <-writeCalled:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected write error")
	}
	close(readRelease)
	select {
	case <-closed:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected close hook")
	}
}

func TestStreamHandlersReadLoopSuccessPath(t *testing.T) {
	oldRead := readMessageFn
	oldWrite := writeMessageFn
	count := 0
	readMessageFn = func(_ *fws.Conn) error {
		count++
		if count == 1 {
			return nil
		}
		return fiber.ErrInternalServerError
	}
	writeMessageFn = func(_ *fws.Conn, _ []byte) error { return nil }
	defer func() {
		readMessageFn = oldRead
		writeMessageFn = oldWrite
	}()

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

	wsURL := "ws://" + ln.Addr().String() + "/stream/ws/session-loop"
	conn, _, err := gws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	conn.Close()

	time.Sleep(20 * time.Millisecond)
}

func TestStreamHandlersFullPath(t *testing.T) {
	oldRead := readMessageFn
	oldWrite := writeMessageFn
	oldHook := onStreamClosed
	readCount := 0
	readMessageFn = func(_ *fws.Conn) error {
		readCount++
		if readCount == 1 {
			return nil
		}
		return fiber.ErrInternalServerError
	}
	writeMessageFn = func(_ *fws.Conn, _ []byte) error { return nil }
	closed := make(chan struct{})
	onStreamClosed = func(_ string) { close(closed) }
	defer func() {
		readMessageFn = oldRead
		writeMessageFn = oldWrite
		onStreamClosed = oldHook
	}()

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

	wsURL := "ws://" + ln.Addr().String() + "/stream/ws/session-full"
	conn, _, err := gws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	hub.Broadcast("session-full", []byte("ping"))
	conn.Close()

	select {
	case <-closed:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected close hook")
	}
}
