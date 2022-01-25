package server

import (
	"time"
)

type Hub interface {
	Run()
	Register() *chan *Client
	Unregister() *chan *Client
	Message() *chan Msg
}

// Hub maintains the set of active clients and messages to the
// clients.
type hub struct {
	// Registered clients.
	clients map[string]*Client

	// Inbound messages from the clients.
	// message chan []byte
	message chan Msg

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

type Msg struct {
	Type      int // 0 = broadcast, 1 = private, 2 = group
	Text      string
	SenderID  string
	Receiver  string
	TimeStamp time.Time
}

func NewHub() Hub {
	return &hub{
		message:    make(chan Msg),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[string]*Client),
	}
}

func NewMsg(msgType int, text []byte, senderID string, receiver string) *Msg {
	return &Msg{
		Type:      msgType, // 0 = broadcast, 1 = private, 2 = group
		Text:      string(text),
		SenderID:  senderID,
		Receiver:  receiver,
		TimeStamp: time.Now(),
	}
}

func (h *hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client.ID] = client
		case client := <-h.unregister:
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)
			}
		case message := <-h.message:
			switch message.Type {
			case 0:
				//#print log
				// fmt.Println("server -> hub -> Run -> msg type = 0: ", message.SenderID, " to ", message.Receiver, ": ", message.Text)

				for client := range h.clients {
					select {
					case h.clients[client].Send <- []byte(message.Text):
					default:
						close(h.clients[client].Send)
						delete(h.clients, client)
					}
				}
			case 1:
				//#print log
				// fmt.Println("hub.server -> hub -> Run -> msg type = 1: ", message.SenderID, " to ", message.Receiver)
				// fmt.Println("hub.[]byte(message.Text)= ", []byte(message.Text))
				// fmt.Println("hub.clients[message.Receiver]= ", h.clients[message.Receiver])

				h.clients[message.Receiver].Send <- []byte(message.Text)
			default:
				//#print log
				// fmt.Println("server -> hub -> Run -> msg type = other : ", message.SenderID, " to ", message.Receiver, ": ", message.Text)

				for client := range h.clients { //#placeholder
					select {
					case h.clients[client].Send <- []byte(message.Text):
					default:
						close(h.clients[client].Send)
						delete(h.clients, client)
					}
				}
			}
		}
	}
}
func (h *hub) Register() *chan *Client {
	return &h.register
}
func (h *hub) Unregister() *chan *Client {
	return &h.unregister
}

func (h *hub) Message() *chan Msg {
	return &h.message
}
