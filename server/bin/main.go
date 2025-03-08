package main

import (
	dbrepo "chat-1b/server/db"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/scylladb/gocqlx"

	_ "net/http/pprof"

	_ "github.com/lib/pq"
)

var ChatEndpointRequestCount = promauto.NewCounter(prometheus.CounterOpts{
	Name: "chat_endpoint_request_count",
})
var UserCreateRequestCount = promauto.NewCounter(prometheus.CounterOpts{
	Name: "user_create_request_count",
})
var SendMessageOpCodeRequestCount = promauto.NewCounter(prometheus.CounterOpts{
	Name: "send_message_op_code_request_count",
})
var ListChatsOpCodeRequestCount = promauto.NewCounter(prometheus.CounterOpts{
	Name: "list_chats_op_code_request_count",
})
var ListUsersOpCodeRequestCount = promauto.NewCounter(prometheus.CounterOpts{
	Name: "list_users_op_code_request_count",
})
var CreateChatOpCodeRequestCount = promauto.NewCounter(prometheus.CounterOpts{
	Name: "create_chat_op_code_request_count",
})
var AddUserOpCodeRequestCount = promauto.NewCounter(prometheus.CounterOpts{
	Name: "add_user_op_code_request_count",
})
var ListMessagesOpCodeRequestCount = promauto.NewCounter(prometheus.CounterOpts{
	Name: "list_messages_op_code_request_count",
})
var WebSocketReadRequestCount = promauto.NewCounter(prometheus.CounterOpts{
	Name: "websocket_read_request_count",
})
var WebSocketWriteRequestCount = promauto.NewCounter(prometheus.CounterOpts{
	Name: "websocket_write_request_count",
})
var WebSocketMessageBytesRead = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "websocket_message_bytes_read",
	Buckets: []float64{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 3000, 7000, 10000, 50000, 100000, 1000000},
})
var WebSocketMessageBytesWritten = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "websocket_message_bytes_written",
	Buckets: []float64{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 3000, 7000, 10000, 50000, 100000, 1000000},
})
var WebSocketRequestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "websocket_request_duration",
	Buckets: []float64{10, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500, 1000, 3000, 7000, 10000, 50000, 100000, 1000000},
})

var WebSocketConnectionCount = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "websocket_connection_count",
})

const SYSTEM_UUID = "10000000-0000-4000-0000-000000000001"

func main() {
	cluster_config := gocql.NewCluster("host.docker.internal")
	db_sess, err := gocqlx.WrapSession(cluster_config.CreateSession())
	if err != nil {
		panic(err)
	}

	repo := dbrepo.NewScylladbRepository(db_sess)
	connMap := sync.Map{}
	t := hub{repo: repo, connList: &connMap}

	fmt.Println("Starting the server")

	// add middleware to handle CORS
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	http.Handle("GET /metrics", promhttp.Handler())
	http.Handle("/chat", corsMiddleware.Handler(http.HandlerFunc(t.wsHandler)))
	http.Handle("GET /user/create", corsMiddleware.Handler(http.HandlerFunc(createUserHandler)))
	http.Handle("GET /user/list", corsMiddleware.Handler(http.HandlerFunc(listUsersHandler(repo))))
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Exit")
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	UserCreateRequestCount.Inc()

	uid := uuid.NewString()
	w.Header().Set("Content-Type", "text/plain")
	_, err := w.Write([]byte(uid))
	if err != nil {
		fmt.Println("error: failed to write uid: ", err)
	}
}

func listUsersHandler(repo dbrepo.IDbRepo) http.HandlerFunc {
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
	repo     dbrepo.IDbRepo
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
	ChatEndpointRequestCount.Inc()
	go readRoutine(NewMetricsConn(conn), h, uid)
	// go writeRoutine(conn, h)
}

type CreateChatRequest struct {
	UID string `json:"uid"`
}

type ListChatsRequest struct {
	UID string `json:"uid"`
}

type ListChatsResponse struct {
	Opcode string                         `json:"opcode"`
	Chats  []dbrepo.ChatWithLatestMessage `json:"chats"` // todo don't use dbrepo types in api
}

type ListMessagesRequest struct {
	ChatID string `json:"chat_id"`
	Page   uint64 `json:"page"`
}

