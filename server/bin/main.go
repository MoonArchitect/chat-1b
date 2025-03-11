package main

import (
	dbrepo "chat-1b/server/db"
	"chat-1b/server/utils/flatbuffer_types/WebsocketMessage"
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"reflect"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/gocql/gocql"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/google/uuid"
	"github.com/lesismal/nbio/nbhttp"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/scylladb/gocqlx"

	_ "net/http/pprof"
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

func onEcho(w http.ResponseWriter, r *http.Request) {
	// time.Sleep(time.Second * 5)
	data, _ := io.ReadAll(r.Body)
	if len(data) > 0 {
		w.Write(data)
	} else {
		w.Write([]byte(time.Now().Format("20060102 15:04:05")))
	}
}

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

	startWorkerPool()

	// add middleware to handle CORS
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	http.Handle("GET /metrics", promhttp.Handler())
	http.Handle("GET /user/create", corsMiddleware.Handler(http.HandlerFunc(createUserHandler)))
	http.Handle("GET /user/list", corsMiddleware.Handler(http.HandlerFunc(listUsersHandler(repo))))
	// http.Handle("/chat", corsMiddleware.Handler(http.HandlerFunc(t.wsHandler)))

	mux := &http.ServeMux{}
	mux.Handle("/chat", corsMiddleware.Handler(http.HandlerFunc(t.wsHandler)))

	engine := nbhttp.NewEngine(nbhttp.Config{
		Network:                 "tcp",
		Addrs:                   []string{":8081"},
		Handler:                 mux,
		MaxLoad:                 100000,
		ReleaseWebsocketPayload: true,
	})

	err = engine.Start()
	if err != nil {
		fmt.Printf("nbio.Start failed: %v\n", err)
		return
	}

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
		jsoniter.NewEncoder(w).Encode(users)
	}
}

var upgrader = newUpgrader()

func newUpgrader() *websocket.Upgrader {
	u := websocket.NewUpgrader()
	u.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	u.OnOpen(func(c *websocket.Conn) {
		// fmt.Println("OnOpen:", c.RemoteAddr().String())
		c.Session()
		WebSocketConnectionCount.Inc()
	})

	u.OnMessage(func(c *websocket.Conn, messageType websocket.MessageType, data []byte) {
		// echo
		// fmt.Println("OnMessage:", messageType, string(data))
		// c.WriteMessage(messageType, data)
		sess := c.Session()
		wsSess, ok := sess.(WebsocketSession)
		if !ok {
			fmt.Printf("ERROR: failed to convert session into WebsocketSession")
			return
		}

		readRoutine(wsSess.metricsConn, *wsSess.h, data)
	})

	u.OnClose(func(c *websocket.Conn, err error) {
		// fmt.Println("OnClose:", c.RemoteAddr().String(), err)
		WebSocketConnectionCount.Dec()

		sess := c.Session()
		wsSess, ok := sess.(WebsocketSession)
		if !ok {
			fmt.Printf("ERROR: failed to convert session into WebsocketSession")
			return
		}
		wsSess.h.connList.Delete(wsSess.uid)
	})

	return u
}

type hub struct {
	repo     dbrepo.IDbRepo
	connList *sync.Map
}

type WebsocketSession struct {
	h           *hub
	metricsConn *MetricsConn
	uid         string
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

	metricsConn := NewMetricsConn(conn, uid)
	conn.SetSession(WebsocketSession{h: &h, uid: uid, metricsConn: metricsConn})
	h.connList.Store(uid, metricsConn)
	ChatEndpointRequestCount.Inc()
}

type ListChatsResponse struct {
	Opcode string
	Chats  []dbrepo.ChatWithLatestMessage // todo don't use dbrepo types in api
}

type ListMessagesResponse struct {
	Opcode   string
	ChatID   string
	Messages []dbrepo.MessageDB
}

type CreateChatResponse struct {
	Opcode string
	ChatID string
}

type ListUsersResponse struct {
	Opcode string
	Users  []string
	ChatID string
}

