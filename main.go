package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

var (
	register       = make(chan *websocket.Conn)
	unregister     = make(chan int)
	clients        = make(map[int]*websocket.Conn)
	broadcast      = make(chan []byte)
	length     int = 0
)

const (
	SENDA    = "senda"
	MITSUO   = "mitsuo"
	NAHANAHA = "nahanaha"
)

type RequestPayloadRoot struct {
	Type    string `json:"type"`
	Payload json.RawMessage
}

type ResponsePayloadRoot struct {
	Status  string `json:"status"`
	Payload json.RawMessage
}

type RegisterResponsePayload struct {
	Status string `json:"status"`
	ID     int    `json:"id"`
	Type   string `json:"type"`
}

type MessageResponsePayload struct {
	Status string `json:"status"`
	ID     int    `json:"id"`
	Type   string `json:"body"`
}

func main() {
	app := fiber.New()

	app.Use("/", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/", websocket.New(func(c *websocket.Conn) {
		var (
			msg []byte
			err error
			//root *RequestPayloadRoot
		)

		defer func() {
			unregister <- length
			c.Close()
		}()

		for {
			if _, msg, err = c.ReadMessage(); err != nil {
				log.Println("read:", err)
			}
			log.Printf("recv: %s", msg)
			//broadcast <- msg
			Handler(c, msg)
			/*
				if err = c.WriteMessage(mt, msg); err != nil {
					log.Println("write:", err)
					break
				}
			*/
		}
	}))

	go WebSocketConnectionHandler()

	log.Fatal(app.Listen(":3000"))
}

func Handler(c *websocket.Conn, msg []byte) {
	root := new(RequestPayloadRoot)

	if err := json.Unmarshal(msg, root); err != nil {
		log.Println("unmarshal", err)
		return
	}
	switch root.Type {
	case "register":
		log.Println(msg)
		register <- c
		if err := c.WriteJSON(map[string]interface{}{"type": "register", "status": "ok", "id": length}); err != nil {
			log.Println("write:", err)
			return
		}
		broadcast <- []byte(fmt.Sprintf("{\"type\": \"join\", \"id\": %d}", length))
	case "message":
		log.Println(msg)
		msgroot := new(MessageResponsePayload)
		if err := json.Unmarshal(msg, msgroot); err != nil {
			log.Println("unmarshal", err)
			return
		}
		switch msgroot.Status {
		case SENDA:
		case MITSUO:
		case NAHANAHA:
		default:
		}
		// きたmessageでswitchする
		// historyに書き込む（SQLのほうがよい？
	case "playerlist":
		log.Println(msg)
		keys := make([]int, 0, len(clients))
		for k := range clients {
			keys = append(keys, k)
		}
		if err := c.WriteJSON(map[string]interface{}{"type": "player_list", "status": "ok", "player": keys}); err != nil {
			log.Println("write:", err)
			return
		}
	default:
		log.Println(msg)
	}

}

func FindWebsocketConnectionById(index int) *websocket.Conn {
	for i, client := range clients {
		if index == i {
			return client
		}
	}
	return nil
}

func WebSocketConnectionHandler() {
	for {
		select {
		case conn := <-register:
			length += 1
			clients[length] = conn
		case msg := <-broadcast:
			for _, client := range clients {
				if err := client.WriteMessage(websocket.TextMessage, msg); err != nil {
					log.Println("write:", err)
					break
				}
			}
		case i := <-unregister:
			delete(clients, i)
			log.Println("conn unregistered")
		}
	}
}
