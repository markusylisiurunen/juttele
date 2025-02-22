package repo

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"path"
	"sort"
	"time"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func Migrate(ctx context.Context, db *sql.DB) error {
	var createSchemaVersionsQuery = `
	create table if not exists schema_versions (
		version integer primary key,
		applied_at text not null
	)
	`
	if _, err := db.ExecContext(ctx, createSchemaVersionsQuery); err != nil {
		return fmt.Errorf("error creating schema_versions table: %w", err)
	}
	files, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("error reading migration files: %w", err)
	}
	type migration struct {
		version  int
		filename string
	}
	migrations := make([]migration, 0, len(files))
	for _, file := range files {
		var version int
		if _, err := fmt.Sscanf(file.Name(), "%05d_", &version); err != nil {
			return fmt.Errorf("invalid migration file %q: %w", file.Name(), err)
		}
		migrations = append(migrations, migration{version, file.Name()})
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})
	var getCurrentVersionQuery = `
	select coalesce(max(version), 0) from schema_versions
	`
	var currentVersion int
	if err := db.QueryRowContext(ctx, getCurrentVersionQuery).Scan(&currentVersion); err != nil {
		return fmt.Errorf("error getting current version: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()
	for _, migration := range migrations {
		if migration.version <= currentVersion {
			continue
		}
		sql, err := migrationFiles.ReadFile(path.Join("migrations", migration.filename))
		if err != nil {
			return fmt.Errorf("error reading migration %d: %w", migration.version, err)
		}
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return fmt.Errorf("error applying migration %d: %w", migration.version, err)
		}
		var insertVersionQuery = `
		insert into schema_versions (version, applied_at) values (?, ?)
		`
		if _, err := tx.ExecContext(ctx, insertVersionQuery,
			migration.version, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return fmt.Errorf("error updating schema_versions: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}
	return nil
}
