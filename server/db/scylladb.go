package dbrepo

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/scylladb/gocqlx"
	"github.com/scylladb/gocqlx/qb"
	"github.com/scylladb/gocqlx/table"
)

type scylladbRepository struct {
	db gocqlx.Session
}

func NewScylladbRepository(db gocqlx.Session) IDbRepo {
	return scylladbRepository{
		db: db,
	}
}

// type ChatDB struct {
// 	ChatID string `db:"chat_id"`
// 	UserID string `db:"user_id"`
// }

// type MessageDB struct {
// 	MsgID          string `db:"msg_id"`
// 	ChatID         string `db:"chat_id"`
// 	UserID         string `db:"user_id"`
// 	CreatedAtMicro int64  `db:"created_at"`
// 	Text           string `db:"text"`
// }

// type UserListItem struct {
// 	ID            string `db:"user_id"`
// 	NumberOfChats int    `db:"count"`
// }

var chatsMetadata = table.Metadata{
	Name:    "main_db.chats",
	Columns: []string{"chat_id", "users"},
	PartKey: []string{"chat_id"},
}

var usersMetadata = table.Metadata{
	Name:    "main_db.users",
	Columns: []string{"user_id", "chats"},
	PartKey: []string{"user_id"},
}

var messagesMetadata = table.Metadata{
	Name:    "main_db.messages",
	Columns: []string{"msg_id", "chat_id", "user_id", "created_at_micro", "text"},
	PartKey: []string{"chat_id", "msg_id"},
	SortKey: []string{"created_at_micro"},
}

var chatsTable = table.New(chatsMetadata)
var usersTable = table.New(usersMetadata)
var messagesTable = table.New(messagesMetadata)

type ChatDB struct {
	ChatID string   `db:"chat_id"`
	Users  []string `db:"users"`
}

type UserDB struct {
	UserID string           `db:"user_id"`
	Chats  map[string]int64 `db:"chats"`
}

type MessageDB struct {
	MsgID          string `db:"msg_id"`
	ChatID         string `db:"chat_id"`
	UserID         string `db:"user_id"`
	CreatedAtMicro int64  `db:"created_at_micro"`
	Text           string `db:"text"`
}

type UserListItem struct {
	ID            string `db:"user_id"`
	NumberOfChats int    `db:"count"`
}

// Format:
// start := time.Now()
// ... query code
// duration := time.Since(start)
// SqliteRequestDuration.WithLabelValues(TODO: method name here).Observe(float64(duration.Milliseconds()))

func (r scylladbRepository) CreateChat(ctx context.Context, userId string) (string, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		SqliteRequestDuration.WithLabelValues("CreateChat").Observe(float64(duration.Milliseconds()))
	}()

	chatId, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("CreateChat: %w", err)
	}

	// Insert new chat with initial user
	stmt1, _ := chatsTable.Insert()
	if err := r.db.Session.Query(stmt1, chatId.String(), []string{userId}).Exec(); err != nil {
		return "", fmt.Errorf("CreateChat: %w", err)
	}

	// Add chat to user's chat map
	stmt2 := "UPDATE main_db.users SET chats = chats + {? : ?} WHERE user_id = ?"
	if err := r.db.Session.Query(stmt2, chatId.String(), time.Now().UnixMicro(), userId).Exec(); err != nil {
		return "", fmt.Errorf("CreateChat: %w", err)
	}

	return chatId.String(), nil
}

func (r scylladbRepository) AddUser(ctx context.Context, chatId, userId string) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		SqliteRequestDuration.WithLabelValues("AddUser").Observe(float64(duration.Milliseconds()))
	}()

	// Add user to chat's user set
	stmt1 := "UPDATE main_db.chats SET users = users + ? WHERE chat_id = ?"
	if err := r.db.Session.Query(stmt1, []string{userId}, chatId).Exec(); err != nil {
		return fmt.Errorf("AddUser: %w", err)
	}

	// Add chat to user's chat map
	stmt2 := "UPDATE main_db.users SET chats = chats + {? : ?} WHERE user_id = ?"
	if err := r.db.Session.Query(stmt2, chatId, time.Now().UnixMicro(), userId).Exec(); err != nil {
		return fmt.Errorf("AddUser: %w", err)
	}

	return nil
}

func (r scylladbRepository) ListAllUsers(ctx context.Context) ([]UserListItem, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		SqliteRequestDuration.WithLabelValues("ListAllUsers").Observe(float64(duration.Milliseconds()))
	}()

	var users []UserDB
	q := r.db.Query("SELECT user_id, chats FROM main_db.users", []string{"user_id", "chats"})
	if err := q.SelectRelease(&users); err != nil {
		return nil, fmt.Errorf("ListAllUsers: %w", err)
	}
	var res []UserListItem
	for _, user := range users {
		res = append(res, UserListItem{
			ID:            user.UserID,
			NumberOfChats: len(user.Chats),
		})
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].NumberOfChats > res[j].NumberOfChats // DESC order
	})
	return res, nil
}

