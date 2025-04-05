package juttele

import (
	"bytes"
	"context"
	"encoding/json"
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
	id        string
	apiKey    string
	modelName string
}

func NewAnthropicModel(
	apiKey string, modelName string, opts ...modelOption,
) *anthropicModel {
	m := &anthropicModel{
		model:     &model{displayName: modelName},
		apiKey:    apiKey,
		modelName: modelName,
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
	if opts.Think {
		if opts.Tools == nil {
			opts.Tools = NewToolCatalog()
		} else {
			opts.Tools = opts.Tools.Copy()
		}
		m.injectThinkTool(opts.Tools)
	}
	copied := make([]Message, len(history))
	copy(copied, history)
	return streamWithTools(ctx, opts.Tools, &copied, func() <-chan Result[Message] {
		out := make(chan Result[Message], 1)
		defer close(out)
		resp, err := m.request(ctx, copied, opts)
		if err != nil {
			out <- Err[Message](err)
			return out
		}
		return streamAnthropic(resp)
	})
}

func (m *anthropicModel) isThinkingModel() bool {
	return strings.Contains(m.modelName, "claude-3-7-sonnet")
}

func (m *anthropicModel) injectThinkTool(tools *ToolCatalog) {
	var spec = `
{
	"name": "think",
	"description": "Use the tool to think about something. It will not obtain new information or make any changes to the repository, but just log the thought. Use it when complex reasoning or brainstorming is needed. For example, if you explore the repo and discover the source of a bug, call this tool to brainstorm several unique ways of fixing the bug, and assess which change(s) are likely to be simplest and most effective. Alternatively, if you receive some test results, call this tool to brainstorm ways to fix the failing tests.",
	"parameters": {
		"type": "object",
		"properties": {
			"thought": {
				"type": "string",
				"description": "Your thoughts."
			}
		},
		"required": ["thought"]
	}
}
	`
	tools.Register(newFuncTool(
		"think",
		[]byte(strings.TrimSpace(spec)),
		func(ctx context.Context, args string) (string, error) { return "", nil },
	))
}

func (m *anthropicModel) request(
	ctx context.Context, history []Message, opts GenerationConfig,
) (*http.Response, error) {
	type reqBody_message_thinking struct {
		Type      string `json:"type"`
		Thinking  string `json:"thinking"`
		Signature string `json:"signature"`
	}
	type reqBody_message_text struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type reqBody_message_toolUse struct {
		Type  string          `json:"type"`
		ID    string          `json:"id"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	}
	type reqBody_message_toolResult struct {
		Type      string `json:"type"`
		ToolUseID string `json:"tool_use_id"`
		Content   string `json:"content"`
	}
	type reqBody_message struct {
		Role    string `json:"role"`
		Content []any  `json:"content"`
	}
	type reqBody_thinking struct {
		Type         string `json:"type"`
		BudgetTokens int64  `json:"budget_tokens"`
	}
	type reqBody struct {
		MaxTokens   int64             `json:"max_tokens"`
		Messages    []reqBody_message `json:"messages"`
		Model       string            `json:"model"`
		Stream      bool              `json:"stream"`
		System      *string           `json:"system,omitempty"`
		Temperature float64           `json:"temperature"`
		Thinking    *reqBody_thinking `json:"thinking,omitempty"`
		Tools       []json.RawMessage `json:"tools,omitempty"`
	}
	b := reqBody{
		MaxTokens:   m.maxTokens,
		Messages:    []reqBody_message{},
		Model:       m.modelName,
		Stream:      true,
		Temperature: m.temperature,
	}
	if opts.MaxTokens > 0 {
		b.MaxTokens = opts.MaxTokens
	}
	if opts.Temperature != nil {
		b.Temperature = *opts.Temperature
	}
	if opts.Think && m.isThinkingModel() {
		b.Temperature = 1.0 // NOTE: Anthropic does not support temperature for extended thinking
		b.Thinking = &reqBody_thinking{
			Type:         "enabled",
			BudgetTokens: 8192, // TODO: make this configurable
		}
	}
	if opts.Tools != nil && opts.Tools.Count() > 0 {
		for _, t := range opts.Tools.List() {
			spec, err := m.spec(t.Spec())
			if err != nil {
				return nil, err
			}
			b.Tools = append(b.Tools, spec)
		}
	}
	thinkingSignatureSeen := false
	for _, i := range history {
		switch i := i.(type) {
		case *SystemMessage:
			loc, _ := time.LoadLocation("Europe/Helsinki")
			now := time.Now().In(loc).Format("Monday 2006-01-02 15:04:05")
			systemPrompt := strings.ReplaceAll(i.Content, "{{current_time}}", now)
			b.System = &systemPrompt
		case *AssistantMessage:
			content := []any{}
			if signature, _ := i.GetTransientMeta("signature"); i.Thinking != "" && signature != "" {
				if !thinkingSignatureSeen {
					thinkingSignatureSeen = true
					content = append(content, reqBody_message_thinking{
						Type:      "thinking",
						Thinking:  i.Thinking,
						Signature: signature,
					})
				}
			}
			if i.Content != "" {
				content = append(content, reqBody_message_text{
					Type: "text",
					Text: i.Content,
				})
			}
			for _, t := range i.ToolCalls {
				content = append(content, reqBody_message_toolUse{
					Type:  "tool_use",
					ID:    t.CallID,
					Name:  t.FuncName,
					Input: json.RawMessage(t.FuncArgs),
				})
			}
			b.Messages = append(b.Messages, reqBody_message{
				Role:    "assistant",
				Content: content,
			})
		case *ToolMessage:
			content := []any{}
			if i.Error != nil {
				content = append(content, reqBody_message_toolResult{
					Type:      "tool_result",
					ToolUseID: i.CallID,
					Content:   fmt.Sprintf("Error: %s", i.Error.Message),
				})
			} else {
				content = append(content, reqBody_message_toolResult{
					Type:      "tool_result",
					ToolUseID: i.CallID,
					Content:   *i.Result,
				})
			}
			b.Messages = append(b.Messages, reqBody_message{
				Role:    "user",
				Content: content,
			})
		case *UserMessage:
			if len(b.Messages) > 0 && b.Messages[len(b.Messages)-1].Role == "user" {
				idx := len(b.Messages) - 1
				b.Messages[idx].Content = append(b.Messages[idx].Content, reqBody_message_text{
					Type: "text",
					Text: i.Content,
				})
				continue
			}
			content := []any{reqBody_message_text{
				Type: "text",
				Text: i.Content,
			}}
			b.Messages = append(b.Messages, reqBody_message{
				Role:    "user",
				Content: content,
			})
		}
	}
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
