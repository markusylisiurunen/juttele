package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/markusylisiurunen/juttele"
	"github.com/tidwall/gjson"
)

//go:embed prompts/neutral.txt
var neutralSystemPrompt string

type Note struct {
	UUID      string `json:"uuid"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Title     string `json:"title"`
	Content   string `json:"content"`
}

var notesStore = make([]Note, 0)
var notesMutex sync.Mutex

var listNotesToolSpec = []byte(strings.TrimSpace(`
{
  "name": "list_notes",
  "description": "List all user's notes.",
  "parameters": {
    "type": "object",
    "properties": {},
    "required": [],
    "additionalProperties": false
  },
  "strict": true
}
`))

var writeNoteToolSpec = []byte(strings.TrimSpace(`
{
	"name": "write_note",
	"description": "Create a new note.",
	"parameters": {
		"type": "object",
		"properties": {
			"title": {
				"type": "string",
				"description": "The title of the note."
			},
			"content": {
				"type": "string",
				"description": "The content of the note."
			}
		},
		"required": ["title", "content"],
		"additionalProperties": false
	},
	"strict": true
}
`))

var updateNoteToolSpec = []byte(strings.TrimSpace(`
{
	"name": "update_note",
	"description": "Update an existing note.",
	"parameters": {
		"type": "object",
		"properties": {
			"uuid": {
				"type": "string",
				"description": "The UUID of the note to update."
			},
			"title": {
				"type": "string",
				"description": "The new title of the note."
			},
			"content": {
				"type": "string",
				"description": "The new content of the note."
			}
		},
		"required": ["uuid", "title", "content"],
		"additionalProperties": false
	},
	"strict": true
}
`))

func listNotesToolHandler(ctx context.Context, args string) (string, error) {
	fmt.Printf("list_notes(%s)\n", args)
	notesMutex.Lock()
	defer notesMutex.Unlock()
	resp, err := json.Marshal(notesStore)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

func writeOrUpdateNoteToolHandler(ctx context.Context, args string) (string, error) {
	notesMutex.Lock()
	defer notesMutex.Unlock()
	var (
		argUuid    = gjson.Get(args, "uuid").String()
		argTitle   = gjson.Get(args, "title").String()
		argContent = gjson.Get(args, "content").String()
	)
	if argUuid == "" {
		fmt.Printf("write_note(%s)\n", args)
		notesStore = append(notesStore, Note{
			UUID:      uuid.Must(uuid.NewV7()).String(),
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			Title:     argTitle,
			Content:   argContent,
		})
		return `{"ok":true}`, nil
	}
	fmt.Printf("update_note(%s)\n", args)
	idx := slices.IndexFunc(notesStore, func(i Note) bool { return i.UUID == argUuid })
	if idx == -1 {
		return `{"ok":false}`, nil
	}
	notesStore[idx].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	notesStore[idx].Title = argTitle
	notesStore[idx].Content = argContent
	return `{"ok":true}`, nil
}

func main() {
	var (
		googleToken     = os.Getenv("GOOGLE_TOKEN")
		groqToken       = os.Getenv("GROQ_TOKEN")
		openRouterToken = os.Getenv("OPEN_ROUTER_TOKEN")
	)
	var (
		gpt4o = juttele.NewOpenRouterModel(openRouterToken, "openai/gpt-4o-2024-11-20",
			juttele.WithOpenRouterModelDisplayName("GPT-4o"),
			juttele.WithOpenRouterModelPersonality("Neutral", neutralSystemPrompt),
			juttele.WithOpenRouterModelTool("list_notes", listNotesToolSpec, listNotesToolHandler),
			juttele.WithOpenRouterModelTool("write_note", writeNoteToolSpec, writeOrUpdateNoteToolHandler),
			juttele.WithOpenRouterModelTool("update_note", updateNoteToolSpec, writeOrUpdateNoteToolHandler),
		)
		claude35Sonnet = juttele.NewOpenRouterModel(openRouterToken, "anthropic/claude-3.5-sonnet:beta",
			juttele.WithOpenRouterModelDisplayName("Claude 3.5 Sonnet"),
			juttele.WithOpenRouterModelPersonality("Neutral", neutralSystemPrompt),
			juttele.WithOpenRouterModelTool("list_notes", listNotesToolSpec, listNotesToolHandler),
			juttele.WithOpenRouterModelTool("write_note", writeNoteToolSpec, writeOrUpdateNoteToolHandler),
			juttele.WithOpenRouterModelTool("update_note", updateNoteToolSpec, writeOrUpdateNoteToolHandler),
		)
		deepseekR1Llama70b = juttele.NewGroqModel(groqToken, "deepseek-r1-distill-llama-70b",
			juttele.WithGroqModelDisplayName("DeepSeek R1 (Llama 70B)"),
			juttele.WithGroqModelPersonality("Neutral", neutralSystemPrompt),
		)
		gemini20FlashThinking = juttele.NewGoogleModel(googleToken, "gemini-2.0-flash-thinking-exp-01-21",
			juttele.WithGoogleModelDisplayName("Gemini 2.0 Flash Thinking"),
			juttele.WithGoogleModelPersonality("Neutral", neutralSystemPrompt),
		)
	)
	app := juttele.New("YOUR_TOKEN_HERE",
		juttele.WithModel(gpt4o),
		juttele.WithModel(claude35Sonnet),
		juttele.WithModel(deepseekR1Llama70b),
		juttele.WithModel(gemini20FlashThinking),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := app.ListenAndServe(ctx); err != nil {
			fmt.Printf("error: %v\n", err)
		}
	}()
	<-c
	cancel()
	<-done
}
