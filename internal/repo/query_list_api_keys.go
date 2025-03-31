package repo

import (
	"context"
	"time"
)

type ListAPIKeysResult struct {
	Items []struct {
		CreatedAt time.Time
		ExpiresAt time.Time
		UUID      string
	}
}

func (r *Repository) ListAPIKeys(ctx context.Context) (ListAPIKeysResult, error) {
	var deleteExpiredQuery = `
	delete from api_keys
	where api_key_expires_at < ?
	`
	_, err := r.db.ExecContext(ctx, deleteExpiredQuery,
		time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return ListAPIKeysResult{}, err
	}
	var listQuery = `
	select api_key_created_at, api_key_expires_at, api_key_uuid
	from api_keys
	`
	rows, err := r.db.QueryContext(ctx, listQuery)
	if err != nil {
		return ListAPIKeysResult{}, err
	}
	defer rows.Close()
	items := make([]struct {
		CreatedAt time.Time
		ExpiresAt time.Time
		UUID      string
	}, 0)
	for rows.Next() {
		var createdAt string
		var expiresAt string
		var item struct {
			CreatedAt time.Time
			ExpiresAt time.Time
			UUID      string
		}
		if err := rows.Scan(&createdAt, &expiresAt, &item.UUID); err != nil {
			return ListAPIKeysResult{}, err
		}
		item.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return ListAPIKeysResult{}, err
		}
		item.ExpiresAt, err = time.Parse(time.RFC3339Nano, expiresAt)
		if err != nil {
			return ListAPIKeysResult{}, err
		}
		items = append(items, item)
	}
	return ListAPIKeysResult{items}, nil
}
