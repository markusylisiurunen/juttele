package repo

import (
	"context"
	"time"
)

type ListChatsResult struct {
	Items []struct {
		ID        int64
		CreatedAt time.Time
		Title     string
	}
}

func (r *Repository) ListChats(ctx context.Context) (ListChatsResult, error) {
	var query = `
	select chat_id, chat_created_at, chat_title
	from chats
	order by chat_created_at asc
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return ListChatsResult{}, err
	}
	defer rows.Close()
	items := make([]struct {
		ID        int64
		CreatedAt time.Time
		Title     string
	}, 0)
	for rows.Next() {
		var createdAt string
		var item struct {
			ID        int64
			CreatedAt time.Time
			Title     string
		}
		if err := rows.Scan(&item.ID, &createdAt, &item.Title); err != nil {
			return ListChatsResult{}, err
		}
		item.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return ListChatsResult{}, err
		}
		items = append(items, item)
	}
	return ListChatsResult{items}, nil
}
