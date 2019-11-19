package main

import (
	"strconv"
)

// BroadCast is room config
type BroadCast struct {
	to  uint64
	msg []byte
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	// clients map[*Client]bool
	clients map[uint64]map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan *BroadCast

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan *BroadCast),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[uint64]map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			if _, ok := h.clients[client.room]; !ok {
				h.clients[client.room] = map[*Client]bool{}
				h.clients[client.room][client] = true
			} else {
				for cls := range h.clients[client.room] {
					cls.send <- []byte(`{"header":"conn", "gender":` + strconv.FormatBool(client.gender) + `,"from":` + strconv.FormatBool(client.id) + `}`)
					client.send <- []byte(`{"header":"conn", "gender":` + strconv.FormatBool(cls.gender) + `,"from":` + strconv.FormatBool(cls.id) + `}`)
				}
				h.clients[client.room][client] = true
			}
		case client := <-h.unregister:
			if _, ok := h.clients[client.room][client]; ok {
				close(client.send)
				delete(h.clients[client.room], client)
			}
			if len(h.clients[client.room]) == 0 {
				delete(h.clients, client.room)
			}
		case message := <-h.broadcast:
			for client := range h.clients[message.to] {
				select {
				case client.send <- message.msg:
				default:
					close(client.send)
					delete(h.clients[client.room], client)
				}
			}
		}
	}
}