type CreateMessageResponse struct {
	Opcode         string
	ChatID         string
	Text           string
	UserID         string
	CreatedAtMicro int64
	MsgID          string
}

type AddUserResponse struct {
	Opcode string
	ChatID string
	UserID string
}

type MessageNotification struct {
	Opcode         string
	ChatID         string
	Text           string
	UserID         string
	CreatedAtMicro int64
	MsgID          string
}

type UserListNotification struct {
	Opcode    string
	ChatID    string
	NewUserID string
}

type ChatListNotification struct {
	Opcode       string
	ChatID       string
	UserID       string
	LastActivity int64
}

// MetricsConn wraps a websocket.Conn with metrics
type MetricsConn struct {
	uid  string
	conn *websocket.Conn
	// m    *sync.Mutex
}

// NewMetricsConn creates a new MetricsConn
func NewMetricsConn(conn *websocket.Conn, uid string) *MetricsConn {
	return &MetricsConn{conn: conn, uid: uid}
	// return &MetricsConn{conn: conn, m: &sync.Mutex{}, uid: uid}
}

// ReadMessage wraps the underlying ReadMessage with metrics
func (m *MetricsConn) ReadMessage(p []byte, v interface{}) error {
	WebSocketReadRequestCount.Inc()
	WebSocketMessageBytesRead.Observe(float64(len(p)))

	msg := WebsocketMessage.GetRootAsMessage(p, 0)

	dest, ok := v.(*struct {
		Opcode string                 `json:"opcode"`
		Data   map[string]interface{} `json:"data"`
	})
	if !ok {
		return fmt.Errorf("invalid destination type")
	}

	switch msg.PayloadType() {
	case WebsocketMessage.PayloadSendMessageRequest:
		obj := new(WebsocketMessage.SendMessageRequest)
		t := obj.Table()
		msg.Payload(&t)
		obj.Init(t.Bytes, t.Pos)
		dest.Opcode = "send_message"
		dest.Data = map[string]interface{}{
			"uid":     string(obj.Uid()),
			"chat_id": string(obj.ChatId()),
			"text":    string(obj.Text()),
		}

	case WebsocketMessage.PayloadListChatsRequest:
		obj := new(WebsocketMessage.ListChatsRequest)
		t := obj.Table()
		msg.Payload(&t)
		obj.Init(t.Bytes, t.Pos)
		dest.Opcode = "list_chats"
		dest.Data = map[string]interface{}{
			"uid": string(obj.Uid()),
		}

	case WebsocketMessage.PayloadListMessagesRequest:
		obj := new(WebsocketMessage.ListMessagesRequest)
		t := obj.Table()
		msg.Payload(&t)
		obj.Init(t.Bytes, t.Pos)
		dest.Opcode = "list_messages"
		dest.Data = map[string]interface{}{
			"chat_id": string(obj.ChatId()),
			"page":    float64(obj.Page()),
		}

	case WebsocketMessage.PayloadListUsersRequest:
		obj := new(WebsocketMessage.ListUsersRequest)
		t := obj.Table()
		msg.Payload(&t)
		obj.Init(t.Bytes, t.Pos)
		dest.Opcode = "list_users"
		dest.Data = map[string]interface{}{
			"chat_id": string(obj.ChatId()),
		}

	case WebsocketMessage.PayloadCreateChatRequest:
		obj := new(WebsocketMessage.CreateChatRequest)
		t := obj.Table()
		msg.Payload(&t)
		obj.Init(t.Bytes, t.Pos)
		dest.Opcode = "create_chat"
		dest.Data = map[string]interface{}{
			"uid": string(obj.Uid()),
		}

	case WebsocketMessage.PayloadAddUserRequest:
		obj := new(WebsocketMessage.AddUserRequest)
		t := obj.Table()
		msg.Payload(&t)
		obj.Init(t.Bytes, t.Pos)
		dest.Opcode = "add_user"
		dest.Data = map[string]interface{}{
			"chat_id": string(obj.ChatId()),
			"user_id": string(obj.UserId()),
		}

	default:
		return fmt.Errorf("unknown payload type: %v", msg.PayloadType())
	}

	return nil
}

