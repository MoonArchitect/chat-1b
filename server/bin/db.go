package main

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	_ "github.com/mattn/go-sqlite3"
)

const (
	ChatTable    = "chats"
	MessageTable = "messages"
)

type dbRepo struct {
	db MetricsDb
}

type IDbRepo interface {
	CreateChat(ctx context.Context, userId string) (string, error)
	AddUser(ctx context.Context, chatId, userId string) error
	ListAllUsers(ctx context.Context) ([]UserListItem, error)
	ListChats(ctx context.Context, userId string) ([]ChatWithLatestMessage, error)
	ListChatUsers(ctx context.Context, chatId string) ([]string, error)
	CreateMessage(ctx context.Context, userId, chatId, text string, createdAtMicro int64) (string, error)
	ListMessages(ctx context.Context, chatId string, page uint64) ([]MessageDB, error)
	NumberOfUsers(ctx context.Context) (int, error)
	NumberOfChats(ctx context.Context) (int, error)
	NumberOfMessages(ctx context.Context) (int, error)
}

type MetricsDb struct {
	db *sqlx.DB
}

func (m MetricsDb) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	start := time.Now()
	resp := m.db.GetContext(ctx, dest, query, args...)
	duration := time.Since(start)
	SqliteRequestDuration.Observe(float64(duration.Milliseconds()))
	return resp
}

func (m MetricsDb) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	resp, err := m.db.ExecContext(ctx, query, args...)
	duration := time.Since(start)
	SqliteRequestDuration.Observe(float64(duration.Milliseconds()))
	return resp, err
}

func (m MetricsDb) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	start := time.Now()
	err := m.db.SelectContext(ctx, dest, query, args...)
	duration := time.Since(start)
	SqliteRequestDuration.Observe(float64(duration.Milliseconds()))
	return err
}

func newDbRepo(db *sqlx.DB) IDbRepo {
	return dbRepo{
		db: MetricsDb{db: db},
	}
}

// chat-id  user-id
// msg-id   chat-id  text  timestamp[DESC]  user-id

// list chats -> get all chat-id by user-id
// get users in a chat -> list user-id by chat-id
// paginate messages from a chat -> get messages by chat-id

func (r dbRepo) NumberOfUsers(ctx context.Context) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(DISTINCT user_id) FROM chats")
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r dbRepo) NumberOfChats(ctx context.Context) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(DISTINCT chat_id) FROM chats")
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r dbRepo) NumberOfMessages(ctx context.Context) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(DISTINCT msg_id) FROM messages")
	if err != nil {
		return 0, err
	}
	return count, nil
}

type ChatDB struct {
	ChatID string `db:"chat_id"`
	UserID string `db:"user_id"`
}

type MessageDB struct {
	MsgID          string `db:"msg_id"`
	ChatID         string `db:"chat_id"`
	UserID         string `db:"user_id"`
	CreatedAtMicro int64  `db:"created_at"`
	Text           string `db:"text"`
}

type UserListItem struct {
	ID            string `db:"user_id"`
	NumberOfChats int    `db:"count"`
}

func (r dbRepo) ListAllUsers(ctx context.Context) ([]UserListItem, error) {
	query, args, err := squirrel.
		Select("user_id", "COUNT(*) AS count").
		From(ChatTable).
		GroupBy("user_id").
		OrderBy("count DESC").
		ToSql()
	if err != nil {
		return nil, err
	}

	var res []UserListItem
	err = r.db.SelectContext(ctx, &res, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list all users: %w", err)
	}

	return res, nil
}

func (r dbRepo) CreateChat(ctx context.Context, userId string) (string, error) {
	raw_id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	chat_id := raw_id.String()

	query, args, err := squirrel.
		Insert(ChatTable).
		Columns(
			"chat_id",
			"user_id",
		).
		Values(
			chat_id,
			userId,
		).
		ToSql()
	if err != nil {
		return "", err
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("Failed to insert entry: %w", err)
	}

	return chat_id, nil
}

func (r dbRepo) AddUser(ctx context.Context, chatId, userId string) error {
	query, args, err := squirrel.
		Insert(ChatTable).
		Columns(
			"chat_id",
			"user_id",
		).
		Values(
			chatId,
			userId,
		).
		ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("Failed to insert entry: %w", err)
	}

	return nil
}

func (r dbRepo) CreateMessage(ctx context.Context, userId, chatId, text string, createdAtMicro int64) (string, error) {
	raw_id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	msg_id := raw_id.String()

	query, args, err := squirrel.
		Insert(MessageTable).
		Columns(
			"msg_id",
			"chat_id",
			"user_id",
			"created_at",
			"text",
		).
		Values(
			msg_id,
			chatId,
			userId,
			createdAtMicro,
			text,
		).
		ToSql()
	if err != nil {
		return "", err
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("Failed to insert entry: %w", err)
	}

	return msg_id, nil
}

const MESSAGE_PAGE_SIZE uint64 = 50

func (r dbRepo) ListMessages(ctx context.Context, chatId string, page uint64) ([]MessageDB, error) {
	query, args, err := squirrel.
		Select("*").
		From(MessageTable).
		Where(squirrel.Eq{"chat_id": chatId}).
		OrderBy("created_at DESC").
		Offset(MESSAGE_PAGE_SIZE * page).
		Limit(MESSAGE_PAGE_SIZE).
		ToSql()
	if err != nil {
		return nil, err
	}

	var res []MessageDB
	err = r.db.SelectContext(ctx, &res, query, args...)
	if err != nil {
		return nil, fmt.Errorf("Failed to insert entry: %w", err)
	}

	slices.Reverse(res)

	return res, err
}

type ChatWithLatestMessage struct {
	ChatID          string `db:"chat_id"`
	LatestMessageAt int64  `db:"latest_message"`
}

func (r dbRepo) ListChats(ctx context.Context, userId string) ([]ChatWithLatestMessage, error) {
	subquery, args, err := squirrel.Select("chat_id", "MAX(created_at) as latest_message").
		From(MessageTable).
		GroupBy("chat_id").
		ToSql()
	if err != nil {
		return nil, err
	}

	query, args, err := squirrel.
		Select("chats.chat_id", "IFNULL(latest_messages.latest_message, 0) as latest_message").
		From(ChatTable).
		JoinClause(fmt.Sprintf("LEFT JOIN (%s) AS latest_messages ON chats.chat_id = latest_messages.chat_id", subquery), args...).
		Where(squirrel.Eq{"user_id": userId}).
		OrderBy("latest_message DESC").
		ToSql()
	if err != nil {
		return nil, err
	}

	var res []ChatWithLatestMessage
	err = r.db.SelectContext(ctx, &res, query, args...)
	if err != nil {
		return nil, fmt.Errorf("Failed to list chats: %w", err)
	}

	return res, nil
}

func (r dbRepo) ListChatUsers(ctx context.Context, chatId string) ([]string, error) {
	query, args, err := squirrel.
		Select("user_id").
		From(ChatTable).
		Where(squirrel.Eq{"chat_id": chatId}).
		ToSql()
	if err != nil {
		return nil, err
	}

	var res []string
	err = r.db.SelectContext(ctx, &res, query, args...)
	if err != nil {
		return nil, fmt.Errorf("Failed to insert entry: %w", err)
	}

	return res, err
}
