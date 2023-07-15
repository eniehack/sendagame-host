package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"reflect"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

var (
	register       = make(chan *websocket.Conn)
	unregister     = make(chan *websocket.Conn)
	clients        = make([]*websocket.Conn, 1024)
	broadcast      = make(chan []byte)
	history        = make([]*History, 4)
	length     int = 0
)

const (
	SEC            = "Sec"
	HACK           = "Hack"
	NAHANAHA       = "365"
	CLIENTS_LENGTH = 4
)

type History struct {
	PeerID   int
	Message  string
	TargetID []int
}

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

type MessageRequestPayload struct {
	Player     int    `json:"player"`
	NextPlayer []int  `json:"nextplayer"`
	Message    string `json:"message"`
}

type MessageResponsePayload struct {
	Type       string `json:"type"`
	Player     int    `json:"player"`
	PlayerList []int  `json:"player_list"`
	NextPlayer []int  `json:"nextplayer"`
	Message    string `json:"message"`
}

var virtualizedhost string

func init() {
	flag.StringVar(&virtualizedhost, "host", "", "host of virtualization")
}

func main() {
	flag.Parse()

	app := fiber.New()
	app.Use(recover.New())
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
		)

		defer func() {
			c.Close()
			c = nil
		}()

		for {
			if _, msg, err = c.ReadMessage(); err != nil {
				log.Println("read:", err)
			}
			log.Printf("recv: %s", msg)
			if err := Handler(c, msg); err != nil {
				return
			}
		}
	}))

	go WebSocketConnectionHandler()

	log.Fatal(app.Listen(":8080"))
}

func Handler(c *websocket.Conn, msg []byte) error {
	root := new(RequestPayloadRoot)

	if err := json.Unmarshal(msg, root); err != nil {
		log.Println("unmarshal", err)
		return err
	}
	switch root.Type {
	case "register":
		register <- c
		if err := c.WriteJSON(map[string]interface{}{"type": "register", "status": "ok", "id": length}); err != nil {
			log.Println("write:", err)
			log.Println("c.WriteJSO, register")
			CloseAllConnection()
			return err
		}
		log.Printf("sent: %v\n", map[string]interface{}{"type": "register", "status": "ok", "id": length})
		keys := GetKeyArray(clients)
		if 5 == GetCountOfNonNilElements(clients) {
			payload := &MessageResponsePayload{
				Type:       "message",
				PlayerList: keys,
				Player:     -1,
				Message:    "start",
				NextPlayer: []int{rand.Intn(CLIENTS_LENGTH)},
			}
			payloadb, _ := json.Marshal(payload)
			broadcast <- payloadb
		}

	case "message":
		//log.Println("message:", msg)
		msgroot := new(MessageRequestPayload)
		if err := json.Unmarshal(msg, msgroot); err != nil {
			log.Println("unmarshal", err)
			return err
		}
		switch msgroot.Message {
		case SEC:
			senda := &History{
				PeerID:   msgroot.Player,
				Message:  msgroot.Message,
				TargetID: msgroot.NextPlayer,
			}

			keys := GetKeyArray(clients)
			payload := new(MessageResponsePayload)
			if history[0] == nil {
				payload = &MessageResponsePayload{
					Type:       "message",
					PlayerList: keys,
					Player:     msgroot.Player,
					Message:    msgroot.Message,
					NextPlayer: msgroot.NextPlayer,
				}
			} else {
				payload = &MessageResponsePayload{
					Type:       "message",
					PlayerList: keys,
					Player:     msgroot.Player,
					Message:    msgroot.Message,
					NextPlayer: msgroot.NextPlayer,
				}
			}

			payloadb, _ := json.Marshal(payload)
			broadcast <- payloadb
			for i := range history {
				history[i] = nil
			}
			history[0] = senda
			//step += 1
			UpdateStateGraph(msgroot.Player, msgroot.NextPlayer[0], 1)
		case HACK:
			if history[0].Message != SEC && contains(history[0].TargetID, msgroot.Player) {
				log.Println("history[0].Message != SEC && contains(history[0].TargetID, msgroot.Player)")
				CloseAllConnection()
			}
			senda := &History{
				PeerID:   msgroot.Player,
				Message:  msgroot.Message,
				TargetID: msgroot.NextPlayer,
			}
			history[1] = senda

			keys := GetKeyArray(clients)
			payload := &MessageResponsePayload{
				Type:       "message",
				PlayerList: keys,
				Player:     msgroot.Player,
				Message:    msgroot.Message,
				NextPlayer: msgroot.NextPlayer,
			}
			payloadb, _ := json.Marshal(payload)
			broadcast <- payloadb
			UpdateStateGraph(msgroot.Player, msgroot.NextPlayer[0], 2)
		case NAHANAHA:

			senda := &History{
				PeerID:   msgroot.Player,
				Message:  msgroot.Message,
				TargetID: msgroot.NextPlayer,
			}
			log.Println("2", history[2])
			log.Println("3", history[3])
			if history[2] == nil {
				history[2] = senda
			} else {
				history[3] = senda
			}
			//log.Println(history[2].PeerID, history[3].PeerID, history[1].TargetID[0])

			if !IsNeighbor([]int{history[2].PeerID}, history[1].TargetID[0]) {
				log.Println("IsNeighbor([]int{history[2].PeerID}, history[1].TargetID[0])")
				log.Println(history[2].PeerID, history[1].TargetID[0])
				CloseAllConnection()
			}
			if history[3] != nil && !IsNeighbor([]int{history[3].PeerID}, history[1].TargetID[0]) {
				log.Println("history[3] != nil && IsNeighbor([]int{history[2].PeerID, history[3].PeerID}, history[1].TargetID[0])")
				log.Println(history[3].PeerID, history[1].TargetID[0])
				CloseAllConnection()
			}

			keys := GetKeyArray(clients)
			payload := &MessageResponsePayload{
				Type:       "message",
				PlayerList: keys,
				Player:     msgroot.Player,
				Message:    msgroot.Message,
				NextPlayer: msgroot.NextPlayer,
			}
			payloadb, err := json.Marshal(payload)
			if err != nil {
				log.Println("payloadb marshal", err)
				return err
			}
			broadcast <- payloadb
			if history[3] != nil {
				UpdateStateGraph(history[2].PeerID, history[3].PeerID, 3)
			}
		default:
			CloseAllConnection()
		}
	default:
		log.Println("recv:", msg)
	}
	return nil
}

