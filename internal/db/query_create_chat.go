package db

import (
	"context"
	"time"
)

type CreateChatArgs struct {
	Title string
}

func (db *DB) CreateChat(ctx context.Context, args CreateChatArgs) (int64, error) {
	var query = `
	insert into chats (chat_created_at, chat_title, chat_pinned)
	values (?, ?, ?)
	`
	res, err := db.ExecContext(ctx, query,
		time.Now().UTC().Format(time.RFC3339), args.Title, false)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