// WriteMessage wraps the underlying WriteMessage with metrics
// func (m *MetricsConn) WriteJSON(v interface{}) error {
// 	m.m.Lock()
// 	defer m.m.Unlock()
// 	data, err := jsoniter.Marshal(v)
// 	if err != nil {
// 		return err
// 	}
// 	err = m.conn.WriteMessage(websocket.TextMessage, data)
// 	if err == nil {
// 		WebSocketWriteRequestCount.Inc()
// 		WebSocketMessageBytesWritten.Observe(float64(len(data)))
// 	}
// 	return err
// }

func hash_uuid(uuid string) int {
	h := fnv.New32a()
	h.Write([]byte(uuid))
	return int(h.Sum32())
}

func (m *MetricsConn) SendMessageAsync(v interface{}) error {
	builder := flatbuffers.NewBuilder(1024)

	var offset flatbuffers.UOffsetT
	var payloadType WebsocketMessage.Payload

	switch msg := v.(type) {
	case ListChatsResponse:
		chats := make([]flatbuffers.UOffsetT, len(msg.Chats))
		for i, chat := range msg.Chats {
			chatID := builder.CreateString(chat.ChatID)
			WebsocketMessage.ChatWithLatestMessageStart(builder)
			WebsocketMessage.ChatWithLatestMessageAddChatId(builder, chatID)
			WebsocketMessage.ChatWithLatestMessageAddLatestMessageAt(builder, chat.LatestMessageAt)
			chats[i] = WebsocketMessage.ChatWithLatestMessageEnd(builder)
		}

		WebsocketMessage.ListChatsResponseStartChatsVector(builder, len(chats))
		for i := len(chats) - 1; i >= 0; i-- {
			builder.PrependUOffsetT(chats[i])
		}
		chatsVector := builder.EndVector(len(chats))

		WebsocketMessage.ListChatsResponseStart(builder)
		WebsocketMessage.ListChatsResponseAddChats(builder, chatsVector)
		offset = WebsocketMessage.ListChatsResponseEnd(builder)
		payloadType = WebsocketMessage.PayloadListChatsResponse

	case ListMessagesResponse:
		chatID := builder.CreateString(msg.ChatID)
		messages := make([]flatbuffers.UOffsetT, len(msg.Messages))

		for i, message := range msg.Messages {
			msgID := builder.CreateString(message.MsgID)
			msgChatID := builder.CreateString(message.ChatID)
			userID := builder.CreateString(message.UserID)
			text := builder.CreateString(message.Text)

			WebsocketMessage.MessageDBStart(builder)
			WebsocketMessage.MessageDBAddMsgId(builder, msgID)
			WebsocketMessage.MessageDBAddChatId(builder, msgChatID)
			WebsocketMessage.MessageDBAddUserId(builder, userID)
			WebsocketMessage.MessageDBAddText(builder, text)
			WebsocketMessage.MessageDBAddCreatedAtMicro(builder, message.CreatedAtMicro)
			messages[i] = WebsocketMessage.MessageDBEnd(builder)
		}

		WebsocketMessage.ListMessagesResponseStartMessagesVector(builder, len(messages))
		for i := len(messages) - 1; i >= 0; i-- {
			builder.PrependUOffsetT(messages[i])
		}
		messagesVector := builder.EndVector(len(messages))

		WebsocketMessage.ListMessagesResponseStart(builder)
		WebsocketMessage.ListMessagesResponseAddChatId(builder, chatID)
		WebsocketMessage.ListMessagesResponseAddMessages(builder, messagesVector)
		offset = WebsocketMessage.ListMessagesResponseEnd(builder)
		payloadType = WebsocketMessage.PayloadListMessagesResponse

	case ListUsersResponse:
		chatID := builder.CreateString(msg.ChatID)
		users := make([]flatbuffers.UOffsetT, len(msg.Users))
		for i, user := range msg.Users {
			users[i] = builder.CreateString(user)
		}

		WebsocketMessage.ListUsersResponseStartUsersVector(builder, len(users))
		for i := len(users) - 1; i >= 0; i-- {
			builder.PrependUOffsetT(users[i])
		}
		usersVector := builder.EndVector(len(users))

		WebsocketMessage.ListUsersResponseStart(builder)
		WebsocketMessage.ListUsersResponseAddUsers(builder, usersVector)
		WebsocketMessage.ListUsersResponseAddChatId(builder, chatID)
		offset = WebsocketMessage.ListUsersResponseEnd(builder)
		payloadType = WebsocketMessage.PayloadListUsersResponse

	case CreateChatResponse:
		chatID := builder.CreateString(msg.ChatID)
		WebsocketMessage.CreateChatResponseStart(builder)
		WebsocketMessage.CreateChatResponseAddChatId(builder, chatID)
		offset = WebsocketMessage.CreateChatResponseEnd(builder)
		payloadType = WebsocketMessage.PayloadCreateChatResponse

	case CreateMessageResponse:
		chatID := builder.CreateString(msg.ChatID)
		text := builder.CreateString(msg.Text)
		userID := builder.CreateString(msg.UserID)
		msgID := builder.CreateString(msg.MsgID)

		WebsocketMessage.CreateMessageResponseStart(builder)
		WebsocketMessage.CreateMessageResponseAddChatId(builder, chatID)
		WebsocketMessage.CreateMessageResponseAddText(builder, text)
		WebsocketMessage.CreateMessageResponseAddUserId(builder, userID)
		WebsocketMessage.CreateMessageResponseAddCreatedAtMicro(builder, msg.CreatedAtMicro)
		WebsocketMessage.CreateMessageResponseAddMsgId(builder, msgID)
		offset = WebsocketMessage.CreateMessageResponseEnd(builder)
		payloadType = WebsocketMessage.PayloadCreateMessageResponse

	case AddUserResponse:
		chatID := builder.CreateString(msg.ChatID)
		userID := builder.CreateString(msg.UserID)

		WebsocketMessage.AddUserResponseStart(builder)
		WebsocketMessage.AddUserResponseAddChatId(builder, chatID)
		WebsocketMessage.AddUserResponseAddUserId(builder, userID)
		offset = WebsocketMessage.AddUserResponseEnd(builder)
		payloadType = WebsocketMessage.PayloadAddUserResponse

	case MessageNotification:
		chatID := builder.CreateString(msg.ChatID)
		text := builder.CreateString(msg.Text)
		userID := builder.CreateString(msg.UserID)
		msgID := builder.CreateString(msg.MsgID)

		WebsocketMessage.MessageNotificationStart(builder)
		WebsocketMessage.MessageNotificationAddChatId(builder, chatID)
		WebsocketMessage.MessageNotificationAddText(builder, text)
		WebsocketMessage.MessageNotificationAddUserId(builder, userID)
		WebsocketMessage.MessageNotificationAddCreatedAtMicro(builder, msg.CreatedAtMicro)
		WebsocketMessage.MessageNotificationAddMsgId(builder, msgID)
		offset = WebsocketMessage.MessageNotificationEnd(builder)
		payloadType = WebsocketMessage.PayloadMessageNotification

	case UserListNotification:
		chatID := builder.CreateString(msg.ChatID)
		newUserID := builder.CreateString(msg.NewUserID)

		WebsocketMessage.UserListNotificationStart(builder)
		WebsocketMessage.UserListNotificationAddChatId(builder, chatID)
		WebsocketMessage.UserListNotificationAddNewUserId(builder, newUserID)
		offset = WebsocketMessage.UserListNotificationEnd(builder)
		payloadType = WebsocketMessage.PayloadUserListNotification

	case ChatListNotification:
		chatID := builder.CreateString(msg.ChatID)
		userID := builder.CreateString(msg.UserID)

		WebsocketMessage.ChatListNotificationStart(builder)
		WebsocketMessage.ChatListNotificationAddChatId(builder, chatID)
		WebsocketMessage.ChatListNotificationAddUserId(builder, userID)
		WebsocketMessage.ChatListNotificationAddLastActivity(builder, msg.LastActivity)
		offset = WebsocketMessage.ChatListNotificationEnd(builder)
		payloadType = WebsocketMessage.PayloadChatListNotification
	}

	WebsocketMessage.MessageStart(builder)
	WebsocketMessage.MessageAddPayloadType(builder, payloadType)
	WebsocketMessage.MessageAddPayload(builder, offset)
	messageOffset := WebsocketMessage.MessageEnd(builder)

	builder.Finish(messageOffset)
	bytes := builder.FinishedBytes()

	i := hash_uuid(m.uid) % N_WORKERS
	chPool[i] <- Task{
		conn:    m.conn,
		payload: bytes,
	}

	return nil
}

