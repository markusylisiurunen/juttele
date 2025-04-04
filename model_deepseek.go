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

var _ Model = (*deepSeekModel)(nil)

type deepSeekModel struct {
	*model
	id        string
	apiKey    string
	modelName string
}

func NewDeepSeekModel(apiKey string, modelName string, opts ...modelOption) *deepSeekModel {
	m := &deepSeekModel{
		model:     &model{displayName: modelName},
		apiKey:    apiKey,
		modelName: modelName,
	}
	for _, opt := range opts {
		opt(m.model)
	}
	id := xxhash.New()
	util.Must(id.WriteString("deepseek"))
	util.Must(id.WriteString(modelName))
	util.Must(id.WriteString(m.displayName))
	m.id = "deepseek_" + strconv.FormatUint(id.Sum64(), 10)
	return m
}

func (m *deepSeekModel) GetModelInfo() ModelInfo {
	return m.getModelInfo(m.id)
}

func (m *deepSeekModel) StreamCompletion(
	ctx context.Context, history []Message, opts GenerationConfig,
) <-chan Result[Message] {
	out := make(chan Result[Message], 1)
	defer close(out)
	resp, err := m.request(ctx, history, opts)
	if err != nil {
		out <- Err[Message](err)
		return out
	}
	return streamOpenAI(resp)
}

func (m *deepSeekModel) request(
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
	type reqBody_responseFormat struct {
		Type string `json:"type"`
	}
	type reqBody struct {
		MaxTokens      int64                   `json:"max_tokens,omitempty"`
		Messages       []reqBody_message       `json:"messages"`
		Model          string                  `json:"model"`
		ResponseFormat *reqBody_responseFormat `json:"response_format,omitempty"`
		Stream         bool                    `json:"stream"`
		Temperature    float64                 `json:"temperature"`
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
	if opts.JSON {
		b.ResponseFormat = &reqBody_responseFormat{
			Type: "json_object",
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
		http.MethodPost, "https://api.deepseek.com/v1/chat/completions", &buf)
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
