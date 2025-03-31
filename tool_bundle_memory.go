package juttele

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"

	_ "github.com/mattn/go-sqlite3"
)

type memoryToolBundle struct {
	dataFolder string
	client     *sql.DB
	clientMu   sync.Mutex
}

func NewMemoryToolBundle(dataFolder string) ToolBundle {
	return &memoryToolBundle{dataFolder: dataFolder}
}

func (m *memoryToolBundle) Tools() []Tool {
	return []Tool{
		m.listMemoriesTool(),
		m.saveMemoryTool(),
		m.updateMemoryTool(),
		m.deleteMemoryTool(),
	}
}

func (m *memoryToolBundle) listMemoriesTool() Tool {
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
	return newFuncTool(
		"list_memories",
		[]byte(strings.TrimSpace(spec)),
		func(ctx context.Context, args string) (string, error) {
			client, err := m.getClient()
			if err != nil {
				return "", err
			}
			var query = `
			select memory_uuid, memory_created_at, memory_content
			from memories
			order by memory_created_at asc
			`
			rows, err := client.QueryContext(ctx, query)
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
		},
	)
}

func (m *memoryToolBundle) saveMemoryTool() Tool {
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
	return newFuncTool(
		"save_memory",
		[]byte(strings.TrimSpace(spec)),
		func(ctx context.Context, args string) (string, error) {
			client, err := m.getClient()
			if err != nil {
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
			_, err = client.ExecContext(ctx, query,
				uuid.Must(uuid.NewV7()).String(),
				time.Now().UTC().Format(time.RFC3339Nano),
				content,
			)
			if err != nil {
				return "", err
			}
			out, err := json.Marshal(map[string]any{"ok": true})
			return string(out), err
		},
	)
}

func (m *memoryToolBundle) updateMemoryTool() Tool {
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
	return newFuncTool(
		"update_memory",
		[]byte(strings.TrimSpace(spec)),
		func(ctx context.Context, args string) (string, error) {
			client, err := m.getClient()
			if err != nil {
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
			_, err = client.ExecContext(ctx, query,
				content, id)
			if err != nil {
				return "", err
			}
			out, err := json.Marshal(map[string]any{"ok": true})
			return string(out), err
		},
	)
}

func (m *memoryToolBundle) deleteMemoryTool() Tool {
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
	return newFuncTool(
		"delete_memory",
		[]byte(strings.TrimSpace(spec)),
		func(ctx context.Context, args string) (string, error) {
			client, err := m.getClient()
			if err != nil {
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
			_, err = client.ExecContext(ctx, query,
				id)
			if err != nil {
				return "", err
			}
			out, err := json.Marshal(map[string]any{"ok": true})
			return string(out), err
		},
	)
}

func (m *memoryToolBundle) getClient() (*sql.DB, error) {
	m.clientMu.Lock()
	defer m.clientMu.Unlock()
	if m.client == nil {
		client, err := sql.Open("sqlite3",
			fmt.Sprintf("file:%s/memories.db?_fk=1", m.dataFolder))
		if err != nil {
			return nil, err
		}
		if err := m.migrateClient(client); err != nil {
			return nil, err
		}
		m.client = client
	}
	return m.client, nil
}

func (m *memoryToolBundle) migrateClient(client *sql.DB) error {
	var query = `
	create table if not exists memories (
		memory_id integer primary key,
		memory_uuid text not null,
		memory_created_at text not null,
		memory_content text not null
	)
	`
	_, err := client.Exec(query)
	return err
}
