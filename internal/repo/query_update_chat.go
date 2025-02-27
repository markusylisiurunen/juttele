package repo

import (
	"context"
)

type UpdateChatArgs struct {
	ID    int64
	Title string
}

func (r *Repository) UpdateChat(ctx context.Context, args UpdateChatArgs) error {
	var updateQuery = `
	update chats
	set chat_title = ?
	where chat_id = ?
	`
	_, err := r.db.ExecContext(ctx, updateQuery,
		args.Title, args.ID)
	if err != nil {
		return err
	}
	return nil
}
