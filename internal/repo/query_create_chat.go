package repo

import (
	"context"
	"time"
)

type CreateChatArgs struct {
	Title string
}

func (r *Repository) CreateChat(ctx context.Context, args CreateChatArgs) (int64, error) {
	var deleteEmptyQuery = `
	delete from chats
	where not exists (
		select 1 from chat_events
		where
			chat_events.chat_id = chats.chat_id
			and chat_event_kind like 'message.%'
	)
	`
	_, err := r.db.ExecContext(ctx, deleteEmptyQuery)
	if err != nil {
		return 0, err
	}
	var createNewQuery = `
	insert into chats (chat_created_at, chat_title, chat_pinned)
	values (?, ?, ?)
	`
	res, err := r.db.ExecContext(ctx, createNewQuery,
		time.Now().UTC().Format(time.RFC3339Nano), args.Title, false)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