func contains(last_target []int, id int) bool {
	for _, i := range last_target {
		if id == i {
			return true
		}
	}
	return false
}

func FindWebsocketConnectionById(index int) *websocket.Conn {
	for i, client := range clients {
		if index == i {
			return client
		}
	}
	return nil
}

func FindIndexByPtr(c *websocket.Conn) int {
	for i, client := range clients {
		if reflect.DeepEqual(&client, &c) {
			return i
		}
	}
	return -1
}

func CloseAllConnection() {
	broadcast <- []byte("{\"status\": \"end\"}")
	for _, c := range clients {
		unregister <- c
	}
}

func IsNeighbor(last_target []int, id int) bool {
	for _, i := range last_target {
		if r, l := GetNeighbor(i); r != id && l != id {
			return false
		}
		return true
	}
	return true
}

func GetNeighbor(index int) (int, int) {
	return (index + 1) % 5, (index - 1) % 5
}

func WebSocketConnectionHandler() {
	for {
		select {
		case conn := <-register:
			clients[length] = conn
			//fmt.Println(clients)
			length += 1
		case msg := <-broadcast:
			for _, client := range clients {
				if client == nil {
					continue
				}
				if err := client.WriteMessage(websocket.TextMessage, msg); err != nil {
					log.Println("write:", err)
					break
				}
			}
			log.Printf("broadcast: %v\n", string(msg))
			/*
				case c := <-unregister:
					c = nil
					log.Println("conn unregistered")
			*/
		}
	}
}

func GetEmptyElementIndex(arr []bool) int {
	for i, elm := range arr {
		if elm {
			return i
		}
	}
	return -1
}

func GetKeyArray(clients []*websocket.Conn) []int {
	keys := make([]int, 0, CLIENTS_LENGTH)
	for i := 0; i < 5; i++ {
		keys = append(keys, i)
	}
	return keys
}

func UpdateStateGraph(peerSubjectID int, peerObjectID int, step int) error {
	client := new(http.Client)

	//log.Println(map[string]int{"Before": peerSubjectID, "After": peerObjectID, "StepNum": step})
	body, err := json.Marshal(map[string]int{"Before": peerSubjectID, "After": peerObjectID, "StepNum": step})
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/run_simulation", virtualizedhost), bytes.NewBuffer(body))
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	log.Println(resp)
	return nil
}

func GetCountOfNonNilElements(keys []*websocket.Conn) int {
	cnt := 0
	for _, elm := range keys {
		if elm != nil {
			cnt++
		}
	}
	return cnt
}