// Close delegates to the underlying connection
func (m *MetricsConn) Close() error {
	return m.conn.Close()
}

// on_open -> save conn pointer
// on_message
//  - read data
//  - perform business logic
//  - push task to notify other users
// writer_pool -> takes tasks and writes to connections

// task stream of (user_id, msg *byte)
// - if the stream has multiple task with the same user_id it should be processed by either the same goroutine or in a blocking fashion to avoid expensive sync between primitives
// -> have N task streams, task is put into stream based on hash(user_id) % N
// -> have M workers in a pool, each is responsible for N/M streams

const N_WORKERS = 10

var chPool []chan Task

type Task struct {
	conn    *websocket.Conn
	payload []byte
}

func startWorkerPool() []chan Task {
	for i := 0; i < N_WORKERS; i++ {
		ch := make(chan Task, 10000)
		chPool = append(chPool, ch)
		go websocketWriteWorker(ch)
	}

	go func() {
		for {
			for i := 0; i < N_WORKERS; i++ {
				fmt.Println("Channel length: ", len(chPool[i]))
			}
			time.Sleep(5 * time.Second)
		}
	}()

	return chPool
}

func websocketWriteWorker(ch <-chan Task) {
	for {
		select {
		case task := <-ch:
			{
				WebSocketWriteRequestCount.Inc()
				WebSocketMessageBytesWritten.Observe(float64(len(task.payload)))
				task.conn.WriteMessage(websocket.BinaryMessage, task.payload)
			}
		}
	}
}

