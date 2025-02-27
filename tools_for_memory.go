package juttele

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

func NewMemoryToolBundle(dataFolder string) []Tool {
	client, err := sql.Open("sqlite3",
		fmt.Sprintf("file:%s/memories.db?_fk=1", dataFolder))
	if err != nil {
		panic(err)
	}
	return []Tool{
		&toolSaveMemory{db: client},
		&toolUpdateMemory{db: client},
		&toolDeleteMemory{db: client},
		&toolListMemories{db: client},
	}
}

//---

type toolSaveMemory struct {
	db *sql.DB
}

func (t *toolSaveMemory) Name() string {
	return "save_memory"
}

func (t *toolSaveMemory) Spec() []byte {
	var spec = `
{
	"name": "save_memory",
	"description": "Save a memory for the user. Only to be used when the user explicitly asks to save a memory.",
	"parameters": {
		"type": "object",
		"properties": {
			"content": {
				"type": "string",
				"description": "The memory content to save."
			}
		},
		"required": ["content"],
		"additionalProperties": false
	},
	"strict": true
}
	`
	return []byte(strings.TrimSpace(spec))
}

func (t *toolSaveMemory) Call(ctx context.Context, args string) (string, error) {
	if err := migrateMemoryTools(t.db); err != nil {
		return "", err
	}
	content := gjson.Get(args, "content").String()
	if content == "" {
		return "", errors.New("content is empty")
	}
	var query = `
	insert into memories (memory_uuid, memory_created_at, memory_content)
	values (?, ?, ?)
	`
	_, err := t.db.ExecContext(ctx, query,
		uuid.Must(uuid.NewV7()).String(),
		time.Now().UTC().Format(time.RFC3339Nano),
		content,
	)
	if err != nil {
		return "", err
	}
	return `{"ok":true}`, err
}

//---

type toolUpdateMemory struct {
	db *sql.DB
}

func (t *toolUpdateMemory) Name() string {
	return "update_memory"
}

func (t *toolUpdateMemory) Spec() []byte {
	var spec = `
{
	"name": "update_memory",
	"description": "Update a user's saved memory. Only to be used when the user explicitly asks to update a memory.",
	"parameters": {
		"type": "object",
		"properties": {
			"id": {
				"type": "string",
				"description": "The memory ID to update."
			},
			"content": {
				"type": "string",
				"description": "The memory content to save."
			}
		},
		"required": ["id", "content"],
		"additionalProperties": false
	},
	"strict": true
}
	`
	return []byte(strings.TrimSpace(spec))
}

func (t *toolUpdateMemory) Call(ctx context.Context, args string) (string, error) {
	if err := migrateMemoryTools(t.db); err != nil {
		return "", err
	}
	id := gjson.Get(args, "id").String()
	content := gjson.Get(args, "content").String()
	if id == "" || content == "" {
		return "", errors.New("id or content is empty")
	}
	var query = `
	update memories
	set memory_content = ?
	where memory_uuid = ?
	`
	_, err := t.db.ExecContext(ctx, query,
		content, id)
	if err != nil {
		return "", err
	}
	return `{"ok":true}`, err

}

//---

type toolDeleteMemory struct {
	db *sql.DB
}

func (t *toolDeleteMemory) Name() string {
	return "delete_memory"
}

func (t *toolDeleteMemory) Spec() []byte {
	var spec = `
{
	"name": "delete_memory",
	"description": "Delete a user's saved memory. Only to be used when the user explicitly asks to delete a memory.",
	"parameters": {
		"type": "object",
		"properties": {
			"id": {
				"type": "string",
				"description": "The memory ID to delete."
			}
		},
		"required": ["id"],
		"additionalProperties": false
	},
	"strict": true
}
	`
	return []byte(strings.TrimSpace(spec))
}

func (t *toolDeleteMemory) Call(ctx context.Context, args string) (string, error) {
	if err := migrateMemoryTools(t.db); err != nil {
		return "", err
	}
	id := gjson.Get(args, "id").String()
	if id == "" {
		return "", errors.New("id is empty")
	}
	var query = `
	delete from memories
	where memory_uuid = ?
	`
	_, err := t.db.ExecContext(ctx, query,
		id)
	if err != nil {
		return "", err
	}
	return `{"ok":true}`, err
}

//---

type toolListMemories struct {
	db *sql.DB
}

func (t *toolListMemories) Name() string {
	return "list_memories"
}

func (t *toolListMemories) Spec() []byte {
	var spec = `
{
	"name": "list_memories",
	"description": "List all user's saved memories. Only to be used when the user explicitly asks to use details from saved memories.",
	"parameters": {
		"type": "object",
		"properties": {},
		"required": [],
		"additionalProperties": false
	},
	"strict": true
}
	`
	return []byte(strings.TrimSpace(spec))
}

func (t *toolListMemories) Call(ctx context.Context, args string) (string, error) {
	if err := migrateMemoryTools(t.db); err != nil {
		return "", err
	}
	var query = `
	select memory_uuid, memory_created_at, memory_content
	from memories
	order by memory_created_at asc
	`
	rows, err := t.db.QueryContext(ctx, query)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	type memory struct {
		UUID      string `json:"id"`
		CreatedAt string `json:"created_at"`
		Content   string `json:"content"`
	}
	var memories []memory
	for rows.Next() {
		var m memory
		if err := rows.Scan(&m.UUID, &m.CreatedAt, &m.Content); err != nil {
			return "", err
		}
		memories = append(memories, m)
	}
	out, err := json.Marshal(memories)
	return string(out), err
}

//---

func migrateMemoryTools(db *sql.DB) error {
	var query = `
	create table if not exists memories (
		memory_id integer primary key,
		memory_uuid text not null,
		memory_created_at text not null,
		memory_content text not null
	)
	`
	_, err := db.Exec(query)
	return err
}
