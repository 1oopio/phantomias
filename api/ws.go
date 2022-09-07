package api

import (
	"context"
	"log"
	"sync"

	"github.com/gofiber/websocket/v2"
)

type wsClient struct {
	ip        string
	isClosing bool
	mu        sync.Mutex
}

type wsRelay struct {
	ctx        context.Context
	clients    map[*websocket.Conn]*wsClient
	register   chan *websocket.Conn
	broadcast  chan []byte
	unregister chan *websocket.Conn
}

func newWSRelay(ctx context.Context) *wsRelay {
	return &wsRelay{
		ctx:        ctx,
		clients:    make(map[*websocket.Conn]*wsClient),
		register:   make(chan *websocket.Conn),
		broadcast:  make(chan []byte),
		unregister: make(chan *websocket.Conn),
	}
}

func (w *wsRelay) hub() {
	for {
		select {
		case conn := <-w.register:
			ip := conn.RemoteAddr().String()
			w.clients[conn] = &wsClient{
				ip: ip,
			}
			log.Printf("client registered with IP: %s", ip)

		case msg := <-w.broadcast:
			for conn, client := range w.clients {
				go func(conn *websocket.Conn, client *wsClient) { // send to each client in parallel so we don't block on a slow client
					client.mu.Lock()
					defer client.mu.Unlock()
					if client.isClosing {
						return
					}

					if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
						client.isClosing = true
						log.Println("write error:", err)
						conn.WriteMessage(websocket.CloseMessage, []byte{})
						conn.Close()
						w.unregister <- conn
					}
				}(conn, client)
			}

		case conn := <-w.unregister:
			if c, ok := w.clients[conn]; ok {
				delete(w.clients, conn)
				log.Printf("client unregistered with IP: %s", c.ip)
			}

		case <-w.ctx.Done():
			return
		}
	}
}

func (s *Server) wsHandler(c *websocket.Conn) {
	defer func() {
		s.wsRelay.unregister <- c
		c.Close()
	}()

	s.wsRelay.register <- c

	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure) {
				log.Println("read error:", err)
			}
			return
		}
	}
}
