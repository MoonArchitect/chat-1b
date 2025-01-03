package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
	"github.com/rs/cors"
)

func main() {
	db, err := sqlx.Open("sqlite3", "../db-data/sqlite-database.db") // Open the created SQLite File
	if err != nil {
		panic(err)
	}
	defer db.Close()
	repo := newDbRepo(db)
	connMap := sync.Map{}
	t := hub{repo: repo, connList: &connMap}

	fmt.Println("Starting the server")

	// add middleware to handle CORS
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	http.Handle("/chat", corsMiddleware.Handler(http.HandlerFunc(t.wsHandler)))
	http.Handle("/user/create", corsMiddleware.Handler(http.HandlerFunc(createUserHandler)))
	http.Handle("/user/list", corsMiddleware.Handler(http.HandlerFunc(listUsersHandler(repo))))
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Exit")
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	uid := uuid.NewString()
	w.Header().Set("Content-Type", "text/plain")
	_, err := w.Write([]byte(uid))
	if err != nil {
		fmt.Println("error: failed to write uid: ", err)
	}
}

func listUsersHandler(repo IDbRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := repo.ListAllUsers(context.TODO())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type hub struct {
	repo     IDbRepo
	connList *sync.Map
}

// panic handler middleware

func (h hub) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	uid := r.URL.Query().Get("uid")
	if uid == "" {
		fmt.Println("uid is required")
		conn.Close()
		return
	}
	// conn.Close()
	// fmt.Println("CLOSED")
	go readRoutine(conn, h, uid)
	// go writeRoutine(conn, h)
}

type CreateChatRequest struct {
	UID string `json:"uid"`
}

type ListChatsRequest struct {
	UID string `json:"uid"`
}

type ListChatsResponse struct {
	Opcode string                  `json:"opcode"`
	Chats  []ChatWithLatestMessage `json:"chats"`
}

type ListMessagesRequest struct {
	ChatID string `json:"chat_id"`
	Page   uint64 `json:"page"`
}

type ListMessagesResponse struct {
	Opcode   string      `json:"opcode"`
	ChatID   string      `json:"chat_id"`
	Messages []MessageDB `json:"messages"`
}

type CreateChatResponse struct {
	Opcode string `json:"opcode"`
	ChatID string `json:"chat_id"`
}

type ListUsersResponse struct {
	Opcode string   `json:"opcode"`
	Users  []string `json:"users"`
	ChatID string   `json:"chat_id"`
}

type CreateMessageResponse struct {
	Opcode         string `json:"opcode"`
	ChatID         string `json:"chat_id"`
	Text           string `json:"text"`
	UserID         string `json:"user_id"`
	CreatedAtMicro int64  `json:"created_at"`
	MsgID          string `json:"msg_id"`
}

type AddUserResponse struct {
	Opcode string `json:"opcode"`
	ChatID string `json:"chat_id"`
	UserID string `json:"user_id"`
}

type MessageNotification struct {
	Opcode         string `json:"opcode"`
	ChatID         string `json:"chat_id"`
	Text           string `json:"text"`
	UserID         string `json:"user_id"`
	CreatedAtMicro int64  `json:"created_at"`
	MsgID          string `json:"msg_id"`
}

type UserListNotification struct {
	Opcode    string `json:"opcode"`
	ChatID    string `json:"chat_id"`
	NewUserID string `json:"new_user_id"`
}

type ChatListNotification struct {
	Opcode       string `json:"opcode"`
	ChatID       string `json:"chat_id"`
	UserID       string `json:"user_id"`
	LastActivity int64  `json:"last_activity"`
}

