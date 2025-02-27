package repo

import (
	"context"
	"encoding/json"
	"time"
)

type CreateChatEventArgs struct {
	ChatID  int64
	UUID    string
	Kind    string
	Content json.RawMessage
}

func (r *Repository) CreateChatEvent(ctx context.Context, args CreateChatEventArgs) (int64, error) {
	var query = `
	insert into chat_events (chat_id, chat_event_created_at, chat_event_uuid, chat_event_kind, chat_event_content)
	values (?, ?, ?, ?, ?)
	on conflict (chat_id, chat_event_uuid) do update set
		chat_event_created_at = excluded.chat_event_created_at,
		chat_event_kind = excluded.chat_event_kind,
		chat_event_content = excluded.chat_event_content
	`
	res, err := r.db.ExecContext(ctx, query,
		args.ChatID, time.Now().UTC().Format(time.RFC3339Nano), args.UUID, args.Kind, args.Content)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
