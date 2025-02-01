package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/exp/rand"
)

// a binary that is started, tasked with being a simulated user for the chat
// - needs to interact with the server in a realistic way
// - record metrics (requests out per second, failures, avg message size, etc.)
// - test for errors
//// - check for correct message order

const API_DOMAIN = "localhost:8080"

const CHAT_WS_URL = "ws://" + API_DOMAIN + "/chat"
const CREATE_USER_URL = "http://" + API_DOMAIN + "/user/create"
const LIST_USERS_URL = "http://" + API_DOMAIN + "/user/list"

var allUserIds = make([]string, 0)
var userPoolCount = atomic.Int32{}
var userPool = sync.Map{}

func popUser() string {
	userPoolCount.Add(-1)
	var uid string
	userPool.Range(func(key, value interface{}) bool {
		uid = key.(string)
		return true
	})
	return uid
}

func pushUser(uid string) {
	userPoolCount.Add(1)
	userPool.Store(uid, struct{}{})
}

func poolSize() int {
	return int(userPoolCount.Load())
}

// Configuration
var NumberOfUsers = 10 // number of concurrent users
const MAX_USERS = 4200
const UserCreationRate = 0.2  // probability of creating a new user instead of using an existing one
const MeanUserOnlineTime = 60 // in seconds
const TimeBetweenActions = 1500 * time.Millisecond
const ProbCreateChat = 0.003
const ProbAddUsers = 0.03
const ProbSwitchChat = 0.05
const LoadIncreaseRate = 35 // max number of users added per second

