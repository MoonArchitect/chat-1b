package main

import (
	"chat-1b/server/utils/flatbuffer_types/WebsocketMessage"
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

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/gorilla/websocket"
	"golang.org/x/exp/rand"
)

// a binary that is started, tasked with being a simulated user for the chat
// - needs to interact with the server in a realistic way
// - record metrics (requests out per second, failures, avg message size, etc.)
// - test for errors
//// - check for correct message order

const API_DOMAIN = "localhost:8080"
const WS_DOMAIN = "localhost:8081"

const CHAT_WS_URL = "ws://" + WS_DOMAIN + "/chat"
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
const MAX_USERS = 1700
const UserCreationRate = 0.2  // probability of creating a new user instead of using an existing one
const MeanUserOnlineTime = 60 // in seconds
const TimeBetweenActions = 750 * time.Millisecond
const ProbCreateChat = 0.003
const ProbAddUsers = 0.03
const ProbSwitchChat = 0.05
const LoadIncreaseRate = 50 // max number of users added per second

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
		fmt.Println(uid)
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
		_, p, err := u.conn.ReadMessage()
		if err != nil {
			fmt.Println("error reading message: ", err)
			return
		}

		msg := WebsocketMessage.GetRootAsMessage(p, 0)
		var table flatbuffers.Table
		msg.Payload(&table)

		switch msg.PayloadType() {
		case WebsocketMessage.PayloadListChatsResponse:
			resp := new(WebsocketMessage.ListChatsResponse)
			resp.Init(table.Bytes, table.Pos)
			for i := 0; i < resp.ChatsLength(); i++ {
				var chat WebsocketMessage.ChatWithLatestMessage
				if resp.Chats(&chat, i) {
					u.chatIds = append(u.chatIds, string(chat.ChatId()))
				}
			}

		case WebsocketMessage.PayloadChatListNotification:
			var notif WebsocketMessage.ChatListNotification
			notif.Init(table.Bytes, table.Pos)
			u.chatIds = append(u.chatIds, string(notif.ChatId()))

		case WebsocketMessage.PayloadCreateChatResponse:
			var resp WebsocketMessage.CreateChatResponse
			resp.Init(table.Bytes, table.Pos)
			u.chatIds = append(u.chatIds, string(resp.ChatId()))
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
			msgLen := max(1, rand.NormFloat64()*10+5)
			sendMessage(u.uid, chat_id, strings.Repeat("Lorem ipsum", int(msgLen)), u.conn)
		}

		time.Sleep(TimeBetweenActions)
	}

	err = u.conn.Close()
	if err != nil {
		fmt.Println("close error: ", err)
	}

	fmt.Println("Done", u.uid, onlineTime)
}

func listChats(uid string, conn *websocket.Conn) {
	builder := flatbuffers.NewBuilder(1024)

	// Create the ListChatsRequest
	uidOffset := builder.CreateString(uid)
	WebsocketMessage.ListChatsRequestStart(builder)
	WebsocketMessage.ListChatsRequestAddUid(builder, uidOffset)
	request := WebsocketMessage.ListChatsRequestEnd(builder)

	// Create the Message wrapper
	WebsocketMessage.MessageStart(builder)
	WebsocketMessage.MessageAddOpcode(builder, WebsocketMessage.OpcodeLIST_CHATS_REQUEST)
	WebsocketMessage.MessageAddPayloadType(builder, WebsocketMessage.PayloadListChatsRequest)
	WebsocketMessage.MessageAddPayload(builder, request)
	msg := WebsocketMessage.MessageEnd(builder)

	builder.Finish(msg)
	conn.WriteMessage(websocket.BinaryMessage, builder.FinishedBytes())
}

func listMessages(chat_id string, conn *websocket.Conn) {
	builder := flatbuffers.NewBuilder(1024)

	chatIDOffset := builder.CreateString(chat_id)
	WebsocketMessage.ListMessagesRequestStart(builder)
	WebsocketMessage.ListMessagesRequestAddChatId(builder, chatIDOffset)
	WebsocketMessage.ListMessagesRequestAddPage(builder, 0)
	request := WebsocketMessage.ListMessagesRequestEnd(builder)

	WebsocketMessage.MessageStart(builder)
	WebsocketMessage.MessageAddOpcode(builder, WebsocketMessage.OpcodeLIST_MESSAGES_REQUEST)
	WebsocketMessage.MessageAddPayloadType(builder, WebsocketMessage.PayloadListMessagesRequest)
	WebsocketMessage.MessageAddPayload(builder, request)
	msg := WebsocketMessage.MessageEnd(builder)

	builder.Finish(msg)
	conn.WriteMessage(websocket.BinaryMessage, builder.FinishedBytes())
}

