package ws

import (
	"log"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stratumfarm/phantomias/recws"
)

type Client struct {
	url       string
	ws        recws.RecConn
	broadcast chan []byte
}

func New(url string, broadcast chan []byte) *Client {
	return &Client{
		url:       url,
		broadcast: broadcast,
	}
}

func (c *Client) Close() {
	c.ws.Close(true)
}

func (c *Client) Listen(doneC <-chan os.Signal) error {
	defer func() { recover() }()

	ws, err := recws.New(
		c.url, nil,
		recws.WithKeepAliveTimeout(time.Second*15),
		recws.WithDebugLogFn(func(s string) {
			log.Printf("[recws][debug] %s\n", s)
		}),
		recws.WithErrorLogFn(func(err error, s string) {
			log.Printf("[recws][err] %s: %s\n", s, err)
		}),
	)
	if err != nil {
		return err
	}
	c.ws = ws
	if err := c.ws.Dial(); err != nil {
		return err
	}
	c.relayMessages(doneC)
	return nil
}

func (c *Client) relayMessages(doneC <-chan os.Signal) {
	for {
		select {
		case <-doneC:
			return
		default:
			if !c.ws.IsConnected() {
				continue
			}
			mtype, message, err := c.ws.ReadMessage()
			if err != nil {
				log.Printf("[wsclient][err] failed to read message: %s", err)
				continue
			}
			if mtype != websocket.TextMessage {
				log.Printf("[wsclient][warn] wont relay non-text message: %d", mtype)
				continue
			}
			c.broadcast <- message
		}
	}
}
