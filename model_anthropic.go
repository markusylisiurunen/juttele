package juttele

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/markusylisiurunen/juttele/internal/util"
)

var _ Model = (*anthropicModel)(nil)

type anthropicModel struct {
	*model
	id               string
	apiKey           string
	modelName        string
	extendedThinking bool
}

func NewAnthropicModel(apiKey string, modelName string, think bool, opts ...modelOption) *anthropicModel {
	m := &anthropicModel{
		model:            &model{displayName: modelName},
		apiKey:           apiKey,
		modelName:        modelName,
		extendedThinking: think,
	}
	for _, opt := range opts {
		opt(m.model)
	}
	id := xxhash.New()
	util.Must(id.WriteString("anthropic"))
	util.Must(id.WriteString(modelName))
	util.Must(id.WriteString(m.displayName))
	m.id = "anthropic_" + strconv.FormatUint(id.Sum64(), 10)
	return m
}

func (m *anthropicModel) GetModelInfo() ModelInfo {
	return m.getModelInfo(m.id)
}

func (m *anthropicModel) StreamCompletion(
	ctx context.Context, history []Message, opts GenerationConfig,
) <-chan Result[Message] {
	if opts.Tools.Count() > 0 && m.extendedThinking {
		out := make(chan Result[Message], 1)
		defer close(out)
		out <- Err[Message](errors.New("tools cannot be used with extended thinking for now"))
		return out
	}
	copied := make([]Message, len(history))
	copy(copied, history)
	return streamWithTools(ctx, opts.Tools, &copied, func() <-chan Result[Message] {
		resp, err := m.request(ctx, copied, opts)
		if err != nil {
			out := make(chan Result[Message], 1)
			defer close(out)
			out <- Err[Message](err)
			return out
		}
		return streamAnthropic(resp)
	})
}

func (m *anthropicModel) request(
	ctx context.Context, history []Message, opts GenerationConfig,
) (*http.Response, error) {
	// basic text message
	type reqTextMessage struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	// tool use message
	type reqToolUseMessage struct {
		Type  string          `json:"type"`
		ID    string          `json:"id"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	}
	// tool result message
	type reqToolResultMessage struct {
		Type      string `json:"type"`
		ToolUseID string `json:"tool_use_id"`
		Content   string `json:"content"`
	}
	// the request body
	type reqMessage struct {
		Role    string `json:"role"`
		Content []any  `json:"content"`
	}
	type reqThinkConfig struct {
		Type         string `json:"type"`
		BudgetTokens int64  `json:"budget_tokens"`
	}
	type reqBody struct {
		MaxTokens   int64             `json:"max_tokens"`
		Messages    []reqMessage      `json:"messages"`
		Model       string            `json:"model"`
		Stream      bool              `json:"stream"`
		System      *string           `json:"system,omitempty"`
		Temperature float64           `json:"temperature"`
		Thinking    *reqThinkConfig   `json:"thinking,omitempty"`
		Tools       []json.RawMessage `json:"tools,omitempty"`
	}
	b := reqBody{
		MaxTokens:   m.maxTokens,
		Messages:    []reqMessage{},
		Model:       m.modelName,
		Stream:      true,
		Temperature: m.temperature,
	}
	if b.MaxTokens == 0 {
		b.MaxTokens = 16384
	}
	// populate the thinking config
	if m.extendedThinking {
		b.Thinking = &reqThinkConfig{
			Type:         "enabled",
			BudgetTokens: 8192,
		}
	}
	// populate the tools
	if opts.Tools.Count() > 0 {
		for _, t := range opts.Tools.List() {
			spec, err := m.spec(t.Spec())
			if err != nil {
				return nil, err
			}
			b.Tools = append(b.Tools, spec)
		}
	}
	// append the message history
	for _, i := range history {
		switch i := i.(type) {
		case *SystemMessage:
			loc, _ := time.LoadLocation("Europe/Helsinki")
			now := time.Now().In(loc).Format("Monday 2006-01-02 15:04:05")
			systemPrompt := strings.ReplaceAll(i.Content, "{{current_time}}", now)
			b.System = &systemPrompt
		case *AssistantMessage:
			content := []any{}
			content = append(content, reqTextMessage{
				Type: "text",
				Text: i.Content,
			})
			for _, t := range i.ToolCalls {
				content = append(content, reqToolUseMessage{
					Type:  "tool_use",
					ID:    t.CallID,
					Name:  t.FuncName,
					Input: json.RawMessage(t.FuncArgs),
				})
			}
			b.Messages = append(b.Messages, reqMessage{
				Role:    "assistant",
				Content: content,
			})
		case *ToolMessage:
			content := []any{}
			if i.Error != nil {
				result := map[string]any{
					"ok": false,
					"error": map[string]any{
						"code":    i.Error.Code,
						"message": i.Error.Message,
					},
				}
				content = append(content, reqToolResultMessage{
					Type:      "tool_result",
					ToolUseID: i.CallID,
					Content:   string(util.Must(json.Marshal(result))),
				})
			} else {
				content = append(content, reqToolResultMessage{
					Type:      "tool_result",
					ToolUseID: i.CallID,
					Content:   *i.Result,
				})
			}
			b.Messages = append(b.Messages, reqMessage{
				Role:    "user",
				Content: content,
			})
		case *UserMessage:
			content := []any{}
			content = append(content, reqTextMessage{
				Type: "text",
				Text: i.Content,
			})
			b.Messages = append(b.Messages, reqMessage{
				Role:    "user",
				Content: content,
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
		http.MethodPost, "https://api.anthropic.com/v1/messages", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", m.apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if err := resp.Body.Close(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}
	return resp, nil
}

func (m *anthropicModel) spec(spec []byte) ([]byte, error) {
	type OpenAIToolSpec struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Parameters  struct {
			Type       string `json:"type"`
			Properties map[string]struct {
				Type        string `json:"type"`
				Description string `json:"description"`
			} `json:"properties"`
			Required             []string `json:"required"`
			AdditionalProperties bool     `json:"additionalProperties"`
		} `json:"parameters"`
	}
	type AnthropicToolSpec struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		InputSchema struct {
			Type       string `json:"type"`
			Properties map[string]struct {
				Type        string `json:"type"`
				Description string `json:"description"`
			} `json:"properties"`
			Required []string `json:"required"`
		} `json:"input_schema"`
	}
	var openAITool OpenAIToolSpec
	if err := json.Unmarshal(spec, &openAITool); err != nil {
		return nil, err
	}
	var anthropicTool AnthropicToolSpec
	anthropicTool.Name = openAITool.Name
	anthropicTool.Description = openAITool.Description
	anthropicTool.InputSchema.Type = openAITool.Parameters.Type
	anthropicTool.InputSchema.Properties = make(map[string]struct {
		Type        string `json:"type"`
		Description string `json:"description"`
	})
	for k, v := range openAITool.Parameters.Properties {
		anthropicTool.InputSchema.Properties[k] = struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		}{
			Type:        v.Type,
			Description: v.Description,
		}
	}
	anthropicTool.InputSchema.Required = openAITool.Parameters.Required
	return json.Marshal(anthropicTool)
}
