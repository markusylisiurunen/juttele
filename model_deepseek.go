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

var _ Model = (*deepSeekModel)(nil)

type deepSeekModelPersonality struct {
	id           string
	name         string
	systemPrompt string
}

type deepSeekModel struct {
	id            string
	apiKey        string
	modelName     string
	displayName   string
	personalities []deepSeekModelPersonality
}

type deepSeekModelOption func(*deepSeekModel)

func WithDeepSeekModelDisplayName(name string) deepSeekModelOption {
	return func(m *deepSeekModel) {
		m.displayName = name
	}
}

func WithDeepSeekModelPersonality(name string, systemPrompt string) deepSeekModelOption {
	return func(m *deepSeekModel) {
		id := xxhash.New()
		util.Must(id.WriteString("deepseek"))
		util.Must(id.WriteString("personality"))
		util.Must(id.WriteString(m.id))
		util.Must(id.WriteString(m.displayName))
		util.Must(id.WriteString(name))
		m.personalities = append(m.personalities, deepSeekModelPersonality{
			id:           strconv.FormatUint(id.Sum64(), 10),
			name:         name,
			systemPrompt: systemPrompt,
		})
	}
}

func NewDeepSeekModel(apiKey string, modelName string, opts ...deepSeekModelOption) *deepSeekModel {
	id := xxhash.New()
	util.Must(id.WriteString("deepseek"))
	util.Must(id.WriteString(modelName))
	m := &deepSeekModel{
		apiKey:        apiKey,
		modelName:     modelName,
		displayName:   modelName,
		personalities: []deepSeekModelPersonality{},
	}
	for _, opt := range opts {
		opt(m)
	}
	util.Must(id.WriteString(m.displayName))
	m.id = "deepseek_" + strconv.FormatUint(id.Sum64(), 10)
	return m
}

func (m *deepSeekModel) GetModelInfo() ModelInfo {
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

func (m *deepSeekModel) StreamCompletion(ctx context.Context, systemPrompt string, history []ChatEvent, _ CompletionOpts) <-chan ChatEvent {
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
					MaxTokens int              `json:"max_tokens"`
					Messages  []reqBodyMessage `json:"messages"`
					Model     string           `json:"model"`
					Stream    bool             `json:"stream"`
				}
				b := reqBody{
					MaxTokens: 8000,
					Messages:  make([]reqBodyMessage, 0),
					Model:     m.modelName,
					Stream:    true,
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
					http.MethodPost, "https://api.deepseek.com/v1/chat/completions",
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
						Content          *string `json:"content"`
						ReasoningContent *string `json:"reasoning_content"`
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
			if b.Choices[0].Delta.ReasoningContent != nil {
				resp.reasoning += *b.Choices[0].Delta.ReasoningContent
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