func (r scylladbRepository) ListChats(ctx context.Context, userId string) ([]ChatWithLatestMessage, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		SqliteRequestDuration.WithLabelValues("ListChats").Observe(float64(duration.Milliseconds()))
	}()

	var user UserDB
	stmt, names := qb.Select(usersMetadata.Name).
		Columns(usersMetadata.Columns...).
		Where(qb.Eq("user_id")).
		ToCql()

	q := r.db.Query(stmt, names).BindMap(qb.M{"user_id": userId})
	if err := q.GetRelease(&user); err != nil {
		return nil, fmt.Errorf("ListChats: %w", err)
	}

	chatList := make([]ChatWithLatestMessage, 0, len(user.Chats))
	for chatId, lastActivity := range user.Chats {
		chatList = append(chatList, ChatWithLatestMessage{
			ChatID:          chatId,
			LatestMessageAt: lastActivity,
		})
	}
	return chatList, nil
}

func (r scylladbRepository) ListChatUsers(ctx context.Context, chatId string) ([]string, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		SqliteRequestDuration.WithLabelValues("ListChatUsers").Observe(float64(duration.Milliseconds()))
	}()

	var chat ChatDB
	stmt, names := qb.Select(chatsMetadata.Name).
		Columns(chatsMetadata.Columns...).
		Where(qb.Eq("chat_id")).
		ToCql()

	q := r.db.Query(stmt, names).BindMap(qb.M{"chat_id": chatId})
	if err := q.GetRelease(&chat); err != nil {
		return nil, fmt.Errorf("ListChatUsers: %w", err)
	}

	users := make([]string, len(chat.Users))
	for i, userId := range chat.Users {
		users[i] = userId
	}
	return users, nil
}

func (r scylladbRepository) CreateMessage(ctx context.Context, userId, chatId, text string, createdAtMicro int64) (string, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		SqliteRequestDuration.WithLabelValues("CreateMessage").Observe(float64(duration.Milliseconds()))
	}()

	msgId, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("CreateMessage: %w", err)
	}

	row := MessageDB{
		MsgID:          msgId.String(),
		ChatID:         chatId,
		UserID:         userId,
		CreatedAtMicro: createdAtMicro,
		Text:           text,
	}

	q := r.db.Query(messagesTable.Insert()).BindStruct(row)
	if err := q.ExecRelease(); err != nil {
		return "", fmt.Errorf("CreateMessage: %w", err)
	}

	return msgId.String(), nil
}

func (r scylladbRepository) ListMessages(ctx context.Context, chatId string, page uint64) ([]MessageDB, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		SqliteRequestDuration.WithLabelValues("ListMessages").Observe(float64(duration.Milliseconds()))
	}()

	var msgs []MessageDB
	stmt, names := qb.Select(messagesMetadata.Name).
		Columns(messagesMetadata.Columns...).
		Where(qb.Eq("chat_id")).
		ToCql()

	q := r.db.Query(stmt, names).BindMap(qb.M{"chat_id": chatId})
	if err := q.SelectRelease(&msgs); err != nil {
		return nil, fmt.Errorf("ListMessages: %w", err)
	}
	return msgs, nil
}

// const MIN_TOKEN_RANGE int64 = -9223372036854775807
// const MAX_TOKEN_RANGE int64 = 9223372036854775807

// func (r scylladbRepository) NumberOfUsers(ctx context.Context, rowSplit int) (int, error) {
// 	start := time.Now()
// 	defer func() {
// 		duration := time.Since(start)
// 		SqliteRequestDuration.WithLabelValues("NumberOfUsers").Observe(float64(duration.Milliseconds()))
// 	}()

// 	cql := fmt.Sprintf(`SELECT COUNT(*) FROM main_db.users WHERE token(id) >= -9204925292781066255 AND token(id) <= 9223372036854775807 USING TIMEOUT 1s BYPASS CACHE`)

// 	var count int
// 	if err := r.db.Session.Query(cql).Scan(&count); err != nil {
// 		return 0, fmt.Errorf("NumberOfUsers: %w", err)
// 	}
// 	return count, nil
// }

// func (r scylladbRepository) NumberOfChats(ctx context.Context, rowSplit int) (int, error) {
// 	start := time.Now()
// 	defer func() {
// 		duration := time.Since(start)
// 		SqliteRequestDuration.WithLabelValues("NumberOfChats").Observe(float64(duration.Milliseconds()))
// 	}()

// 	var count int
// 	if err := r.db.Session.Query(`SELECT COUNT(*) FROM main_db.chats`).Scan(&count); err != nil {
// 		return 0, fmt.Errorf("NumberOfChats: %w", err)
// 	}
// 	return count, nil
// }

// func (r scylladbRepository) NumberOfMessages(ctx context.Context, rowSplit int) (int, error) {
// 	start := time.Now()
// 	defer func() {
// 		duration := time.Since(start)
// 		SqliteRequestDuration.WithLabelValues("NumberOfMessages").Observe(float64(duration.Milliseconds()))
// 	}()

// 	var count int
// 	if err := r.db.Session.Query(`SELECT COUNT(*) FROM main_db.messages`).Scan(&count); err != nil {
// 		return 0, fmt.Errorf("NumberOfMessages: %w", err)
// 	}
// 	return count, nil
// }
