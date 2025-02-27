package juttele

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/markusylisiurunen/juttele/internal/repo"
	"github.com/markusylisiurunen/juttele/internal/util"
	"github.com/tidwall/gjson"
)

type rpcRequest struct {
	Op   string          `json:"op"`
	Args json.RawMessage `json:"args"`
}

func (app *App) rpcRouteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var v rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		http.Error(w, fmt.Sprintf("error decoding request: %v", err), http.StatusBadRequest)
		return
	}
	var (
		rpcResp json.RawMessage
		rpcErr  error
	)
	switch v.Op {
	case "create_chat":
		rpcResp, rpcErr = app.rpcCreateChat(ctx, v.Args)
	case "rename_chat":
		rpcResp, rpcErr = app.rpcRenameChat(ctx, v.Args)
	default:
		rpcErr = fmt.Errorf("unknown op: %q", v.Op)
	}
	if rpcErr != nil {
		http.Error(w, fmt.Sprintf("error handling rpc request: %v", rpcErr), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(rpcResp); err != nil {
		http.Error(w, fmt.Sprintf("error writing response: %v", err), http.StatusInternalServerError)
		return
	}
}

func (app *App) rpcCreateChat(ctx context.Context, args []byte) ([]byte, error) {
	title := gjson.GetBytes(args, "title").String()
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	id, err := app.repo.CreateChat(ctx, repo.CreateChatArgs{
		Title: title,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating chat: %w", err)
	}
	type resp struct {
		ChatID int64 `json:"chat_id"`
	}
	return json.Marshal(resp{ChatID: id})
}

func (app *App) rpcRenameChat(ctx context.Context, args []byte) ([]byte, error) {
	chatID := gjson.GetBytes(args, "id").Int()
	if chatID == 0 {
		return nil, fmt.Errorf("id is required")
	}
	modelID := gjson.GetBytes(args, "model_id").String()
	if modelID == "" {
		return nil, fmt.Errorf("model_id is required")
	}
	var model Model
	modelFound := false
	for _, m := range app.models {
		if m.GetModelInfo().ID == modelID {
			model = m
			modelFound = true
			break
		}
	}
	if !modelFound {
		return nil, fmt.Errorf("model with ID %q not found", modelID)
	}
	var systemPrompt = `
Your task is to come up with a short name for the following conversation between the user and an LLM assistant.
The name should be concise (roughly 3 to 10 words) and descriptive, capturing the main theme or topic of the conversation.
Do not capitalize every word, only the first word and any proper nouns should be capitalized.
The name should be in English and should not contain any special characters or punctuation.

Here is the conversation:
{{conversation}}

Your response must be exactly one line long, and should only include the name.

<example_response>
Name: A conversation about the weather
<example_response>
	`
	systemPrompt = strings.TrimSpace(systemPrompt)
	conversation := []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{}
	events, err := app.repo.ListChatEvents(ctx, repo.ListChatEventsArgs{ChatID: chatID})
	if err != nil {
		return nil, fmt.Errorf("error listing chat events: %w", err)
	}
	for _, i := range events.Items {
		if !strings.HasPrefix(i.Kind, "message.") {
			continue
		}
		event, err := parseChatEvent(i.CreatedAt, i.UUID, i.Kind, i.Content)
		if err != nil {
			return nil, fmt.Errorf("error parsing chat event: %w", err)
		}
		if i.Kind == chatEventMessageAssistant {
			conversation = append(conversation, struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{
				Role:    "assistant",
				Content: event.(*AssistantMessageChatEvent).content,
			})
		} else if i.Kind == chatEventMessageUser {
			conversation = append(conversation, struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{
				Role:    "user",
				Content: event.(*UserMessageChatEvent).content,
			})
		}
	}
	systemPrompt = strings.ReplaceAll(systemPrompt,
		"{{conversation}}",
		string(util.Must(json.MarshalIndent(conversation, "", "  "))),
	)
	opts := CompletionOpts{}
	out := model.StreamCompletion(ctx, systemPrompt, []ChatEvent{
		NewUserMessageChatEvent("Please rename the chat."),
		NewAssistantMessageChatEvent("Name:"),
	}, opts)
	var completion string
	for i := range out {
		if i == nil {
			continue
		}
		switch i := i.(type) {
		case *AssistantMessageChatEvent:
			completion = i.content
			continue
		}
	}
	completion = strings.TrimSpace(completion)
	if len(completion) <= 3 || len(completion) >= 256 || strings.Contains(completion, "\n") {
		return nil, fmt.Errorf("error getting completion: %q", completion)
	}
	if err := app.repo.UpdateChat(ctx, repo.UpdateChatArgs{
		ID:    chatID,
		Title: completion,
	}); err != nil {
		return nil, fmt.Errorf("error updating chat: %w", err)
	}
	type resp struct {
		Ok bool `json:"ok"`
	}
	return json.Marshal(resp{Ok: true})
}
