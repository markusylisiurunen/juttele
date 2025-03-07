package repo

import (
	"context"
)

type DeleteChatEventArgs struct {
	ID string
}

func (r *Repository) DeleteChatEvent(ctx context.Context, args DeleteChatEventArgs) error {
	var query = `
	delete from chat_events
	where chat_event_uuid = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		args.ID)
	if err != nil {
		return err
	}
	return nil
}
