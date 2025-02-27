package juttele

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/markusylisiurunen/juttele/internal/util"
)

var _ Model = (*groqModel)(nil)

type groqModelPersonality struct {
	id           string
	name         string
	systemPrompt string
}

type groqModel struct {
	id            string
	apiKey        string
	modelName     string
	displayName   string
	personalities []groqModelPersonality
}

type groqModelOption func(*groqModel)

func WithGroqModelDisplayName(name string) groqModelOption {
	return func(m *groqModel) {
		m.displayName = name
	}
}

func WithGroqModelPersonality(name string, systemPrompt string) groqModelOption {
	return func(m *groqModel) {
		id := xxhash.New()
		util.Must(id.WriteString("groq"))
		util.Must(id.WriteString("personality"))
		util.Must(id.WriteString(m.id))
		util.Must(id.WriteString(m.displayName))
		util.Must(id.WriteString(name))
		m.personalities = append(m.personalities, groqModelPersonality{
			id:           strconv.FormatUint(id.Sum64(), 10),
			name:         name,
			systemPrompt: systemPrompt,
		})
	}
}

func NewGroqModel(apiKey string, modelName string, opts ...groqModelOption) *groqModel {
	id := xxhash.New()
	util.Must(id.WriteString("groq"))
	util.Must(id.WriteString(modelName))
	m := &groqModel{
		apiKey:        apiKey,
		modelName:     modelName,
		displayName:   modelName,
		personalities: []groqModelPersonality{},
	}
	for _, opt := range opts {
		opt(m)
	}
	util.Must(id.WriteString(m.displayName))
	m.id = "groq_" + strconv.FormatUint(id.Sum64(), 10)
	return m
}

func (m *groqModel) GetModelInfo() ModelInfo {
	personalities := make([]ModelPersonality, len(m.personalities))
	for i, p := range m.personalities {
		personalities[i] = ModelPersonality{
			ID:           p.id,
			Name:         p.name,
			SystemPrompt: p.systemPrompt,
		}
	}
	return ModelInfo{
		ID:            m.id,
		Name:          m.displayName,
		Personalities: personalities,
	}
}

func (m *groqModel) StreamCompletion(ctx context.Context, systemPrompt string, history []ChatEvent, _ CompletionOpts) <-chan ChatEvent {
	out := make(chan ChatEvent)
	go func() {
		defer close(out)
		chunks := callOpenAICompatibleAPI(ctx,
			func(ctx context.Context) (*http.Response, error) {
				type reqBodyMessage struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}
				type reqBody struct {
					Messages        []reqBodyMessage `json:"messages"`
					Model           string           `json:"model"`
					ReasoningFormat string           `json:"reasoning_format"`
					Stream          bool             `json:"stream"`
					Temperature     float64          `json:"temperature"`
				}
				b := reqBody{
					Messages:        make([]reqBodyMessage, 0),
					Model:           m.modelName,
					ReasoningFormat: "parsed",
					Stream:          true,
					Temperature:     0.6,
				}
				if systemPrompt != "" {
					loc, _ := time.LoadLocation("Europe/Helsinki")
					now := time.Now().In(loc).Format("Monday 2006-01-02 15:04:05")
					b.Messages = append(b.Messages, reqBodyMessage{
						Role:    "system",
						Content: strings.ReplaceAll(systemPrompt, "{{current_time}}", now),
					})
				}
				for _, i := range history {
					switch i := i.(type) {
					case *AssistantMessageChatEvent:
						b.Messages = append(b.Messages, reqBodyMessage{
							Role:    "assistant",
							Content: i.content,
						})
					case *UserMessageChatEvent:
						b.Messages = append(b.Messages, reqBodyMessage{
							Role:    "user",
							Content: i.content,
						})
					}
				}
				var buf bytes.Buffer
				enc := json.NewEncoder(&buf)
				enc.SetEscapeHTML(false)
				if err := enc.Encode(b); err != nil {
					return nil, err
				}
				req, err := http.NewRequestWithContext(ctx,
					http.MethodPost, "https://api.groq.com/openai/v1/chat/completions",
					&buf,
				)
				if err != nil {
					return nil, err
				}
				req.Header.Set("Authorization", "Bearer "+m.apiKey)
				req.Header.Set("Content-Type", "application/json")
				return http.DefaultClient.Do(req)
			},
		)
		resp := NewAssistantMessageChatEvent("")
		out <- resp
		for r := range chunks {
			if r.Err != nil {
				fmt.Printf("error: %v\n", r.Err)
				return
			}
			type respBody struct {
				Model   string `json:"model"`
				Choices []struct {
					Delta struct {
						Reasoning *string `json:"reasoning"`
						Content   *string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			var b respBody
			if err := json.Unmarshal([]byte(r.Val), &b); err != nil {
				fmt.Printf("error: %v\n", err)
				return
			}
			if len(b.Choices) == 0 {
				return
			}
			if b.Choices[0].Delta.Reasoning != nil {
				resp.reasoning += *b.Choices[0].Delta.Reasoning
				out <- resp
			}
			if b.Choices[0].Delta.Content != nil {
				resp.content += *b.Choices[0].Delta.Content
				out <- resp
			}
		}
	}()
	return out
}
