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
		util.Must(id.WriteString("google"))
		util.Must(id.WriteString("personality"))
		util.Must(id.WriteString(m.id))
		util.Must(id.WriteString(m.displayName))
		util.Must(id.WriteString(name))
		m.personalities = append(m.personalities, googleModelPersonality{
			id:           strconv.FormatUint(id.Sum64(), 10),
			name:         name,
			systemPrompt: systemPrompt,
		})
	}
}

func NewGoogleModel(apiKey string, modelName string, opts ...googleModelOption) *googleModel {
	id := xxhash.New()
	util.Must(id.WriteString("google"))
	util.Must(id.WriteString(modelName))
	m := &googleModel{
		apiKey:        apiKey,
		modelName:     modelName,
		displayName:   modelName,
		personalities: []googleModelPersonality{},
	}
	for _, opt := range opts {
		opt(m)
	}
	util.Must(id.WriteString(m.displayName))
	m.id = "google_" + strconv.FormatUint(id.Sum64(), 10)
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

func (m *googleModel) StreamCompletion(ctx context.Context, systemPrompt string, history []ChatEvent, _ CompletionOpts) <-chan ChatEvent {
	out := make(chan ChatEvent)
	go func() {
		defer close(out)
		chunks := callOpenAICompatibleAPI(ctx,
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
				if systemPrompt != "" {
					loc, _ := time.LoadLocation("Europe/Helsinki")
					now := time.Now().In(loc).Format("Monday 2006-01-02 15:04:05")
					b.SystemInstruction = &reqContent{
						Parts: []reqContentPart{
							{Text: strings.ReplaceAll(systemPrompt, "{{current_time}}", now)},
						},
					}
				}
				// add the message history
				for _, i := range history {
					switch i := i.(type) {
					case *AssistantMessageChatEvent:
						role := "model"
						b.Contents = append(b.Contents, reqContent{
							Role:  &role,
							Parts: []reqContentPart{{Text: i.content}},
						})
					case *UserMessageChatEvent:
						role := "user"
						b.Contents = append(b.Contents, reqContent{
							Role:  &role,
							Parts: []reqContentPart{{Text: i.content}},
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
					http.MethodPost, fmt.Sprintf("https://generativelanguage.googleapis.com/v1alpha/models/%s:streamGenerateContent", m.modelName),
					&buf,
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
		)
		resp := NewAssistantMessageChatEvent("")
		out <- resp
		for r := range chunks {
			if r.Err != nil {
				fmt.Printf("error: %v\n", r.Err)
				return
			}
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
			if err := json.Unmarshal([]byte(r.Val), &b); err != nil {
				fmt.Printf("error: %v\n", err)
				return
			}
			if len(b.Candidates) == 0 {
				return
			}
			if len(b.Candidates[0].Content.Parts) == 0 {
				return
			}
			resp.content += b.Candidates[0].Content.Parts[0].Text
			out <- resp
		}
	}()
	return out
}
