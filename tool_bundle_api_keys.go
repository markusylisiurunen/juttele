package juttele

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/markusylisiurunen/juttele/internal/repo"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tidwall/gjson"
)

type apiKeyToolBundle struct {
	dataFolder string
	client     *sql.DB
	clientMu   sync.Mutex
}

func NewAPIKeyToolBundle(dataFolder string) ToolBundle {
	return &apiKeyToolBundle{dataFolder: dataFolder}
}

func (m *apiKeyToolBundle) Tools() []Tool {
	return []Tool{
		m.createAPIKey(),
	}
}

func (m *apiKeyToolBundle) createAPIKey() Tool {
	var spec = `
{
	"name": "create_api_key",
	"description": "Generates a new API key for authenticating requests to the Juttele API.",
	"parameters": {
		"type": "object",
		"properties": {
			"expiration_minutes": {
				"type": "integer",
				"description": "Optional duration in minutes until the API key expires. If omitted, a system default is used."
			}
		},
		"required": [],
		"additionalProperties": false
	}
}
	`
	return newFuncTool(
		"create_api_key",
		[]byte(strings.TrimSpace(spec)),
		func(ctx context.Context, args string) (string, error) {
			minutes := gjson.Get(args, "expiration_minutes").Int()
			if minutes == 0 {
				minutes = 60
			}
			client, err := m.getClient()
			if err != nil {
				return "", err
			}
			apiKeyUUID := uuid.New().String()
			err = repo.New(client).CreateAPIKey(ctx, repo.CreateAPIKeyArgs{
				ExpiresIn: time.Duration(minutes) * time.Minute,
				UUID:      apiKeyUUID,
			})
			if err != nil {
				return "", err
			}
			out, err := json.Marshal(map[string]any{"api_key": apiKeyUUID})
			return string(out), err
		},
	)
}

func (m *apiKeyToolBundle) getClient() (*sql.DB, error) {
	m.clientMu.Lock()
	defer m.clientMu.Unlock()
	if m.client == nil {
		client, err := sql.Open("sqlite3",
			fmt.Sprintf("file:%s/juttele.db?_fk=1", m.dataFolder))
		if err != nil {
			return nil, err
		}
		m.client = client
	}
	return m.client, nil
}
