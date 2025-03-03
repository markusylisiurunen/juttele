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

type openRouterModel struct {
	*model
	id        string
	apiKey    string
	modelName string
}

func NewOpenRouterModel(apiKey string, modelName string, opts ...modelOption) *openRouterModel {
	m := &openRouterModel{
		model:     &model{displayName: modelName},
		apiKey:    apiKey,
		modelName: modelName,
	}
	for _, opt := range opts {
		opt(m.model)
	}
	id := xxhash.New()
	util.Must(id.WriteString("openrouter"))
	util.Must(id.WriteString(modelName))
	util.Must(id.WriteString(m.displayName))
	m.id = "openrouter_" + strconv.FormatUint(id.Sum64(), 10)
	return m
}

func (m *openRouterModel) GetModelInfo() ModelInfo {
	return m.getModelInfo(m.id)
}

func (m *openRouterModel) StreamCompletion(
	ctx context.Context, history []ChatEvent, opts StreamCompletionOpts,
) <-chan Result[ChatEvent] {
	copied := make([]ChatEvent, len(history))
	copy(copied, history)
	return streamWithTools(ctx, opts.Tools, &copied, func() <-chan Result[ChatEvent] {
		resp, err := m.request(ctx, copied, opts)
		if err != nil {
			out := make(chan Result[ChatEvent], 1)
			defer close(out)
			out <- Err[ChatEvent](err)
			return out
		}
		return streamOpenAI(resp)
	})
}

func (m *openRouterModel) request(
	ctx context.Context, history []ChatEvent, opts StreamCompletionOpts,
) (*http.Response, error) {
	// basic text message
	type reqTextMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	// assistant message with possible tool calls
	type reqToolCallFunction struct {
		Name string `json:"name"`
		Args string `json:"arguments"`
	}
	type reqToolCall struct {
		ID       string              `json:"id"`
		Type     string              `json:"type"`
		Function reqToolCallFunction `json:"function"`
	}
	type reqAssistantMessage struct {
		Role      string        `json:"role"`
		Content   string        `json:"content"`
		ToolCalls []reqToolCall `json:"tool_calls,omitempty"`
	}
	// tool message
	type reqToolMessage struct {
		Role       string `json:"role"`
		Content    string `json:"content"`
		ToolCallID string `json:"tool_call_id"`
	}
	// the request body
	type reqTool struct {
		Type     string          `json:"type"`
		Function json.RawMessage `json:"function"`
	}
	type reqBody struct {
		IncludeReasoning bool      `json:"include_reasoning"`
		MaxTokens        int64     `json:"max_tokens,omitempty"`
		Messages         []any     `json:"messages"`
		Model            string    `json:"model"`
		Stream           bool      `json:"stream"`
		Temperature      float64   `json:"temperature"`
		Tools            []reqTool `json:"tools,omitempty"`
	}
	b := reqBody{
		IncludeReasoning: true,
		MaxTokens:        m.maxTokens,
		Messages:         []any{},
		Model:            m.modelName,
		Stream:           true,
		Temperature:      m.temperature,
	}
	// populate the tools
	if opts.UseTools {
		for _, t := range opts.Tools.List() {
			b.Tools = append(b.Tools, reqTool{
				Type:     "function",
				Function: t.Spec(),
			})
		}
	}
	// append the system prompt
	if len(opts.SystemPrompt) > 0 {
		loc, _ := time.LoadLocation("Europe/Helsinki")
		now := time.Now().In(loc).Format("Monday 2006-01-02 15:04:05")
		b.Messages = append(b.Messages, reqTextMessage{
			Role:    "system",
			Content: strings.ReplaceAll(opts.SystemPrompt, "{{current_time}}", now),
		})
	}
	// append the message history
	for _, i := range history {
		switch i := i.(type) {
		case *AssistantMessageChatEvent:
			msg := reqAssistantMessage{
				Role:    "assistant",
				Content: i.content,
			}
			for _, t := range i.toolCalls {
				msg.ToolCalls = append(msg.ToolCalls, reqToolCall{
					ID:   t.ID,
					Type: "function",
					Function: reqToolCallFunction{
						Name: t.FuncName,
						Args: t.FuncArgs,
					},
				})
			}
			b.Messages = append(b.Messages, msg)
		case *ToolMessageChatEvent:
			b.Messages = append(b.Messages, reqToolMessage{
				Role:       "tool",
				Content:    i.content,
				ToolCallID: i.callID,
			})
		case *UserMessageChatEvent:
			b.Messages = append(b.Messages, reqTextMessage{
				Role:    "user",
				Content: i.content,
			})
		}
	}
	// make the request
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(b); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost, "https://openrouter.ai/api/v1/chat/completions", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return resp, nil
}
