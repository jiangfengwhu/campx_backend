package main

import (
	// "bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

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
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	id     bool
	gender bool
	room   uint64
	hub    *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	// c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Println(err)
			condi := "close"
			if websocket.IsCloseError(err, 3888) {
				condi = "leave"
			}
			msg := []byte(`{"header":"` + condi + `"}`)
			tomessage := &BroadCast{
				msg: msg,
				to:  c.room,
			}
			c.hub.broadcast <- tomessage
			break
		}
		var re map[string]interface{}
		// msg := bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		json.Unmarshal(message, &re)
		re["from"] = c.id
		msgHeader := re["header"].(string)
		mid := re["mid"].(float64)
		delete(re, "mid")
		switch msgHeader {
		case "text", "img":
			newmsg, _ := json.Marshal(&re)
			tomessage := &BroadCast{
				msg:  newmsg,
				to:   c.room,
				from: c.id,
				mid:  mid,
			}
			c.hub.broadcastTo <- tomessage
			dataBase.Collection("anchats").InsertOne(context.Background(), re)
		default:
			log.Println("default")
			newmsg, _ := json.Marshal(&re)
			tomessage := &BroadCast{
				msg: newmsg,
				to:  c.room,
			}
			c.hub.broadcast <- tomessage
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type wsModel struct {
	Gender *bool  `form:"gender" binding:"required"`
	Room   uint64 `form:"room"`
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, c *gin.Context) {
	var params wsModel
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "msg": err.Error()})
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}
	id := false
	var client *Client
	if params.Room > 0 && len(hub.clients[params.Room]) == 1 {
		for cl := range hub.clients[params.Room] {
			id = !cl.id
		}
		client = &Client{hub: hub, conn: conn, send: make(chan []byte, 256), id: id, room: params.Room, gender: *params.Gender}
		client.hub.register <- client
	} else {
		var mux sync.Mutex
		mux.Lock()
		var room uint64
		if len(hub.clients[roomSq]) == 1 {
			room = roomSq
			roomSq++
		} else {
			id = true
			room = roomSq
		}
		client = &Client{hub: hub, conn: conn, send: make(chan []byte, 256), id: id, room: room, gender: *params.Gender}
		client.hub.register <- client
		mux.Unlock()
	}
	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
