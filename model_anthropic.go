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

var _ Model = (*anthropicModel)(nil)

type anthropicModelPersonality struct {
	id           string
	name         string
	systemPrompt string
}

type anthropicModel struct {
	id               string
	apiKey           string
	modelName        string
	displayName      string
	personalities    []anthropicModelPersonality
	tools            []Tool
	extendedThinking bool
}

type anthropicModelOption func(*anthropicModel)

func WithAnthropicModelDisplayName(name string) anthropicModelOption {
	return func(m *anthropicModel) {
		m.displayName = name
	}
}

func WithAnthropicModelPersonality(name string, systemPrompt string) anthropicModelOption {
	return func(m *anthropicModel) {
		id := xxhash.New()
		util.Must(id.WriteString("anthropic"))
		util.Must(id.WriteString("personality"))
		util.Must(id.WriteString(m.id))
		util.Must(id.WriteString(m.displayName))
		util.Must(id.WriteString(name))
		m.personalities = append(m.personalities, anthropicModelPersonality{
			id:           strconv.FormatUint(id.Sum64(), 10),
			name:         name,
			systemPrompt: systemPrompt,
		})
	}
}

func WithAnthropicModelTools(tools ...Tool) anthropicModelOption {
	return func(m *anthropicModel) {
		m.tools = append(m.tools, tools...)
	}
}

func WithAnthropicModelExtendedThinking() anthropicModelOption {
	return func(m *anthropicModel) {
		m.extendedThinking = true
	}
}

func NewAnthropicModel(apiKey string, modelName string, opts ...anthropicModelOption) *anthropicModel {
	id := xxhash.New()
	util.Must(id.WriteString("anthropic"))
	util.Must(id.WriteString(modelName))
	m := &anthropicModel{
		apiKey:           apiKey,
		modelName:        modelName,
		displayName:      modelName,
		personalities:    []anthropicModelPersonality{},
		tools:            []Tool{},
		extendedThinking: false,
	}
	for _, opt := range opts {
		opt(m)
	}
	util.Must(id.WriteString(m.displayName))
	m.id = "anthropic_" + strconv.FormatUint(id.Sum64(), 10)
	return m
}

