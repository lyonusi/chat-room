// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"chatroom/api"
	"chatroom/server"
	"log"
	"net/http"
)

// var hubServer server.Hub

func main() {
	hub := server.NewHub()
	endpoint := api.NewApi(hub)
	go hub.Run()
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		endpoint.ServeWs(w, r)
	})

	http.HandleFunc("/createchatroom", func(w http.ResponseWriter, r *http.Request) {
		endpoint.CreateGroup(w, r)
	})

	http.HandleFunc("/deletechatroom", func(w http.ResponseWriter, r *http.Request) {
		endpoint.DeleteGroup(w, r)
	})

	// http.HandleFunc("/ws1", func(w http.ResponseWriter, r *http.Request) {
	// 	serveWs1(hub, w, r)
	// })
	err := http.ListenAndServe(":8090", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
