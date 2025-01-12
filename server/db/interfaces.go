package dbrepo

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

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

var SqliteRequestDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "sqlite_request_duration",
		Help:    "Duration of SQLite requests by operation",
		Buckets: []float64{5, 10, 15, 25, 30, 35, 40, 45, 50, 100, 200, 300, 400, 500, 700, 1000, 2000, 3000, 4000, 5000, 10000, 50000, 100000, 500000, 1000000},
	},
	[]string{"operation"},
)