func readRoutine(conn *websocket.Conn, h hub, conn_uid string) {
	h.connList.Store(conn_uid, conn)
	defer h.connList.Delete(conn_uid)

	for {
		ctx := context.TODO()
		var dest struct {
			Opcode string                 `json:"opcode"`
			Data   map[string]interface{} `json:"data"`
		}
		err := conn.ReadJSON(&dest)
		if err != nil {
			fmt.Println("err: ", err)
			fmt.Println("closing connection due to error")
			err = conn.Close()
			if err != nil {
				fmt.Println("error closing conn: ", err)
			}
			break
		}

		fmt.Println("dest: ", dest)

		switch dest.Opcode {
		case "list_chats":
			uid, ok := dest.Data["uid"].(string)
			if !ok {
				fmt.Println("failed to read dest.Data: ", dest.Data)
				continue
			}
			chatIds, err := h.repo.ListChats(ctx, uid)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}
			resp := ListChatsResponse{Opcode: "chat_list", Chats: chatIds}

			err = conn.WriteJSON(resp)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}
		case "list_messages":
			chat_id, ok := dest.Data["chat_id"].(string)
			if !ok {
				fmt.Println("failed to read chat_id: ", dest.Data)
				continue
			}
			page, ok := dest.Data["page"]
			if !ok {
				fmt.Println("failed to read page: ", dest.Data, "type: ", reflect.TypeOf(page))
				continue
			}

			messages, err := h.repo.ListMessages(ctx, chat_id, uint64(page.(float64)))
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}

			resp := ListMessagesResponse{Opcode: "chat_messages", Messages: messages, ChatID: chat_id}

			err = conn.WriteJSON(resp)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}
		case "list_users":
			chat_id, ok := dest.Data["chat_id"].(string)
			if !ok {
				fmt.Println("failed to read chat_id: ", dest.Data)
				continue
			}
			users, err := h.repo.ListChatUsers(ctx, chat_id)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}
			resp := ListUsersResponse{Opcode: "chat_users", Users: users, ChatID: chat_id}
			err = conn.WriteJSON(resp)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}
		case "send_message":
			uid, ok := dest.Data["uid"].(string)
			if !ok {
				fmt.Println("failed to read uid: ", dest.Data)
				continue
			}
			chat_id, ok := dest.Data["chat_id"].(string)
			if !ok {
				fmt.Println("failed to read chat_id: ", dest.Data)
				continue
			}
			text, ok := dest.Data["text"].(string)
			if !ok {
				fmt.Println("failed to read text: ", dest.Data)
				continue
			}
			created_at := time.Now().UnixMicro()
			msg_id, err := h.repo.CreateMessage(ctx, uid, chat_id, text, created_at)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}

			resp := CreateMessageResponse{Opcode: "message_sent", ChatID: chat_id, Text: text, UserID: uid, CreatedAtMicro: created_at, MsgID: msg_id}
			err = conn.WriteJSON(resp)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}

			err = notifyAboutMessage(ctx, h, chat_id, text, created_at, msg_id, uid)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}

		case "create_chat":
			uid, ok := dest.Data["uid"].(string)
			if !ok {
				fmt.Println("failed to read uid: ", dest.Data)
				continue
			}
			chat_id, err := h.repo.CreateChat(ctx, uid)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}
			resp := CreateChatResponse{Opcode: "chat_created", ChatID: chat_id}
			err = conn.WriteJSON(resp)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}
		case "add_user":
			// notify all other users that a user was added, show this as a status event in chat
			chat_id, ok := dest.Data["chat_id"].(string)
			if !ok {
				fmt.Println("failed to read chat_id: ", dest.Data)
				continue
			}
			uid, ok := dest.Data["user_id"].(string)
			if !ok {
				fmt.Println("failed to read user_id: ", dest.Data)
				continue
			}
			err := h.repo.AddUser(ctx, chat_id, uid)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}

			// add system message to chat
			created_at := time.Now().UnixMicro()
			sys_msg_id, err := h.repo.CreateMessage(ctx, "system", chat_id, "User "+uid+" added", created_at)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}
			//resp := AddUserResponse{Opcode: "user_added", ChatID: chat_id, UserID: uid}
			//err = conn.WriteJSON(resp)
			//if err != nil {
			//	fmt.Println("err: ", err)
			//	continue
			//}

			//resp2 := CreateMessageResponse{Opcode: "message_sent", ChatID: chat_id, Text: "User " + uid + " added", UserID: "system", CreatedAtMicro: time.Now().UnixMicro(), MsgID: sys_msg_id}
			//err = conn.WriteJSON(resp2)
			//if err != nil {
			//	fmt.Println("err: ", err)
			//	continue
			//}

			err = notifyAboutMessage(ctx, h, chat_id, "User "+uid+" added", created_at, sys_msg_id, "system")
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}

			err = notifyAboutUserList(ctx, h, chat_id, "system", uid)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}

			err = notifyAboutChatList(h, chat_id, uid, created_at)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}
		}
	}
}