func readRoutine(conn *MetricsConn, h hub, data []byte) {
	ctx := context.TODO()
	var dest struct {
		Opcode string                 `json:"opcode"`
		Data   map[string]interface{} `json:"data"`
	}
	err := conn.ReadMessage(data, &dest)
	if err != nil {
		// fmt.Println("err: ", err)
		fmt.Println("closing connection due to error")
		err = conn.Close()
		if err != nil {
			// fmt.Println("error closing conn: ", err)
		}
		// break
		return
	}
	reqT := time.Now()
	// fmt.Println("dest: ", dest)

	switch dest.Opcode {
	case "list_chats":
		ListChatsOpCodeRequestCount.Inc()
		uid, ok := dest.Data["uid"].(string)
		if !ok {
			fmt.Println("failed to read dest.Data: ", dest.Data)
			return
		}
		chatIds, err := h.repo.ListChats(ctx, uid)
		if err != nil {
			fmt.Println("err: ", err)
			return
		}
		resp := ListChatsResponse{Opcode: "chat_list", Chats: chatIds}

		err = conn.SendMessageAsync(resp)
		if err != nil {
			fmt.Println("err: ", err)
			return
		}
	case "list_messages":
		ListMessagesOpCodeRequestCount.Inc()
		chat_id, ok := dest.Data["chat_id"].(string)
		if !ok {
			fmt.Println("failed to read chat_id: ", dest.Data)
			return
		}
		page, ok := dest.Data["page"]
		if !ok {
			fmt.Println("failed to read page: ", dest.Data, "type: ", reflect.TypeOf(page))
			return
		}

		messages, err := h.repo.ListMessages(ctx, chat_id, uint64(page.(float64)))
		if err != nil {
			fmt.Println("err: ", err)
			return
		}

		resp := ListMessagesResponse{Opcode: "chat_messages", Messages: messages, ChatID: chat_id}

		err = conn.SendMessageAsync(resp)
		if err != nil {
			fmt.Println("err: ", err)
			return
		}
	case "list_users":
		ListUsersOpCodeRequestCount.Inc()
		chat_id, ok := dest.Data["chat_id"].(string)
		if !ok {
			fmt.Println("failed to read chat_id: ", dest.Data)
			return
		}
		users, err := h.repo.ListChatUsers(ctx, chat_id)
		if err != nil {
			fmt.Println("err: ", err)
			return
		}
		resp := ListUsersResponse{Opcode: "chat_users", Users: users, ChatID: chat_id}
		err = conn.SendMessageAsync(resp)
		if err != nil {
			fmt.Println("err: ", err)
			return
		}
	case "send_message":
		SendMessageOpCodeRequestCount.Inc()
		uid, ok := dest.Data["uid"].(string)
		if !ok {
			fmt.Println("failed to read uid: ", dest.Data)
			return
		}
		chat_id, ok := dest.Data["chat_id"].(string)
		if !ok {
			fmt.Println("failed to read chat_id: ", dest.Data)
			return
		}
		text, ok := dest.Data["text"].(string)
		if !ok {
			fmt.Println("failed to read text: ", dest.Data)
			return
		}
		created_at := time.Now().UnixMicro()
		msg_id, err := h.repo.CreateMessage(ctx, uid, chat_id, text, created_at)
		if err != nil {
			fmt.Println("err: ", err)
			return
		}

		resp := CreateMessageResponse{Opcode: "message_sent", ChatID: chat_id, Text: text, UserID: uid, CreatedAtMicro: created_at, MsgID: msg_id}
		err = conn.SendMessageAsync(resp)
		if err != nil {
			fmt.Println("err: ", err)
			return
		}

		err = notifyAboutMessage(ctx, h, chat_id, text, created_at, msg_id, uid)
		if err != nil {
			fmt.Println("err: ", err)
			return
		}

	case "create_chat":
		CreateChatOpCodeRequestCount.Inc()
		uid, ok := dest.Data["uid"].(string)
		if !ok {
			fmt.Println("failed to read uid: ", dest.Data)
			return
		}
		chat_id, err := h.repo.CreateChat(ctx, uid)
		if err != nil {
			fmt.Println("err: ", err)
			return
		}
		resp := CreateChatResponse{Opcode: "chat_created", ChatID: chat_id}
		err = conn.SendMessageAsync(resp)
		if err != nil {
			fmt.Println("err: ", err)
			return
		}
	case "add_user":
		AddUserOpCodeRequestCount.Inc()
		// notify all other users that a user was added, show this as a status event in chat
		chat_id, ok := dest.Data["chat_id"].(string)
		if !ok {
			fmt.Println("failed to read chat_id: ", dest.Data)
			return
		}
		uid, ok := dest.Data["user_id"].(string)
		if !ok {
			fmt.Println("failed to read user_id: ", dest.Data)
			return
		}
		err := h.repo.AddUser(ctx, chat_id, uid)
		if err != nil {
			fmt.Println("err: ", err)
			return
		}

		// add system message to chat
		created_at := time.Now().UnixMicro()
		sys_msg_id, err := h.repo.CreateMessage(ctx, SYSTEM_UUID, chat_id, "User "+uid+" added", created_at)
		if err != nil {
			fmt.Println("err: ", err)
			return
		}
		//resp := AddUserResponse{Opcode: "user_added", ChatID: chat_id, UserID: uid}
		//err = conn.SendMessageAsync(resp)
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
			return
		}

		err = notifyAboutUserList(ctx, h, chat_id, SYSTEM_UUID, uid)
		if err != nil {
			fmt.Println("err: ", err)
			return
		}

		err = notifyAboutChatList(h, chat_id, uid, created_at)
		if err != nil {
			fmt.Println("err: ", err)
			return
		}
	}

	duration := time.Since(reqT)
	WebSocketRequestDuration.Observe(float64(duration.Milliseconds()))
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
	err := userConn.SendMessageAsync(resp)
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
		err = userConn.SendMessageAsync(resp)
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
		err = userConn.SendMessageAsync(resp)
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
