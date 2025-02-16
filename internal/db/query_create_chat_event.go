package db

import (
	"context"
	"encoding/json"
	"time"
)

type CreateChatEventArgs struct {
	ChatID  int64
	Kind    string
	Content json.RawMessage
}

func (db *DB) CreateChatEvent(ctx context.Context, args CreateChatEventArgs) (int64, error) {
	var query = `
	insert into chat_events (chat_id, chat_event_created_at, chat_event_kind, chat_event_content)
	values (?, ?, ?, ?)
	`
	res, err := db.ExecContext(ctx, query,
		args.ChatID, time.Now().UTC().Format(time.RFC3339), args.Kind, args.Content)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
