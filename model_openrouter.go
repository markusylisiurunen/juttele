package juttele

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cespare/xxhash/v2"
	"github.com/markusylisiurunen/juttele/internal/util"
)

var _ Model = (*openRouterModel)(nil)

type openRouterModelPersonality struct {
	id           string
	name         string
	systemPrompt string
}

type openRouterModel struct {
	id            string
	apiKey        string
	modelName     string
	displayName   string
	personalities []openRouterModelPersonality
}

type openRouterModelOption func(*openRouterModel)

func WithOpenRouterModelDisplayName(name string) openRouterModelOption {
	return func(m *openRouterModel) {
		m.displayName = name
	}
}

func WithOpenRouterModelPersonality(name string, systemPrompt string) openRouterModelOption {
	return func(m *openRouterModel) {
		id := xxhash.New()
		util.Must(id.WriteString("openrouter"))
		util.Must(id.WriteString("personality"))
		util.Must(id.WriteString(m.id))
		util.Must(id.WriteString(name))
		m.personalities = append(m.personalities, openRouterModelPersonality{
			id:           strconv.FormatUint(id.Sum64(), 10),
			name:         name,
			systemPrompt: systemPrompt,
		})
	}
}

func NewOpenRouterModel(apiKey string, modelName string, opts ...openRouterModelOption) *openRouterModel {
	id := xxhash.New()
	util.Must(id.WriteString("openrouter"))
	util.Must(id.WriteString(modelName))
	m := &openRouterModel{
		id:            "openrouter_" + strconv.FormatUint(id.Sum64(), 10),
		apiKey:        apiKey,
		modelName:     modelName,
		displayName:   modelName,
		personalities: []openRouterModelPersonality{},
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *openRouterModel) GetModelInfo() ModelInfo {
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

func (m *openRouterModel) StreamCompletion(ctx context.Context, systemPrompt string, history []ChatEvent) <-chan ChatEvent {
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
					IncludeReasoning bool             `json:"include_reasoning"`
					Messages         []reqBodyMessage `json:"messages"`
					Model            string           `json:"model"`
					Stream           bool             `json:"stream"`
					Temperature      float64          `json:"temperature"`
				}
				b := reqBody{
					IncludeReasoning: true,
					Messages:         make([]reqBodyMessage, 0),
					Model:            m.modelName,
					Stream:           true,
					Temperature:      0.7,
				}
				if len(systemPrompt) > 0 {
					b.Messages = append(b.Messages, reqBodyMessage{
						Role:    "system",
						Content: systemPrompt,
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
				body, err := json.Marshal(b)
				if err != nil {
					return nil, err
				}
				req, err := http.NewRequestWithContext(ctx,
					http.MethodPost, "https://openrouter.ai/api/v1/chat/completions",
					bytes.NewReader(body),
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
				fmt.Printf("reasoning: %v\n", *b.Choices[0].Delta.Reasoning)
			}
			if b.Choices[0].Delta.Content != nil {
				resp.content += *b.Choices[0].Delta.Content
				out <- resp
			}
		}
	}()
	return out
}
