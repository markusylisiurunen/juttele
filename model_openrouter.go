package juttele

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/cespare/xxhash/v2"
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
		must(id.WriteString("openrouter"))
		must(id.WriteString("personality"))
		must(id.WriteString(m.id))
		must(id.WriteString(name))
		m.personalities = append(m.personalities, openRouterModelPersonality{
			id:           strconv.FormatUint(id.Sum64(), 10),
			name:         name,
			systemPrompt: systemPrompt,
		})
	}
}

func NewOpenRouterModel(apiKey string, modelName string, opts ...openRouterModelOption) *openRouterModel {
	id := xxhash.New()
	must(id.WriteString("openrouter"))
	must(id.WriteString(modelName))
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

func (m *openRouterModel) StreamCompletion(ctx context.Context, history []Message) <-chan Chunk {
	return streamOpenAICompatibleCompletion(ctx,
		func(ctx context.Context) (*http.Response, error) {
			type reqBodyMessage struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}
			type reqBody struct {
				Model    string           `json:"model"`
				Messages []reqBodyMessage `json:"messages"`
				Stream   bool             `json:"stream"`
				// MaxTokens        int              `json:"max_tokens"`
				Temperature      float64 `json:"temperature"`
				IncludeReasoning bool    `json:"include_reasoning"`
			}
			b := reqBody{
				Model:    m.modelName,
				Messages: make([]reqBodyMessage, len(history)),
				Stream:   true,
				// MaxTokens:        4096,
				Temperature:      0.7,
				IncludeReasoning: true,
			}
			for i, msg := range history {
				b.Messages[i] = reqBodyMessage{
					Role:    string(msg.GetRole()),
					Content: msg.GetContent(),
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
		func(chunk []byte) (Chunk, error) {
			type respBody struct {
				Model   string `json:"model"`
				Choices []struct {
					Delta struct {
						Content   *string `json:"content"`
						Reasoning *string `json:"reasoning"`
					} `json:"delta"`
				} `json:"choices"`
			}
			var b respBody
			if err := json.Unmarshal([]byte(chunk), &b); err != nil {
				return nil, err
			}
			if len(b.Choices) == 0 {
				return nil, nil
			}
			if b.Choices[0].Delta.Reasoning != nil {
				return ThinkingChunk(*b.Choices[0].Delta.Reasoning), nil
			}
			if b.Choices[0].Delta.Content != nil {
				return ContentChunk(*b.Choices[0].Delta.Content), nil
			}
			return nil, nil
		},
	)
}