func notifyAboutChatList(h hub, chat_id string, uid string, lastActivity int64) error {
	userConnInterface, ok := h.connList.Load(uid)
	if !ok {
		return nil
	}

	userConn, ok := userConnInterface.(*websocket.Conn)
	if !ok {
		return fmt.Errorf("failed to cast user conn: %s", uid)
	}

	resp := ChatListNotification{Opcode: "chat_list_notification", ChatID: chat_id, UserID: uid, LastActivity: lastActivity}
	err := userConn.WriteJSON(resp)
	if err != nil {
		return fmt.Errorf("failed to notify user: %s", uid)
	}

	return nil
}

func notifyAboutUserList(ctx context.Context, h hub, chat_id string, origin_uid string, new_uid string) error {
	user_ids, err := h.repo.ListChatUsers(ctx, chat_id)
	if err != nil {
		fmt.Println("err: ", err)
		return err
	}

	for _, user_id := range user_ids {
		if user_id == origin_uid {
			continue
		}
		userConnInterface, ok := h.connList.Load(user_id)
		if !ok {
			continue
		}

		userConn, ok := userConnInterface.(*websocket.Conn)
		if !ok {
			fmt.Println("failed to cast user conn: ", user_id)
			continue
		}

		resp := UserListNotification{Opcode: "user_list_notification", ChatID: chat_id, NewUserID: new_uid}
		err = userConn.WriteJSON(resp)
		if err != nil {
			fmt.Println("err notifying user: ", err)
			continue
		}
	}
	return nil
}

func notifyAboutMessage(ctx context.Context, h hub, chat_id string, text string, created_at int64, msg_id string, origin_uid string) error {
	// notify all users in the chat
	user_ids, err := h.repo.ListChatUsers(ctx, chat_id)
	if err != nil {
		fmt.Println("err: ", err)
		return err
	}

	for _, user_id := range user_ids {
		if user_id == origin_uid {
			continue
		}
		userConnInterface, ok := h.connList.Load(user_id)
		if !ok {
			continue
		}

		userConn, ok := userConnInterface.(*websocket.Conn)
		if !ok {
			fmt.Println("failed to cast user conn: ", user_id)
			continue
		}

		resp := MessageNotification{Opcode: "message_notification", ChatID: chat_id, Text: text, UserID: origin_uid, CreatedAtMicro: created_at, MsgID: msg_id}
		err = userConn.WriteJSON(resp)
		if err != nil {
			fmt.Println("err notifying user: ", err)
			continue
		}
	}
	return nil
}

// func writeRoutine(conn *websocket.Conn, h hub) {
// 	for {
// 		var dest interface{}
// 		err := conn.ReadJSON(&dest)
// 		if err != nil {
// 			fmt.Println("err: ", err)
// 			continue
// 		}
// 		fmt.Println("dest: ", dest)
// 	}
// }

// open ws comm
// send
// - /create_user -> returns user_id
// - /list_chats?user_id -> returns []chat_id (in the order of latest activity)
// - /read_chat?chat_id=...&page= -> returns []messages
// - /send_chat?chat_id=...&user_id=...&text=... -> returns success or failure
// recv
// - /new_message?message -> there is a new message

/// create some messages
// chat_id := "dcdefe05-2165-4e69-84e6-6d9add08c0f9"
// user_id := "4825629d-55d3-4d00-9c96-6f07d67a89d1"

// res, err := repo.ListMessages(context.TODO(), chat_id, 0)
// fmt.Println(err, res)
// res, err = repo.ListMessages(context.TODO(), chat_id, 1)
// fmt.Println(err, res)

// _, err = repo.CreateMessage(context.TODO(), user_id, chat_id, "random text", time.Now().UnixMicro())
// fmt.Println(err, res)
// _, err = repo.CreateMessage(context.TODO(), user_id, chat_id, "random 2", time.Now().UnixMicro())
// fmt.Println(err, res)
// _, err = repo.CreateMessage(context.TODO(), user_id, chat_id, "text 3", time.Now().UnixMicro())
// fmt.Println(err, res)

// res, err = repo.ListMessages(context.TODO(), chat_id, 0)
// fmt.Println(err, res)
// res, err = repo.ListMessages(context.TODO(), chat_id, 1)
// fmt.Println(err, res)
