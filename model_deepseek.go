package juttele

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"strconv"
// 	"strings"
// 	"time"

// 	"github.com/cespare/xxhash/v2"
// 	"github.com/markusylisiurunen/juttele/internal/util"
// )

// var _ Model = (*deepSeekModel)(nil)

// type deepSeekModel struct {
// 	*model
// 	id        string
// 	apiKey    string
// 	modelName string
// }

// func NewDeepSeekModel(apiKey string, modelName string, opts ...modelOption) *deepSeekModel {
// 	m := &deepSeekModel{
// 		model:     &model{displayName: modelName},
// 		apiKey:    apiKey,
// 		modelName: modelName,
// 	}
// 	for _, opt := range opts {
// 		opt(m.model)
// 	}
// 	id := xxhash.New()
// 	util.Must(id.WriteString("deepseek"))
// 	util.Must(id.WriteString(modelName))
// 	util.Must(id.WriteString(m.displayName))
// 	m.id = "deepseek_" + strconv.FormatUint(id.Sum64(), 10)
// 	return m
// }

// func (m *deepSeekModel) GetModelInfo() ModelInfo {
// 	return m.getModelInfo(m.id)
// }

// func (m *deepSeekModel) StreamCompletion(
// 	ctx context.Context, history []ChatEvent, opts StreamCompletionOpts,
// ) <-chan Result[ChatEvent] {
// 	out := make(chan Result[ChatEvent], 1)
// 	defer close(out)
// 	resp, err := m.request(ctx, history, opts)
// 	if err != nil {
// 		out <- Err[ChatEvent](err)
// 		return out
// 	}
// 	return streamOpenAI(resp, false)
// }

// func (m *deepSeekModel) request(
// 	ctx context.Context, history []ChatEvent, opts StreamCompletionOpts,
// ) (*http.Response, error) {
// 	type reqTextMessage struct {
// 		Role    string `json:"role"`
// 		Content string `json:"content"`
// 	}
// 	type reqBody struct {
// 		MaxTokens int64  `json:"max_tokens,omitempty"`
// 		Messages  []any  `json:"messages"`
// 		Model     string `json:"model"`
// 		Stream    bool   `json:"stream"`
// 	}
// 	b := reqBody{
// 		MaxTokens: m.maxTokens,
// 		Messages:  []any{},
// 		Model:     m.modelName,
// 		Stream:    true,
// 	}
// 	// append the system prompt
// 	if len(opts.SystemPrompt) > 0 {
// 		loc, _ := time.LoadLocation("Europe/Helsinki")
// 		now := time.Now().In(loc).Format("Monday 2006-01-02 15:04:05")
// 		b.Messages = append(b.Messages, reqTextMessage{
// 			Role:    "system",
// 			Content: strings.ReplaceAll(opts.SystemPrompt, "{{current_time}}", now),
// 		})
// 	}
// 	// append the message history
// 	for _, i := range history {
// 		switch i := i.(type) {
// 		case *AssistantMessageChatEvent:
// 			b.Messages = append(b.Messages, reqTextMessage{
// 				Role:    "assistant",
// 				Content: i.content,
// 			})
// 		case *UserMessageChatEvent:
// 			b.Messages = append(b.Messages, reqTextMessage{
// 				Role:    "user",
// 				Content: i.content,
// 			})
// 		}
// 	}
// 	// make the request
// 	var buf bytes.Buffer
// 	encoder := json.NewEncoder(&buf)
// 	encoder.SetEscapeHTML(false)
// 	if err := encoder.Encode(b); err != nil {
// 		return nil, err
// 	}
// 	req, err := http.NewRequestWithContext(ctx,
// 		http.MethodPost, "https://api.deepseek.com/v1/chat/completions", &buf)
// 	if err != nil {
// 		return nil, err
// 	}
// 	req.Header.Set("Authorization", "Bearer "+m.apiKey)
// 	req.Header.Set("Content-Type", "application/json")
// 	resp, err := http.DefaultClient.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if resp.StatusCode != http.StatusOK {
// 		body, err := io.ReadAll(resp.Body)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if err := resp.Body.Close(); err != nil {
// 			return nil, err
// 		}
// 		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
// 	}
// 	return resp, nil
// }
