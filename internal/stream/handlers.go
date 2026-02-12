package stream

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func RegisterRoutes(r fiber.Router, hub *Hub) {
	r.Get("/ws/:sessionID", websocket.New(func(c *websocket.Conn) {
		sessionID := c.Params("sessionID")
		client := hub.Register(sessionID)
		defer hub.Unregister(client)

		done := make(chan struct{})
		go func() {
			for msg := range client.Send {
				if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
					break
				}
			}
			close(done)
		}()

		for {
			if _, _, err := c.ReadMessage(); err != nil {
				break
			}
		}
		<-done
	}))
}
