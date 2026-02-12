package stream

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func RegisterRoutes(r fiber.Router, hub *Hub) {
	r.Get("/ws/:sessionID", websocket.New(func(c *websocket.Conn) {
		sessionID := c.Params("sessionID")
		client := hub.Register(sessionID)

		done := make(chan struct{})
		go func() {
			for msg := range client.Send {
				if err := writeMessageFn(c, msg); err != nil {
					break
				}
			}
			close(done)
		}()

		for {
			if err := readMessageFn(c); err != nil {
				break
			}
		}
		hub.Unregister(client)
		<-done
		onStreamClosed(sessionID)
	}))
}

var writeMessageFn = func(c *websocket.Conn, msg []byte) error {
	return c.WriteMessage(websocket.TextMessage, msg)
}

var readMessageFn = func(c *websocket.Conn) error {
	_, _, err := c.ReadMessage()
	return err
}

var onStreamClosed = func(string) {}
