package server

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Hub interface {
	Run()
	Register() *chan *Client
	Unregister() *chan *Client
	Message() *chan Msg
	CreateGroup(clients []string) string
	DeleteGroup(groupID string, clientID string) error
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

	groups map[string][]string
}

type Msg struct {
	Type      int // 0 = broadcast, 1 = private, 2 = group
	Text      string
	SenderID  string
	Receiver  string
	TimeStamp time.Time
}

func NewHub() Hub {
	fmt.Println("hub.NewHub")
	return &hub{
		message:    make(chan Msg),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[string]*Client),
		groups:     make(map[string][]string),
		// groups: groupHC,
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
					case h.clients[client].Send <- message:
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
				if h.clients[message.Receiver] == nil {
					fmt.Println("hub.Run: h.client[message.Receiver].Send: receiver is nil")
					message.Text = fmt.Sprintf("failed to send message: %v\nerror:  invalid receiver", message)
					h.clients[message.SenderID].Send <- message
				} else {
					h.clients[message.Receiver].Send <- message
				}
			case 2:
				groupID := message.Receiver
				if h.groups[groupID] == nil {
					fmt.Printf("hub.Run: h.groups[%s]: receiver is nil\n", groupID)
					message.Text = fmt.Sprintf("failed to send message: %v\nerror:  invalid receiver\n", message)
					h.clients[message.SenderID].Send <- message
				} else {
					//#print log
					// fmt.Println("hub.Run.msg type = 2 <group>: ", message.SenderID, " to GROUP ", groupID)
					// fmt.Println("hub.Run.msg type = 2 <group>: h.groups[groupID] = ", h.groups[groupID])

					for _, clientID := range h.groups[groupID] {
						select {
						case h.clients[clientID].Send <- message:
						default:
							close(h.clients[clientID].Send)
							delete(h.clients, clientID)
						}
					}
				}
			default:
				//#print log
				// fmt.Println("server -> hub -> Run -> msg type = other : ", message.SenderID, " to ", message.Receiver, ": ", message.Text)

				for client := range h.clients { //#placeholder
					select {
					case h.clients[client].Send <- message:
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

func (h *hub) CreateGroup(clients []string) string {
	groupID := uuid.New().String()
	h.groups[groupID] = clients

	//#print log
	fmt.Printf("server.hub.CreatedGroup: h.groups[%s] = %s \n", groupID, h.groups[groupID])
	return groupID
}

func (h *hub) DeleteGroup(groupID string, clientID string) error {
	//#print log
	fmt.Printf("server.hub.CreatedGroup: h.groups[%s] = %s \n", groupID, h.groups[groupID])

	_, ok := h.groups[groupID]
	if ok {
		delete(h.groups, groupID)
		return nil
	}

	return fmt.Errorf("error: server.hub.DeleteGroup: h.groups[%s] = nil", groupID)
}
