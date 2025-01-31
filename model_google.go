package juttele

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/cespare/xxhash/v2"
)

var _ Model = (*googleModel)(nil)

type googleModelPersonality struct {
	id           string
	name         string
	systemPrompt string
}

type googleModel struct {
	id            string
	apiKey        string
	modelName     string
	displayName   string
	personalities []googleModelPersonality
}

type googleModelOption func(*googleModel)

func WithGoogleModelDisplayName(name string) googleModelOption {
	return func(m *googleModel) {
		m.displayName = name
	}
}

func WithGoogleModelPersonality(name string, systemPrompt string) googleModelOption {
	return func(m *googleModel) {
		id := xxhash.New()
		must(id.WriteString("google"))
		must(id.WriteString("personality"))
		must(id.WriteString(m.id))
		must(id.WriteString(name))
		m.personalities = append(m.personalities, googleModelPersonality{
			id:           strconv.FormatUint(id.Sum64(), 10),
			name:         name,
			systemPrompt: systemPrompt,
		})
	}
}

func NewGoogleModel(apiKey string, modelName string, opts ...googleModelOption) *googleModel {
	id := xxhash.New()
	must(id.WriteString("google"))
	must(id.WriteString(modelName))
	m := &googleModel{
		id:            "google_" + strconv.FormatUint(id.Sum64(), 10),
		apiKey:        apiKey,
		modelName:     modelName,
		displayName:   modelName,
		personalities: []googleModelPersonality{},
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *googleModel) GetModelInfo() ModelInfo {
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

func (m *googleModel) StreamCompletion(ctx context.Context, history []Message) <-chan Chunk {
	return streamOpenAICompatibleCompletion(ctx,
		func(ctx context.Context) (*http.Response, error) {
			type reqContentPart struct {
				Text string `json:"text"`
			}
			type reqContent struct {
				Role  *string          `json:"role,omitempty"`
				Parts []reqContentPart `json:"parts"`
			}
			type reqBody struct {
				SystemInstruction *reqContent  `json:"systemInstruction,omitempty"`
				Contents          []reqContent `json:"contents"`
			}
			b := reqBody{
				Contents: make([]reqContent, 0),
			}
			// add system instructions if they exist
			systemMessageIdx := slices.IndexFunc(history, func(m Message) bool {
				return m.role == SystemRole
			})
			if systemMessageIdx != -1 {
				b.SystemInstruction = &reqContent{
					Parts: []reqContentPart{
						{Text: history[systemMessageIdx].content},
					},
				}
			}
			// add the message history
			for _, m := range history {
				if m.role == UserRole {
					role := "user"
					b.Contents = append(b.Contents, reqContent{
						Role: &role,
						Parts: []reqContentPart{
							{Text: m.content},
						},
					})
				}
				if m.role == AssistantRole {
					role := "model"
					b.Contents = append(b.Contents, reqContent{
						Role: &role,
						Parts: []reqContentPart{
							{Text: m.content},
						},
					})
				}
			}
			body, err := json.Marshal(b)
			if err != nil {
				return nil, err
			}
			req, err := http.NewRequestWithContext(ctx,
				http.MethodPost, fmt.Sprintf("https://generativelanguage.googleapis.com/v1alpha/models/%s:streamGenerateContent", m.modelName),
				bytes.NewReader(body),
			)
			if err != nil {
				return nil, err
			}
			q := req.URL.Query()
			q.Add("alt", "sse")
			q.Add("key", m.apiKey)
			req.URL.RawQuery = q.Encode()
			req.Header.Set("Content-Type", "application/json")
			return http.DefaultClient.Do(req)
		},
		func(chunk []byte) (Chunk, error) {
			type respBody struct {
				Candidates []struct {
					Content struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					} `json:"content"`
				} `json:"candidates"`
			}
			var b respBody
			if err := json.Unmarshal([]byte(chunk), &b); err != nil {
				return nil, err
			}
			if len(b.Candidates) == 0 {
				return nil, nil
			}
			if len(b.Candidates[0].Content.Parts) == 0 {
				return nil, nil
			}
			return ContentChunk(b.Candidates[0].Content.Parts[0].Text), nil
		},
	)
}
