package repo

import (
	"context"
	"time"
)

type CreateAPIKeyArgs struct {
	ExpiresIn time.Duration
	UUID      string
}

func (r *Repository) CreateAPIKey(ctx context.Context, args CreateAPIKeyArgs) error {
	now := time.Now().UTC()
	var query = `
	insert into api_keys (api_key_created_at, api_key_expires_at, api_key_uuid)
	values (?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		now.Format(time.RFC3339Nano), now.Add(args.ExpiresIn).Format(time.RFC3339Nano), args.UUID)
	return err
}