func (m *anthropicModel) GetModelInfo() ModelInfo {
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

func (m *anthropicModel) StreamCompletion(ctx context.Context, systemPrompt string, history []ChatEvent, opts CompletionOpts) <-chan ChatEvent {
	temp := make([]ChatEvent, len(history))
	copy(temp, history)
	history = temp
	out := make(chan ChatEvent)
	go func() {
		defer close(out)
		type respContentBlockStart struct {
			Type         string `json:"type"`
			Index        int    `json:"index"`
			ContentBlock struct {
				Type string `json:"type"`
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"content_block"`
		}
		type respContentBlockDelta struct {
			Type  string `json:"type"`
			Index int    `json:"index"`
			Delta struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				Thinking    string `json:"thinking"`
				PartialJSON string `json:"partial_json"`
			} `json:"delta"`
		}
		type respBodyToolUse struct {
			Index    int
			ID       string
			FuncName string
			FuncArgs string
		}
	llm:
		chunks := callAnySSEAPI(ctx,
			func(ctx context.Context) (*http.Response, error) {
				type reqBodyThinking struct {
					Type         string `json:"type"`
					BudgetTokens int64  `json:"budget_tokens"`
				}
				type reqBodyMessageContent struct {
					Type string `json:"type"`
					// text fields
					Text string `json:"text,omitzero"`
					// tool result fields
					ToolUseID string `json:"tool_use_id,omitzero"`
					Content   string `json:"content,omitzero"`
					// tool call fields
					ID    string          `json:"id,omitzero"`
					Name  string          `json:"name,omitzero"`
					Input json.RawMessage `json:"input,omitzero"`
				}
				type reqBodyMessage struct {
					Role    string                  `json:"role"`
					Content []reqBodyMessageContent `json:"content"`
				}
				type reqBody struct {
					MaxTokens   int64             `json:"max_tokens"`
					Messages    []reqBodyMessage  `json:"messages"`
					Model       string            `json:"model"`
					Stream      bool              `json:"stream"`
					System      *string           `json:"system,omitempty"`
					Temperature float64           `json:"temperature"`
					Thinking    *reqBodyThinking  `json:"thinking,omitempty"`
					Tools       []json.RawMessage `json:"tools,omitempty"`
				}
				b := reqBody{
					MaxTokens:   128_000 / 2,
					Messages:    make([]reqBodyMessage, 0),
					Model:       m.modelName,
					Stream:      true,
					Temperature: 1.0,
				}
				if opts.UseTools {
					allTools := make([]Tool, 0, len(m.tools)+len(opts.ClientTools))
					allTools = append(allTools, m.tools...)
					allTools = append(allTools, opts.ClientTools...)
					for _, t := range allTools {
						spec, err := toAnthropicToolSpec(t.Spec())
						if err != nil {
							return nil, err
						}
						b.Tools = append(b.Tools, spec)
					}
				}
				if m.extendedThinking {
					if len(b.Tools) > 0 {
						return nil, fmt.Errorf("extended thinking is not supported with tools yet")
					}
					b.Thinking = &reqBodyThinking{
						Type:         "enabled",
						BudgetTokens: 8192,
					}
				}
				if len(systemPrompt) > 0 {
					loc, _ := time.LoadLocation("Europe/Helsinki")
					now := time.Now().In(loc).Format("Monday 2006-01-02 15:04:05")
					prompt := strings.ReplaceAll(systemPrompt, "{{current_time}}", now)
					b.System = &prompt
				}
				for _, i := range history {
					switch i := i.(type) {
					case *AssistantMessageChatEvent:
						content := []reqBodyMessageContent{}
						content = append(content, reqBodyMessageContent{
							Type: "text",
							Text: i.content,
						})
						for _, t := range i.toolCalls {
							content = append(content, reqBodyMessageContent{
								Type:  "tool_use",
								ID:    t.ID,
								Name:  t.FuncName,
								Input: json.RawMessage(t.FuncArgs),
							})
						}
						b.Messages = append(b.Messages, reqBodyMessage{
							Role:    "assistant",
							Content: content,
						})
					case *ToolMessageChatEvent:
						content := []reqBodyMessageContent{}
						content = append(content, reqBodyMessageContent{
							Type:      "tool_result",
							ToolUseID: i.callID,
							Content:   i.content,
						})
						b.Messages = append(b.Messages, reqBodyMessage{
							Role:    "user",
							Content: content,
						})
					case *UserMessageChatEvent:
						content := []reqBodyMessageContent{}
						content = append(content, reqBodyMessageContent{
							Type: "text",
							Text: i.content,
						})
						b.Messages = append(b.Messages, reqBodyMessage{
							Role:    "user",
							Content: content,
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
					http.MethodPost, "https://api.anthropic.com/v1/messages",
					&buf,
				)
				if err != nil {
					return nil, err
				}
				req.Header.Set("anthropic-version", "2023-06-01")
				req.Header.Set("content-type", "application/json")
				req.Header.Set("x-api-key", m.apiKey)
				return http.DefaultClient.Do(req)
			},
		)
		toolCallBuffer := make(map[int]respBodyToolUse)
		resp := NewAssistantMessageChatEvent("")
		out <- resp
		for r := range chunks {
			if r.Err != nil {
				fmt.Printf("error: %v\n", r.Err)
				return
			}
			if r.Val.T1 == "content_block_start" {
				var b respContentBlockStart
				if err := json.Unmarshal([]byte(r.Val.T2), &b); err != nil {
					fmt.Printf("error: %v\n", err)
					return
				}
				if b.ContentBlock.Type == "tool_use" {
					toolCallBuffer[len(toolCallBuffer)] = respBodyToolUse{
						Index:    b.Index,
						ID:       b.ContentBlock.ID,
						FuncName: b.ContentBlock.Name,
						FuncArgs: "",
					}
					resp.toolCalls = append(resp.toolCalls, assistantMessageToolCall{
						ID:       b.ContentBlock.ID,
						Type:     "function",
						FuncName: b.ContentBlock.Name,
						FuncArgs: "",
					})
					out <- resp
				}
				continue
			}
			if r.Val.T1 == "content_block_delta" {
				var b respContentBlockDelta
				if err := json.Unmarshal([]byte(r.Val.T2), &b); err != nil {
					fmt.Printf("error: %v\n", err)
					return
				}
				if b.Delta.Type == "thinking_delta" && b.Delta.Thinking != "" {
					resp.reasoning += b.Delta.Thinking
					out <- resp
				}
				if b.Delta.Type == "text_delta" && b.Delta.Text != "" {
					resp.content += b.Delta.Text
					out <- resp
				}
				if b.Delta.Type == "input_json_delta" && b.Delta.PartialJSON != "" {
					for i, v := range toolCallBuffer {
						if v.Index == b.Index {
							v.FuncArgs += b.Delta.PartialJSON
							toolCallBuffer[i] = v
							resp.toolCalls[i].FuncArgs = v.FuncArgs
							out <- resp
							break
						}
					}
				}
				continue
			}
		}
		if len(toolCallBuffer) > 0 {
			// send the tool calls
			resp.toolCalls = make([]assistantMessageToolCall, 0)
			for _, t := range toolCallBuffer {
				funcArgs := t.FuncArgs
				if funcArgs == "" {
					funcArgs = "{}"
				}
				resp.toolCalls = append(resp.toolCalls, assistantMessageToolCall{
					ID:       t.ID,
					Type:     "function",
					FuncName: t.FuncName,
					FuncArgs: funcArgs,
				})
			}
			history = append(history, resp)
			out <- resp
			// execute the tool calls
			for _, t := range toolCallBuffer {
				allTools := make([]Tool, 0, len(m.tools)+len(opts.ClientTools))
				allTools = append(allTools, m.tools...)
				allTools = append(allTools, opts.ClientTools...)
				for _, tt := range allTools {
					if tt.Name() == t.FuncName {
						v, err := tt.Call(ctx, t.FuncArgs)
						if err != nil {
							fmt.Printf("error: %v\n", err)
							vv, _ := json.Marshal(map[string]any{"ok": false, "error": err.Error()})
							v = string(vv)
						}
						resp := NewToolMessageChatEvent(t.ID, v)
						history = append(history, resp)
						out <- resp
						break
					}
				}
			}
			goto llm
		}
	}()
	return out
}
