package server

import (
	"encoding/json"
	"log"

	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	// space   = []byte{' '}
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	Hub *Hub

	// The websocket connection.
	Conn *websocket.Conn

	// Buffered channel of outbound messages.
	Send chan Msg

	ID string
}

func NewClient(hub *Hub, conn *websocket.Conn, send chan Msg, id string) *Client {
	return &Client{
		Hub:  hub,
		Conn: conn,
		Send: send,
		ID:   id,
	}
}

// serveWs handles websocket requests from the peer.

func (c *Client) ReadPump() {
	defer func() {
		client := NewClient(c.Hub, c.Conn, c.Send, c.ID)
		c.Unregister(client)
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		// _, message, err := c.Conn.ReadMessage() //original template
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		//#print log
		// fmt.Println("client.ReadPump: message = ", message)

		var msgJSON Msg

		// msgByte := bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		// msgByte := bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		///#print log
		// fmt.Println("client.ReadPump: msg in []Byte = ", message)

		err = json.Unmarshal(message, &msgJSON)
		if err != nil {
			log.Printf("client.ReadPump.JsonUnmarshal: error: %v", err)
		}

		//#print log
		// fmt.Println("client.ReadPump: msgJSON = ", msgJSON)

		msg := *NewMsg(
			msgJSON.Type,
			[]byte(msgJSON.Text),
			c.ID,
			msgJSON.Receiver)
		//#print log
		// fmt.Println("client.ReadPump: mgs=", msg)
		c.SendMessageToHub(msg)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write([]byte(message.Text))

			// Add queued chat messages to the current websocket message.
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				b := <-c.Send
				w.Write([]byte(b.Text))
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) Register(client *Client) {
	hub := *c.Hub
	registerChan := *hub.Register()
	registerChan <- client
}

func (c *Client) Unregister(client *Client) {
	hub := *c.Hub
	unregisterChan := *hub.Unregister()
	unregisterChan <- client
}

func (c *Client) SendMessageToHub(msg Msg) {
	hub := *c.Hub
	messageChan := *hub.Message()
	messageChan <- msg
}