func sendMessage(uid string, chat_id string, text string, conn *websocket.Conn) {
	builder := flatbuffers.NewBuilder(1024)

	textOffset := builder.CreateString(text)
	uidOffset := builder.CreateString(uid)
	chatIDOffset := builder.CreateString(chat_id)

	WebsocketMessage.SendMessageRequestStart(builder)
	WebsocketMessage.SendMessageRequestAddText(builder, textOffset)
	WebsocketMessage.SendMessageRequestAddUid(builder, uidOffset)
	WebsocketMessage.SendMessageRequestAddChatId(builder, chatIDOffset)
	request := WebsocketMessage.SendMessageRequestEnd(builder)

	WebsocketMessage.MessageStart(builder)
	WebsocketMessage.MessageAddOpcode(builder, WebsocketMessage.OpcodeSEND_MESSAGE_REQUEST)
	WebsocketMessage.MessageAddPayloadType(builder, WebsocketMessage.PayloadSendMessageRequest)
	WebsocketMessage.MessageAddPayload(builder, request)
	msg := WebsocketMessage.MessageEnd(builder)

	builder.Finish(msg)
	conn.WriteMessage(websocket.BinaryMessage, builder.FinishedBytes())
}

func listUsers(chat_id string, conn *websocket.Conn) {
	builder := flatbuffers.NewBuilder(1024)

	chatIDOffset := builder.CreateString(chat_id)
	WebsocketMessage.ListUsersRequestStart(builder)
	WebsocketMessage.ListUsersRequestAddChatId(builder, chatIDOffset)
	request := WebsocketMessage.ListUsersRequestEnd(builder)

	WebsocketMessage.MessageStart(builder)
	WebsocketMessage.MessageAddOpcode(builder, WebsocketMessage.OpcodeLIST_USERS_REQUEST)
	WebsocketMessage.MessageAddPayloadType(builder, WebsocketMessage.PayloadListUsersRequest)
	WebsocketMessage.MessageAddPayload(builder, request)
	msg := WebsocketMessage.MessageEnd(builder)

	builder.Finish(msg)
	conn.WriteMessage(websocket.BinaryMessage, builder.FinishedBytes())
}

func createChat(uid string, conn *websocket.Conn) {
	builder := flatbuffers.NewBuilder(1024)

	uidOffset := builder.CreateString(uid)
	WebsocketMessage.CreateChatRequestStart(builder)
	WebsocketMessage.CreateChatRequestAddUid(builder, uidOffset)
	request := WebsocketMessage.CreateChatRequestEnd(builder)

	WebsocketMessage.MessageStart(builder)
	WebsocketMessage.MessageAddOpcode(builder, WebsocketMessage.OpcodeCREATE_CHAT_REQUEST)
	WebsocketMessage.MessageAddPayloadType(builder, WebsocketMessage.PayloadCreateChatRequest)
	WebsocketMessage.MessageAddPayload(builder, request)
	msg := WebsocketMessage.MessageEnd(builder)

	builder.Finish(msg)
	conn.WriteMessage(websocket.BinaryMessage, builder.FinishedBytes())
}

func addUser(chat_id string, uid string, conn *websocket.Conn) {
	builder := flatbuffers.NewBuilder(1024)

	uidOffset := builder.CreateString(uid)
	chatIDOffset := builder.CreateString(chat_id)
	WebsocketMessage.AddUserRequestStart(builder)
	WebsocketMessage.AddUserRequestAddUserId(builder, uidOffset)
	WebsocketMessage.AddUserRequestAddChatId(builder, chatIDOffset)
	request := WebsocketMessage.AddUserRequestEnd(builder)

	WebsocketMessage.MessageStart(builder)
	WebsocketMessage.MessageAddOpcode(builder, WebsocketMessage.OpcodeADD_USER_REQUEST)
	WebsocketMessage.MessageAddPayloadType(builder, WebsocketMessage.PayloadAddUserRequest)
	WebsocketMessage.MessageAddPayload(builder, request)
	msg := WebsocketMessage.MessageEnd(builder)

	builder.Finish(msg)
	conn.WriteMessage(websocket.BinaryMessage, builder.FinishedBytes())
}
