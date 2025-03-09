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

// var _ Model = (*googleModel)(nil)

// type googleModel struct {
// 	*model
// 	id        string
// 	apiKey    string
// 	modelName string
// }

// func NewGoogleModel(apiKey string, modelName string, opts ...modelOption) *googleModel {
// 	m := &googleModel{
// 		model:     &model{displayName: modelName},
// 		apiKey:    apiKey,
// 		modelName: modelName,
// 	}
// 	for _, opt := range opts {
// 		opt(m.model)
// 	}
// 	id := xxhash.New()
// 	util.Must(id.WriteString("google"))
// 	util.Must(id.WriteString(modelName))
// 	util.Must(id.WriteString(m.displayName))
// 	m.id = "google_" + strconv.FormatUint(id.Sum64(), 10)
// 	return m
// }

// func (m *googleModel) GetModelInfo() ModelInfo {
// 	return m.getModelInfo(m.id)
// }

// func (m *googleModel) StreamCompletion(
// 	ctx context.Context, history []ChatEvent, opts StreamCompletionOpts,
// ) <-chan Result[ChatEvent] {
// 	out := make(chan Result[ChatEvent])
// 	go func() {
// 		defer close(out)
// 		resp, err := m.request(ctx, history, opts)
// 		if err != nil {
// 			out <- Err[ChatEvent](err)
// 			return
// 		}
// 		events := streamSSE(resp)
// 		msg := NewAssistantMessageChatEvent("")
// 		out <- Ok[ChatEvent](msg)
// 		for event := range events {
// 			if event.Err != nil {
// 				out <- Err[ChatEvent](event.Err)
// 				return
// 			}
// 			if event.Val.T1 != "message" {
// 				continue
// 			}
// 			if string(event.Val.T2) == "[DONE]" {
// 				break
// 			}
// 			type respBody struct {
// 				Candidates []struct {
// 					Content struct {
// 						Parts []struct {
// 							Text string `json:"text"`
// 						} `json:"parts"`
// 					} `json:"content"`
// 				} `json:"candidates"`
// 			}
// 			var b respBody
// 			if err := json.Unmarshal(event.Val.T2, &b); err != nil {
// 				out <- Err[ChatEvent](err)
// 				return
// 			}
// 			if len(b.Candidates) == 0 {
// 				continue
// 			}
// 			if len(b.Candidates[0].Content.Parts) == 0 {
// 				continue
// 			}
// 			msg.content += b.Candidates[0].Content.Parts[0].Text
// 			out <- Ok[ChatEvent](msg)
// 		}
// 	}()
// 	return out
// }

// func (m *googleModel) request(
// 	ctx context.Context, history []ChatEvent, opts StreamCompletionOpts,
// ) (*http.Response, error) {
// 	type reqContentPart struct {
// 		Text string `json:"text"`
// 	}
// 	type reqContent struct {
// 		Role  *string          `json:"role,omitempty"`
// 		Parts []reqContentPart `json:"parts"`
// 	}
// 	type reqGenerationConfig struct {
// 		MaxOutputTokens int64   `json:"maxOutputTokens,omitempty"`
// 		Temperature     float64 `json:"temperature,omitempty"`
// 	}
// 	type reqBody struct {
// 		Contents          []reqContent         `json:"contents"`
// 		GenerationConfig  *reqGenerationConfig `json:"generationConfig,omitempty"`
// 		SystemInstruction *reqContent          `json:"systemInstruction,omitempty"`
// 	}
// 	b := reqBody{
// 		Contents: []reqContent{},
// 	}
// 	b.GenerationConfig = &reqGenerationConfig{
// 		MaxOutputTokens: m.maxTokens,
// 		Temperature:     m.temperature,
// 	}
// 	// add system instructions if they exist
// 	if len(opts.SystemPrompt) > 0 {
// 		loc, _ := time.LoadLocation("Europe/Helsinki")
// 		now := time.Now().In(loc).Format("Monday 2006-01-02 15:04:05")
// 		b.SystemInstruction = &reqContent{
// 			Parts: []reqContentPart{
// 				{Text: strings.ReplaceAll(opts.SystemPrompt, "{{current_time}}", now)},
// 			},
// 		}
// 	}
// 	// add the message history
// 	for _, i := range history {
// 		switch i := i.(type) {
// 		case *AssistantMessageChatEvent:
// 			role := "model"
// 			content := reqContent{
// 				Role:  &role,
// 				Parts: []reqContentPart{{Text: i.content}},
// 			}
// 			b.Contents = append(b.Contents, content)
// 		case *UserMessageChatEvent:
// 			role := "user"
// 			content := reqContent{
// 				Role:  &role,
// 				Parts: []reqContentPart{{Text: i.content}},
// 			}
// 			b.Contents = append(b.Contents, content)
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
// 		http.MethodPost, fmt.Sprintf("https://generativelanguage.googleapis.com/v1alpha/models/%s:streamGenerateContent", m.modelName),
// 		&buf,
// 	)
// 	if err != nil {
// 		return nil, err
// 	}
// 	q := req.URL.Query()
// 	q.Add("alt", "sse")
// 	q.Add("key", m.apiKey)
// 	req.URL.RawQuery = q.Encode()
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
