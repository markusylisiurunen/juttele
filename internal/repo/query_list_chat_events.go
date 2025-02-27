package repo

import (
	"context"
	"encoding/json"
	"time"
)

type ListChatEventsArgs struct {
	ChatID int64
}

type ListChatEventsResult struct {
	Items []struct {
		CreatedAt time.Time
		UUID      string
		Kind      string
		Content   json.RawMessage
	}
}

func (r *Repository) ListChatEvents(ctx context.Context, args ListChatEventsArgs) (ListChatEventsResult, error) {
	var query = `
	select chat_event_created_at, chat_event_uuid, chat_event_kind, chat_event_content
	from chat_events
	where chat_id = ?
	order by chat_event_created_at asc, chat_event_id asc
	`
	rows, err := r.db.QueryContext(ctx, query, args.ChatID)
	if err != nil {
		return ListChatEventsResult{}, err
	}
	defer rows.Close()
	items := make([]struct {
		CreatedAt time.Time
		UUID      string
		Kind      string
		Content   json.RawMessage
	}, 0)
	for rows.Next() {
		var createdAt string
		var item struct {
			CreatedAt time.Time
			UUID      string
			Kind      string
			Content   json.RawMessage
		}
		if err := rows.Scan(&createdAt, &item.UUID, &item.Kind, &item.Content); err != nil {
			return ListChatEventsResult{}, err
		}
		item.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return ListChatEventsResult{}, err
		}
		items = append(items, item)
	}
	return ListChatEventsResult{items}, nil
}
