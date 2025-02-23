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

var _ Model = (*openRouterModel)(nil)

type openRouterModelPersonality struct {
	id           string
	name         string
	systemPrompt string
}

type openRouterModelTool struct {
	name    string
	spec    []byte
	handler func(context.Context, string) (string, error)
}

type openRouterModel struct {
	id            string
	apiKey        string
	modelName     string
	displayName   string
	personalities []openRouterModelPersonality
	tools         []openRouterModelTool
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

func WithOpenRouterModelTool(name string, spec []byte, handler func(context.Context, string) (string, error)) openRouterModelOption {
	return func(m *openRouterModel) {
		m.tools = append(m.tools, openRouterModelTool{name: name, spec: spec, handler: handler})
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
		tools:         []openRouterModelTool{},
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
	temp := make([]ChatEvent, len(history))
	copy(temp, history)
	history = temp
	out := make(chan ChatEvent)
	go func() {
		defer close(out)
		type respBodyToolCall struct {
			Index    int64  `json:"index"`
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		}
		type respBody struct {
			Model   string `json:"model"`
			Choices []struct {
				Delta struct {
					Reasoning *string            `json:"reasoning"`
					Content   *string            `json:"content"`
					ToolCalls []respBodyToolCall `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
		}
	llm:
		chunks := callOpenAICompatibleAPI(ctx,
			func(ctx context.Context) (*http.Response, error) {
				type reqBodyToolCall struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name string `json:"name"`
						Args string `json:"arguments"`
					} `json:"function"`
				}
				type reqBodyMessage struct {
					Role       string            `json:"role"`
					Content    string            `json:"content"`
					ToolCalls  []reqBodyToolCall `json:"tool_calls,omitempty"`
					ToolCallID *string           `json:"tool_call_id,omitempty"`
				}
				type reqBodyTool struct {
					Type     string          `json:"type"`
					Function json.RawMessage `json:"function"`
				}
				type reqBody struct {
					IncludeReasoning bool             `json:"include_reasoning"`
					Messages         []reqBodyMessage `json:"messages"`
					Model            string           `json:"model"`
					Stream           bool             `json:"stream"`
					Temperature      float64          `json:"temperature"`
					Tools            []reqBodyTool    `json:"tools,omitempty"`
				}
				b := reqBody{
					IncludeReasoning: true,
					Messages:         make([]reqBodyMessage, 0),
					Model:            m.modelName,
					Stream:           true,
					Temperature:      0.7,
				}
				if len(m.tools) > 0 {
					tools := make([]reqBodyTool, len(m.tools))
					for i, t := range m.tools {
						tools[i] = reqBodyTool{
							Type:     "function",
							Function: t.spec,
						}
					}
					b.Tools = tools
				}
				if len(systemPrompt) > 0 {
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
						message := reqBodyMessage{
							Role:    "assistant",
							Content: i.content,
						}
						for _, t := range i.toolCalls {
							message.ToolCalls = append(message.ToolCalls, reqBodyToolCall{
								ID:   t.ID,
								Type: t.Type,
								Function: struct {
									Name string `json:"name"`
									Args string `json:"arguments"`
								}{
									Name: t.FuncName,
									Args: t.FuncArgs,
								},
							})
						}
						b.Messages = append(b.Messages, message)
					case *ToolMessageChatEvent:
						toolCallID := i.callID
						b.Messages = append(b.Messages, reqBodyMessage{
							Role:       "tool",
							Content:    i.content,
							ToolCallID: &toolCallID,
						})
					case *UserMessageChatEvent:
						b.Messages = append(b.Messages, reqBodyMessage{
							Role:    "user",
							Content: i.content,
						})
					}
				}
				// fmt.Printf("%#v\n", b.Messages)
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
		toolCallBuffer := make(map[int64]respBodyToolCall)
		resp := NewAssistantMessageChatEvent("")
		out <- resp
		for r := range chunks {
			if r.Err != nil {
				fmt.Printf("error: %v\n", r.Err)
				return
			}
			var b respBody
			if err := json.Unmarshal([]byte(r.Val), &b); err != nil {
				fmt.Printf("error: %v\n", err)
				return
			}
			if len(b.Choices) == 0 {
				return
			}
			if len(b.Choices[0].Delta.ToolCalls) > 0 {
				for _, t := range b.Choices[0].Delta.ToolCalls {
					if _, ok := toolCallBuffer[t.Index]; !ok {
						c := respBodyToolCall{
							Index: t.Index,
							ID:    t.ID,
							Type:  t.Type,
						}
						c.Function.Name = t.Function.Name
						toolCallBuffer[t.Index] = c
					}
					c := toolCallBuffer[t.Index]
					c.Function.Arguments += t.Function.Arguments
					toolCallBuffer[t.Index] = c
				}
			}
			if b.Choices[0].Delta.Reasoning != nil {
				fmt.Printf("reasoning: %v\n", *b.Choices[0].Delta.Reasoning)
			}
			if b.Choices[0].Delta.Content != nil {
				resp.content += *b.Choices[0].Delta.Content
				out <- resp
			}
		}
		if len(toolCallBuffer) > 0 {
			// send the tool calls
			resp.toolCalls = make([]assistantMessageToolCall, 0)
			for _, t := range toolCallBuffer {
				resp.toolCalls = append(resp.toolCalls, assistantMessageToolCall{
					ID:       t.ID,
					Type:     t.Type,
					FuncName: t.Function.Name,
					FuncArgs: t.Function.Arguments,
				})
			}
			history = append(history, resp)
			out <- resp
			// execute the tool calls
			for _, t := range toolCallBuffer {
				for _, tt := range m.tools {
					if tt.name == t.Function.Name {
						v, err := tt.handler(ctx, t.Function.Arguments)
						if err != nil {
							fmt.Printf("error: %v\n", err)
							return
						}
						resp := NewToolMessageChatEvent(t.ID, v)
						history = append(history, resp)
						out <- resp
					}
				}
			}
			goto llm
		}
	}()
	return out
}
