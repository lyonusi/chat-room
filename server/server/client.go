package server

import (
	"encoding/json"
	"fmt"
	"log"

	"net/http"
	"time"

	"github.com/google/uuid"
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
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	Hub *Hub

	// The websocket connection.
	Conn *websocket.Conn

	// Buffered channel of outbound messages.
	Send chan []byte

	ID string
}

// type Client interface {
// 	ReadPump()
// 	WritePump()
// 	Register(client *Client)
// 	Unregister(client *Client)
// 	SendMessageToHub([]byte)
// 	ID() string
// 	Send() *chan []byte
// }

func NewClient(hub *Hub, conn *websocket.Conn, send chan []byte, id string) *Client {
	return &Client{
		Hub:  hub,
		Conn: conn,
		Send: send,
		ID:   id,
	}
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.

// serveWs handles websocket requests from the peer.
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	// send := make(chan []byte, 256)
	id := uuid.New().String()

	//print log
	fmt.Println("client.ServeWs.id = ", id)

	// client := NewClient(hub, conn, send, id)
	// client.Register(&client)

	client := &Client{Hub: hub, Conn: conn, Send: make(chan []byte, 256), ID: id}

	clientHub := *client.Hub
	registerChan := *clientHub.Register()
	registerChan <- client

	//print log
	fmt.Println("client.ServeWs.client = ", client)
	fmt.Println("client.ServeWs.client.hub = ", *client.Hub)

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.WritePump()
	go client.ReadPump()
}

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

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
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
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.Send)
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

// func (c *Client) ID() string {
// 	return c.ID
//

// func (c *Client) Send() *chan []byte {
// 	return &c.Send
// }
