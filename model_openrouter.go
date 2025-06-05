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

var _ Model = (*openRouterModel)(nil)

type openRouterModel struct {
	*model
	id        string
	apiKey    string
	modelName string
	providers []string
}

func NewOpenRouterModel(
	apiKey string, modelName string, providers []string, opts ...modelOption,
) *openRouterModel {
	m := &openRouterModel{
		model:     &model{displayName: modelName},
		apiKey:    apiKey,
		modelName: modelName,
		providers: providers,
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
		return streamOpenAI(resp)
	})
}

func (m *openRouterModel) injectThinkTool(tools *ToolCatalog) {
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

func (m *openRouterModel) request(
	ctx context.Context, history []Message, opts GenerationConfig,
) (*http.Response, error) {
	type reqBody_toolCall_function struct {
		Name string `json:"name"`
		Args string `json:"arguments"`
	}
	type reqBody_toolCall struct {
		ID       string                    `json:"id"`
		Type     string                    `json:"type"`
		Function reqBody_toolCall_function `json:"function"`
	}
	type reqBody_message struct {
		Role       string             `json:"role"`
		Content    string             `json:"content"`
		ToolCalls  []reqBody_toolCall `json:"tool_calls,omitempty"`
		ToolCallID string             `json:"tool_call_id,omitempty"`
	}
	type reqBody_tool struct {
		Type     string          `json:"type"`
		Function json.RawMessage `json:"function"`
	}
	type reqBody_provider struct {
		AllowFallbacks bool     `json:"allow_fallbacks"`
		Order          []string `json:"order"`
	}
	type reqBody_Reasoning struct {
		Effort    string `json:"effort,omitzero"`
		MaxTokens int64  `json:"max_tokens,omitzero"`
	}
	type reqBody_responseFormat struct {
		Type string `json:"type"`
	}
	type reqBody struct {
		MaxTokens      int64                   `json:"max_tokens,omitempty"`
		Messages       []reqBody_message       `json:"messages"`
		Model          string                  `json:"model"`
		Provider       *reqBody_provider       `json:"provider,omitempty"`
		Reasoning      *reqBody_Reasoning      `json:"reasoning,omitempty"`
		ResponseFormat *reqBody_responseFormat `json:"response_format,omitempty"`
		Stream         bool                    `json:"stream"`
		Temperature    float64                 `json:"temperature"`
		Tools          []reqBody_tool          `json:"tools,omitempty"`
	}
	b := reqBody{
		MaxTokens:   m.maxTokens,
		Messages:    []reqBody_message{},
		Model:       m.modelName,
		Reasoning:   &reqBody_Reasoning{MaxTokens: 1024},
		Stream:      true,
		Temperature: m.temperature,
	}
	if len(m.providers) > 0 {
		b.Provider = &reqBody_provider{
			AllowFallbacks: false,
			Order:          m.providers,
		}
	}
	if opts.MaxTokens > 0 {
		b.MaxTokens = opts.MaxTokens
	}
	if opts.Temperature != nil {
		b.Temperature = *opts.Temperature
	}
	if opts.Think {
		b.Reasoning = &reqBody_Reasoning{Effort: "high"}
	}
	if opts.JSON {
		b.ResponseFormat = &reqBody_responseFormat{
			Type: "json_object",
		}
	}
	if opts.Tools != nil && opts.Tools.Count() > 0 {
		for _, t := range opts.Tools.List() {
			spec, err := m.spec(t.Spec())
			if err != nil {
				return nil, err
			}
			b.Tools = append(b.Tools, reqBody_tool{
				Type:     "function",
				Function: spec,
			})
		}
	}
	for _, i := range history {
		switch i := i.(type) {
		case *SystemMessage:
			loc, _ := time.LoadLocation("Europe/Helsinki")
			now := time.Now().In(loc).Format("Monday 2006-01-02 15:04:05")
			systemPrompt := strings.ReplaceAll(i.Content, "{{current_time}}", now)
			b.Messages = append(b.Messages, reqBody_message{
				Role:    "system",
				Content: systemPrompt,
			})
		case *AssistantMessage:
			msg := reqBody_message{
				Role:    "assistant",
				Content: i.Content,
			}
			for _, t := range i.ToolCalls {
				msg.ToolCalls = append(msg.ToolCalls, reqBody_toolCall{
					ID:   t.CallID,
					Type: "function",
					Function: reqBody_toolCall_function{
						Name: t.FuncName,
						Args: t.FuncArgs,
					},
				})
			}
			b.Messages = append(b.Messages, msg)
		case *ToolMessage:
			msg := reqBody_message{
				Role:       "tool",
				ToolCallID: i.CallID,
			}
			if i.Error != nil {
				msg.Content = fmt.Sprintf("Error: %s", i.Error.Message)
			} else {
				msg.Content = *i.Result
			}
			b.Messages = append(b.Messages, msg)
		case *UserMessage:
			if len(b.Messages) > 0 && b.Messages[len(b.Messages)-1].Role == "user" {
				b.Messages[len(b.Messages)-1].Content += "\n\n" + i.Content
				continue
			}
			b.Messages = append(b.Messages, reqBody_message{
				Role:    "user",
				Content: i.Content,
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

func (m *openRouterModel) spec(spec []byte) ([]byte, error) {
	if strings.HasPrefix(m.modelName, "google/") {
		return m.specGoogle(spec)
	}
	return spec, nil
}

func (m *openRouterModel) specGoogle(spec []byte) ([]byte, error) {
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
	type GoogleToolSpec struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Parameters  *struct {
			Type       string `json:"type"`
			Properties map[string]struct {
				Type        string `json:"type"`
				Description string `json:"description"`
			} `json:"properties"`
			Required             []string `json:"required"`
			AdditionalProperties bool     `json:"additionalProperties"`
		} `json:"parameters,omitempty"`
	}
	var openAITool OpenAIToolSpec
	if err := json.Unmarshal(spec, &openAITool); err != nil {
		return nil, err
	}
	var googleTool GoogleToolSpec
	googleTool.Name = openAITool.Name
	googleTool.Description = openAITool.Description
	if openAITool.Parameters.Type == "object" && len(openAITool.Parameters.Properties) > 0 {
		googleTool.Parameters = &openAITool.Parameters
	}
	return json.Marshal(googleTool)
}