func main() {
	resp, err := http.Get(LIST_USERS_URL)
	if err != nil {
		log.Fatal("fatal", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("fatal", err)
	}

	user := []struct {
		ID            string `json:"ID"`
		NumberOfChats int    `json:"NumberOfChats"`
	}{}
	err = json.Unmarshal(body, &user)
	if err != nil {
		log.Fatal("fatal", err)
	}

	for _, u := range user {
		allUserIds = append(allUserIds, u.ID)
		pushUser(u.ID)
	}

	var userCount int32 = 0
	st := time.Now()

	for {
		if time.Since(st) > time.Second/LoadIncreaseRate {
			if NumberOfUsers < MAX_USERS {
				NumberOfUsers += 1
			}
			st = time.Now()
		}
		if atomic.LoadInt32(&userCount) >= int32(NumberOfUsers) {
			time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
			continue
		}

		r := rand.Float32()
		uid := ""
		if r < UserCreationRate || poolSize() == 0 {
			resp, err := http.Get(CREATE_USER_URL)
			if err != nil {
				log.Fatal("fatal", err)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatal("fatal", err)
			}
			uid = string(body)
			allUserIds = append(allUserIds, uid)
		} else {
			uid = popUser()
		}

		userCount += 1
		su := &SimulatedUser{uid: uid}
		go su.simulateUser(&userCount)
		// time.Sleep(time.Duration(rand.Intn(1)) * time.Millisecond)
	}
}

type ChatWithLatestMessage struct {
	ChatID          string
	LatestMessageAt int64
}

type SimulatedUser struct {
	uid     string
	chatIds []string
	conn    *websocket.Conn
}

func (u *SimulatedUser) readPump() {
	for {
		var dest map[string]interface{}
		err := u.conn.ReadJSON(&dest)
		if err != nil {
			fmt.Println("error reading json: ", err)
			return // todo: proper handling of conn
		}
		if dest["opcode"] == "chat_list" {
			if dest["chats"] == nil {
				continue
			}
			chats := dest["chats"].([]interface{})
			for _, raw_chat := range chats {
				chat := raw_chat.(map[string]interface{})
				u.chatIds = append(u.chatIds, chat["ChatID"].(string))
			}
		} else if dest["opcode"] == "chat_list_notification" {
			u.chatIds = append(u.chatIds, dest["chat_id"].(string))
		} else if dest["opcode"] == "chat_created" {
			u.chatIds = append(u.chatIds, dest["chat_id"].(string))
		}
	}
}

func (u *SimulatedUser) simulateUser(userCount *int32) {
	defer func() { atomic.AddInt32(userCount, -1) }()
	defer func() { pushUser(u.uid) }()

	onlineTime := max(0, rand.NormFloat64()*MeanUserOnlineTime/6+MeanUserOnlineTime)
	ctx, cancel := context.WithDeadline(context.TODO(), time.Now().Add(time.Duration(onlineTime)*time.Second))
	defer cancel()

	conn, _, err := websocket.DefaultDialer.Dial(CHAT_WS_URL+"?uid="+u.uid, nil)
	if err != nil {
		fmt.Println("fatal", err)
		return
	}
	u.conn = conn
	go u.readPump()

	listChats(u.uid, u.conn)

eventLoop:
	for {
		select {
		case <-ctx.Done():
			break eventLoop
		default:
		}

		p := rand.Float32()
		if len(u.chatIds) == 0 || p < ProbCreateChat {
			createChat(u.uid, u.conn)
			time.Sleep(time.Second) // additional wait just in case I guess
		} else if p < ProbCreateChat+ProbAddUsers {
			chat_id := u.chatIds[rand.Intn(len(u.chatIds))]
			nUsers := rand.Intn(8)
			for i := 0; i < nUsers; i++ {
				user := allUserIds[rand.Intn(len(allUserIds))]
				addUser(chat_id, user, u.conn)
			}
		} else if p < ProbCreateChat+ProbAddUsers+ProbSwitchChat {
			chat_id := u.chatIds[rand.Intn(len(u.chatIds))]
			listMessages(chat_id, u.conn)
			listUsers(chat_id, u.conn)
		} else {
			chat_id := u.chatIds[rand.Intn(len(u.chatIds))]
			msgLen := max(1, rand.NormFloat64()*10+20)
			sendMessage(u.uid, chat_id, strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", int(msgLen)), u.conn)
		}

		time.Sleep(TimeBetweenActions)
	}

	err = u.conn.Close()
	if err != nil {
		fmt.Println("close error: ", err)
	}

	fmt.Println("Done", u.uid, onlineTime)
}

type ListChatsRequest struct {
	Opcode string                 `json:"opcode"`
	Data   map[string]interface{} `json:"data"`
}

func listChats(uid string, conn *websocket.Conn) {
	conn.WriteJSON(ListChatsRequest{Opcode: "list_chats", Data: map[string]interface{}{"uid": uid}})
}

type ListMessagesRequest struct {
	Opcode string                 `json:"opcode"`
	Data   map[string]interface{} `json:"data"`
}

func listMessages(chat_id string, conn *websocket.Conn) {
	conn.WriteJSON(ListMessagesRequest{Opcode: "list_messages", Data: map[string]interface{}{"chat_id": chat_id, "page": 0}})
}

type SendMessageRequest struct {
	Opcode string                 `json:"opcode"`
	Data   map[string]interface{} `json:"data"`
}

func sendMessage(uid string, chat_id string, text string, conn *websocket.Conn) {
	conn.WriteJSON(SendMessageRequest{Opcode: "send_message", Data: map[string]interface{}{"uid": uid, "chat_id": chat_id, "text": text}})
}

type ListUsersRequest struct {
	Opcode string                 `json:"opcode"`
	Data   map[string]interface{} `json:"data"`
}

func listUsers(chat_id string, conn *websocket.Conn) {
	conn.WriteJSON(ListUsersRequest{Opcode: "list_users", Data: map[string]interface{}{"chat_id": chat_id}})
}

type CreateChatRequest struct {
	Opcode string                 `json:"opcode"`
	Data   map[string]interface{} `json:"data"`
}

func createChat(uid string, conn *websocket.Conn) {
	conn.WriteJSON(CreateChatRequest{Opcode: "create_chat", Data: map[string]interface{}{"uid": uid}})
}

type AddUserRequest struct {
	Opcode string                 `json:"opcode"`
	Data   map[string]interface{} `json:"data"`
}

func addUser(chat_id string, uid string, conn *websocket.Conn) {
	conn.WriteJSON(AddUserRequest{Opcode: "add_user", Data: map[string]interface{}{"chat_id": chat_id, "user_id": uid}})
}
