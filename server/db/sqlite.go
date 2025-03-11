package dbrepo

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

type sqliteRepository struct {
	db MetricsDb
}

type MetricsDb struct {
	db *sqlx.DB
}

func (m MetricsDb) GetContext(ctx context.Context, operation string, dest interface{}, query string, args ...interface{}) error {
	start := time.Now()
	resp := m.db.GetContext(ctx, dest, query, args...)
	duration := time.Since(start)
	SqliteRequestDuration.WithLabelValues(operation).Observe(float64(duration.Milliseconds()))
	return resp
}

func (m MetricsDb) ExecContext(ctx context.Context, operation string, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	resp, err := m.db.ExecContext(ctx, query, args...)
	duration := time.Since(start)
	SqliteRequestDuration.WithLabelValues(operation).Observe(float64(duration.Milliseconds()))
	return resp, err
}

func (m MetricsDb) SelectContext(ctx context.Context, operation string, dest interface{}, query string, args ...interface{}) error {
	start := time.Now()
	err := m.db.SelectContext(ctx, dest, query, args...)
	duration := time.Since(start)
	SqliteRequestDuration.WithLabelValues(operation).Observe(float64(duration.Milliseconds()))
	return err
}

func NewSqliteRepository(db *sqlx.DB) IDbRepo {
	return sqliteRepository{
		db: MetricsDb{db: db},
	}
}

// chat-id  user-id
// msg-id   chat-id  text  timestamp[DESC]  user-id

// list chats -> get all chat-id by user-id
// get users in a chat -> list user-id by chat-id
// paginate messages from a chat -> get messages by chat-id

// TODO: rowSplit is not implemented, hence broken, hence stats returned are inflated
func (r sqliteRepository) NumberOfUsers(ctx context.Context, rowSplit int) (int, error) {
	var count int
	err := r.db.GetContext(ctx, "count_users", &count, "SELECT COUNT(DISTINCT user_id) FROM chats")
	if err != nil {
		return 0, err
	}
	return count, nil
}

// TODO: rowSplit is not implemented, hence broken, hence stats returned are inflated
func (r sqliteRepository) NumberOfChats(ctx context.Context, rowSplit int) (int, error) {
	var count int
	err := r.db.GetContext(ctx, "count_chats", &count, "SELECT COUNT(DISTINCT chat_id) FROM chats")
	if err != nil {
		return 0, err
	}
	return count, nil
}

// TODO: rowSplit is not implemented, hence broken, hence stats returned are inflated
func (r sqliteRepository) NumberOfMessages(ctx context.Context, rowSplit int) (int, error) {
	var count int
	err := r.db.GetContext(ctx, "count_messages", &count, "SELECT COUNT(DISTINCT msg_id) FROM messages")
	if err != nil {
		return 0, err
	}
	return count, nil
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

func (r sqliteRepository) ListAllUsers(ctx context.Context) ([]UserListItem, error) {
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
	err = r.db.SelectContext(ctx, "list_all_users", &res, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list all users: %w", err)
	}

	return res, nil
}

func (r sqliteRepository) CreateChat(ctx context.Context, userId string) (string, error) {
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

	_, err = r.db.ExecContext(ctx, "create_chat", query, args...)
	if err != nil {
		return "", fmt.Errorf("Failed to insert entry: %w", err)
	}

	return chat_id, nil
}

func (r sqliteRepository) AddUser(ctx context.Context, chatId, userId string) error {
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

	_, err = r.db.ExecContext(ctx, "add_user", query, args...)
	if err != nil {
		return fmt.Errorf("Failed to insert entry: %w", err)
	}

	return nil
}

func (r sqliteRepository) CreateMessage(ctx context.Context, userId, chatId, text string, createdAtMicro int64) (string, error) {
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

	_, err = r.db.ExecContext(ctx, "create_message", query, args...)
	if err != nil {
		return "", fmt.Errorf("Failed to insert entry: %w", err)
	}

	return msg_id, nil
}

const MESSAGE_PAGE_SIZE uint64 = 50

func (r sqliteRepository) ListMessages(ctx context.Context, chatId string, page uint64) ([]MessageDB, error) {
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
	err = r.db.SelectContext(ctx, "list_messages", &res, query, args...)
	if err != nil {
		return nil, fmt.Errorf("Failed to insert entry: %w", err)
	}

	slices.Reverse(res)

	return res, err
}

type ChatWithLatestMessage struct {
	ChatID          string `db:"chat_id"`
	LatestMessageAt int64  `db:"latest_message_at"`
}

func (r sqliteRepository) ListChats(ctx context.Context, userId string) ([]ChatWithLatestMessage, error) {
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
	err = r.db.SelectContext(ctx, "list_chats", &res, query, args...)
	if err != nil {
		return nil, fmt.Errorf("Failed to list chats: %w", err)
	}

	return res, nil
}

func (r sqliteRepository) ListChatUsers(ctx context.Context, chatId string) ([]string, error) {
	query, args, err := squirrel.
		Select("user_id").
		From(ChatTable).
		Where(squirrel.Eq{"chat_id": chatId}).
		ToSql()
	if err != nil {
		return nil, err
	}

	var res []string
	err = r.db.SelectContext(ctx, "list_chat_users", &res, query, args...)
	if err != nil {
		return nil, fmt.Errorf("Failed to insert entry: %w", err)
	}

	return res, err
}
