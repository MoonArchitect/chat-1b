package dbrepo

import "context"

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
