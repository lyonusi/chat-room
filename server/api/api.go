package api

import (
	"chatroom/server"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

type Api interface {
	ServeWs(w http.ResponseWriter, r *http.Request)
	CreateGroup(w http.ResponseWriter, r *http.Request)
	DeleteGroup(w http.ResponseWriter, r *http.Request) error
}

type api struct {
	hub server.Hub
	// client server.Client
}

// func NewApi(hub server.Hub, client server.Client) Api {
func NewApi(hub server.Hub) Api {
	return &api{
		hub: hub,
		// client: client,
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// func (a *api) ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
func (a *api) ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	id := r.Header.Get("id")

	//#print log
	fmt.Println("client.ServeWs.id = ", id)

	// client := NewClient(hub, conn, send, id)
	// client.Register(&client)

	client := &server.Client{Hub: &a.hub, Conn: conn, Send: make(chan server.Msg, 256), ID: id}

	clientHub := *client.Hub
	registerChan := *clientHub.Register()
	registerChan <- client

	//#print log
	// fmt.Println("client.ServeWs.client = ", client)
	// fmt.Println("client.ServeWs.client.hub = ", *client.Hub)

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.WritePump()
	go client.ReadPump()
}

// func (a *api) CreateGroup(hub *Hub, w http.ResponseWriter, r *http.Request) {
func (a *api) CreateGroup(w http.ResponseWriter, r *http.Request) {

	var newGroup []string
	initiatorID := r.Header.Get("id")
	newGroup = append(newGroup, initiatorID)
	joinersString := r.FormValue("joiner")
	joinersArray := strings.Split(joinersString, ", ")
	newGroup = append(newGroup, joinersArray...)

	//#print log
	// fmt.Println("server.hub.CreatedGroup: newGroup = ", newGroup)

	a.hub.CreateGroup(newGroup)
}

// func (a *api) DeleteGroup() (hub *Hub, w http.ResponseWriter, r *http.Request) {
func (a *api) DeleteGroup(w http.ResponseWriter, r *http.Request) error {
	clientID := r.Header.Get("id")
	groupID := r.FormValue(("groupID"))
	err := a.hub.DeleteGroup(groupID, clientID)
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusOK)
	fmt.Printf("group %s deleted by %s", groupID, clientID)
	return nil
}
