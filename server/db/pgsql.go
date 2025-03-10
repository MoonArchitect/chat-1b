package dbrepo

import (
	"context"
	"fmt"
	"slices"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"
)

var psql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

type pgsqlRepository struct {
	db MetricsDb
}

func NewPgsqlRepository(db *sqlx.DB) IDbRepo {
	return pgsqlRepository{
		db: MetricsDb{db: db},
	}
}

// chat-id  user-id
// msg-id   chat-id  text  timestamp[DESC]  user-id

// list chats -> get all chat-id by user-id
// get users in a chat -> list user-id by chat-id
// paginate messages from a chat -> get messages by chat-id

// TODO: rowSplit is not implemented, hence broken, hence stats returned are inflated
func (r pgsqlRepository) NumberOfUsers(ctx context.Context, rowSplit int) (int, error) {
	var count int
	err := r.db.GetContext(ctx, "count_users", &count, "SELECT COUNT(DISTINCT user_id) FROM chats")
	if err != nil {
		return 0, err
	}
	return count, nil
}

// TODO: rowSplit is not implemented, hence broken, hence stats returned are inflated
func (r pgsqlRepository) NumberOfChats(ctx context.Context, rowSplit int) (int, error) {
	var count int
	err := r.db.GetContext(ctx, "count_chats", &count, "SELECT COUNT(DISTINCT chat_id) FROM chats")
	if err != nil {
		return 0, err
	}
	return count, nil
}

// TODO: rowSplit is not implemented, hence broken, hence stats returned are inflated
func (r pgsqlRepository) NumberOfMessages(ctx context.Context, rowSplit int) (int, error) {
	var count int
	// err := r.db.GetContext(ctx, "count_messages", &count, "SELECT COUNT(DISTINCT msg_id) FROM messages")
	err := r.db.GetContext(ctx, "count_messages", &count, "SELECT FLOOR((reltuples/relpages) * (pg_relation_size('messages') / (current_setting('block_size')::integer)))::integer FROM pg_class where relname = 'messages'")
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r pgsqlRepository) ListAllUsers(ctx context.Context) ([]UserListItem, error) {
	query, args, err := psql.
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

func (r pgsqlRepository) CreateChat(ctx context.Context, userId string) (string, error) {
	raw_id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	chat_id := raw_id.String()

	query, args, err := psql.
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

func (r pgsqlRepository) AddUser(ctx context.Context, chatId, userId string) error {
	query, args, err := psql.
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

func (r pgsqlRepository) CreateMessage(ctx context.Context, userId, chatId, text string, createdAtMicro int64) (string, error) {
	raw_id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	msg_id := raw_id.String()

	query, args, err := psql.
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

// 79551a0b-1232-4325-84d6-423f8b9ae249
// b1945990-9774-4841-87ea-af9f88283bde
// 392f8df61e8c

// 500

func (r pgsqlRepository) ListMessages(ctx context.Context, chatId string, page uint64) ([]MessageDB, error) {
	query, args, err := psql.
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

func (r pgsqlRepository) ListChats(ctx context.Context, userId string) ([]ChatWithLatestMessage, error) {
	subquery, args, err := squirrel.Select("chat_id").
		From(ChatTable).
		Where(squirrel.Eq{"user_id": userId}).
		ToSql()
	if err != nil {
		return nil, err
	}

	query, args, err := psql.
		Select("user_chats.chat_id", "MAX(COALESCE(messages.created_at, CAST(0 AS BIGINT))) as latest_message").
		From(MessageTable).
		JoinClause(fmt.Sprintf("RIGHT JOIN (%s) AS user_chats ON messages.chat_id = user_chats.chat_id", subquery), args...).
		GroupBy("user_chats.chat_id").
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

func (r pgsqlRepository) ListChatUsers(ctx context.Context, chatId string) ([]string, error) {
	query, args, err := psql.
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
