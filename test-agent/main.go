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
var NumberOfUsers = 10        // number of concurrent users
const UserCreationRate = 0.02 // probability of creating a new user instead of using an existing one
const MeanUserOnlineTime = 60 // in seconds
const TimeBetweenActions = 2000 * time.Millisecond
const ProbCreateChat = 0.0005
const ProbAddUsers = 0.01
const ProbSwitchChat = 0.04

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
		if time.Since(st) > 1*time.Second {
			NumberOfUsers += 2
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
		go simulateUser(uid, &userCount)
		time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
	}
}

type ChatWithLatestMessage struct {
	ChatID          string `json:"chat_id"`
	LatestMessageAt int64  `json:"latest_message"`
}

type SimulatedUser struct {
	uid     string
	chatIds []string
	conn    *websocket.Conn
}

func simulateUser(uid string, userCount *int32) {
	defer func() { atomic.AddInt32(userCount, -1) }()

	defer func() { pushUser(uid) }()

	onlineTime := max(0, rand.NormFloat64()*MeanUserOnlineTime/6+MeanUserOnlineTime)
	ctx, cancel := context.WithDeadline(context.TODO(), time.Now().Add(time.Duration(onlineTime)*time.Second))
	defer cancel()

	conn, _, err := websocket.DefaultDialer.Dial(CHAT_WS_URL+"?uid="+uid, nil)
	if err != nil {
		log.Fatal("fatal", err)
	}

	var dest map[string]interface{}
	var chats interface{}

outer:
	for {
		//fmt.Println("listChats")
		listChats(uid, conn)

		conn.ReadJSON(&dest)
		select {
		case <-ctx.Done():
			break outer
		default:
			opcode, ok := dest["opcode"]
			if ok && opcode == "chat_list" {
				chats, ok = dest["chats"]
				if ok {
					break outer
				}
			}
			//fmt.Println("did not get list_chats: ", dest)
			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
		}
	}

	//fmt.Println("Chats: ", chats)

outer2:
	for {
		select {
		case <-ctx.Done():
			break outer2
		default:
		}

		p := rand.Float32()
		if p < ProbCreateChat {
			// create a new chat
			//fmt.Println("createChat")
			createChat(uid, conn)
			// update chats list on response from server
		} else if p < ProbCreateChat+ProbAddUsers {
			if chats == nil {
				continue
			}
			chats_list := chats.([]interface{})
			chat := chats_list[rand.Intn(len(chats_list))]
			chat_id := chat.(map[string]interface{})["ChatID"].(string)
			nUsers := rand.Intn(5)
			//fmt.Println("addUsers", nUsers)
			for i := 0; i < nUsers; i++ {
				user := allUserIds[rand.Intn(len(allUserIds))]
				addUser(chat_id, user, conn)
			}
		} else if p < ProbCreateChat+ProbAddUsers+ProbSwitchChat {
			//fmt.Println("switchChat")
			if chats == nil {
				continue
			}
			// switch to a random chat
			chats_list := chats.([]interface{})
			chat := chats_list[rand.Intn(len(chats_list))]
			chat_id := chat.(map[string]interface{})["ChatID"].(string)
			listMessages(chat_id, conn)
			listUsers(chat_id, conn)
		} else {
			//fmt.Println("sendMessage")
			// send a message to a random chat
			if chats == nil {
				continue
			}
			chats_list := chats.([]interface{})
			chat := chats_list[rand.Intn(len(chats_list))]
			chat_id := chat.(map[string]interface{})["ChatID"].(string)
			msgLen := max(1, rand.NormFloat64()*10+20)
			sendMessage(uid, chat_id, strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", int(msgLen)), conn)
		}

		time.Sleep(TimeBetweenActions)
	}

	err = conn.Close()
	if err != nil {
		fmt.Println("close error: ", err)
	}

	fmt.Println("Done", uid, onlineTime)
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