type ListMessagesResponse struct {
	Opcode   string             `json:"opcode"`
	ChatID   string             `json:"chat_id"`
	Messages []dbrepo.MessageDB `json:"messages"`
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

// MetricsConn wraps a websocket.Conn with metrics
type MetricsConn struct {
	conn *websocket.Conn
	m    *sync.Mutex
}

// NewMetricsConn creates a new MetricsConn
func NewMetricsConn(conn *websocket.Conn) *MetricsConn {
	return &MetricsConn{conn: conn, m: &sync.Mutex{}}
}

// ReadMessage wraps the underlying ReadMessage with metrics
func (m *MetricsConn) ReadJSON(v interface{}) error {
	_, p, err := m.conn.ReadMessage()
	if err != nil {
		return err
	}
	WebSocketReadRequestCount.Inc()
	WebSocketMessageBytesRead.Observe(float64(len(p)))
	err = json.Unmarshal(p, v)
	if err != nil {
		return err
	}

	return nil
}

// WriteMessage wraps the underlying WriteMessage with metrics
func (m *MetricsConn) WriteJSON(v interface{}) error {
	m.m.Lock()
	defer m.m.Unlock()
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	err = m.conn.WriteMessage(websocket.TextMessage, data)
	if err == nil {
		WebSocketWriteRequestCount.Inc()
		WebSocketMessageBytesWritten.Observe(float64(len(data)))
	}
	return err
}

// Close delegates to the underlying connection
func (m *MetricsConn) Close() error {
	return m.conn.Close()
}

func readRoutine(conn *MetricsConn, h hub, conn_uid string) {
	h.connList.Store(conn_uid, conn)
	defer h.connList.Delete(conn_uid)

	WebSocketConnectionCount.Inc()
	defer WebSocketConnectionCount.Dec()

	for {
		ctx := context.TODO()
		var dest struct {
			Opcode string                 `json:"opcode"`
			Data   map[string]interface{} `json:"data"`
		}
		err := conn.ReadJSON(&dest)
		if err != nil {
			// fmt.Println("err: ", err)
			fmt.Println("closing connection due to error")
			err = conn.Close()
			if err != nil {
				// fmt.Println("error closing conn: ", err)
			}
			break
		}
		reqT := time.Now()
		// fmt.Println("dest: ", dest)

		switch dest.Opcode {
		case "list_chats":
			ListChatsOpCodeRequestCount.Inc()
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
			ListMessagesOpCodeRequestCount.Inc()
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
			ListUsersOpCodeRequestCount.Inc()
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
			SendMessageOpCodeRequestCount.Inc()
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
			CreateChatOpCodeRequestCount.Inc()
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
			AddUserOpCodeRequestCount.Inc()
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
			sys_msg_id, err := h.repo.CreateMessage(ctx, SYSTEM_UUID, chat_id, "User "+uid+" added", created_at)
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

			//resp2 := CreateMessageResponse{Opcode: "message_sent", ChatID: chat_id, Text: "User " + uid + " added", UserID: SYSTEM_UUID, CreatedAtMicro: time.Now().UnixMicro(), MsgID: sys_msg_id}
			//err = conn.WriteJSON(resp2)
			//if err != nil {
			//	fmt.Println("err: ", err)
			//	continue
			//}

			err = notifyAboutMessage(ctx, h, chat_id, "User "+uid+" added", created_at, sys_msg_id, SYSTEM_UUID)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}

			err = notifyAboutUserList(ctx, h, chat_id, SYSTEM_UUID, uid)
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

		duration := time.Since(reqT)
		WebSocketRequestDuration.Observe(float64(duration.Milliseconds()))
	}
}

func notifyAboutChatList(h hub, chat_id string, uid string, lastActivity int64) error {
	userConnInterface, ok := h.connList.Load(uid)
	if !ok {
		return nil
	}

	userConn, ok := userConnInterface.(*MetricsConn)
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

		userConn, ok := userConnInterface.(*MetricsConn)
		if !ok {
			fmt.Println("failed to cast user conn: ", user_id)
			continue
		}

		resp := UserListNotification{Opcode: "user_list_notification", ChatID: chat_id, NewUserID: new_uid}
		err = userConn.WriteJSON(resp)
		if err != nil {
			// fmt.Println("err notifying user: ", err)
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

		userConn, ok := userConnInterface.(*MetricsConn)
		if !ok {
			fmt.Println("failed to cast user conn: ", user_id)
			continue
		}

		resp := MessageNotification{Opcode: "message_notification", ChatID: chat_id, Text: text, UserID: origin_uid, CreatedAtMicro: created_at, MsgID: msg_id}
		err = userConn.WriteJSON(resp)
		if err != nil {
			// fmt.Println("err notifying user: ", err)
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
