package juttele

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/markusylisiurunen/juttele/internal/repo"
	"github.com/markusylisiurunen/juttele/internal/util"
)

type sendRequest struct {
	ModelID       string `json:"model_id"`
	PersonalityID string `json:"personality_id"`
	Content       string `json:"content"`
}

func (app *App) sendRouteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chatID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("error parsing chat ID: %v", err), http.StatusBadRequest)
		return
	}
	var v sendRequest
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		http.Error(w, fmt.Sprintf("error decoding request: %v", err), http.StatusBadRequest)
		return
	}
	if chatID <= 0 || v.ModelID == "" || v.PersonalityID == "" || v.Content == "" {
		http.Error(w, "chat ID, model ID, personality ID, and content must be provided", http.StatusBadRequest)
		return
	}
	modelIdx := slices.IndexFunc(app.models, func(model Model) bool { return model.GetModelInfo().ID == v.ModelID })
	if modelIdx == -1 {
		http.Error(w, fmt.Sprintf("model with ID %q not found", v.ModelID), http.StatusNotFound)
		return
	}
	model := app.models[modelIdx]
	var systemPrompt *string
	for _, i := range model.GetModelInfo().Personalities {
		if i.ID == v.PersonalityID {
			v := i.SystemPrompt
			systemPrompt = &v
			break
		}
	}
	if systemPrompt == nil {
		http.Error(w, fmt.Sprintf("personality with ID %q not found", v.PersonalityID), http.StatusNotFound)
		return
	}
	events, err := app.repo.ListChatEvents(ctx, repo.ListChatEventsArgs{ChatID: chatID})
	if err != nil {
		http.Error(w, fmt.Sprintf("error listing chat events: %v", err), http.StatusInternalServerError)
		return
	}
	history := make([]ChatEvent, 0, 1+len(events.Items))
	for _, i := range events.Items {
		if !strings.HasPrefix(i.Kind, "message.") {
			continue
		}
		event, err := parseChatEvent(i.CreatedAt, i.UUID, i.Kind, i.Content)
		if err != nil {
			http.Error(w, fmt.Sprintf("error parsing chat event: %v", err), http.StatusInternalServerError)
			return
		}
		history = append(history, event)
	}
	history = append(history, NewUserMessageChatEvent(v.Content))
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	var (
		reasoning  strings.Builder
		completion strings.Builder
	)
	out := model.StreamCompletion(r.Context(), *systemPrompt, history)
	for i := range out {
		if i == nil {
			continue
		}
		switch i := i.(type) {
		case *AssistantMessageChatEvent:
			data := map[string]string{"content": i.content[completion.Len():]}
			completion.Reset()
			completion.WriteString(i.content)
			fmt.Fprintf(w, "data: %s\n\n", util.Must(json.Marshal(data)))
			flusher.Flush()
			continue
		}
	}
	if _, err := app.repo.CreateChatEvent(ctx, repo.CreateChatEventArgs{
		ChatID:  chatID,
		UUID:    uuid.Must(uuid.NewV7()).String(),
		Kind:    "message.user",
		Content: util.Must(json.Marshal(map[string]any{"content": v.Content})),
	}); err != nil {
		fmt.Printf("error creating chat event: %v\n", err)
		return
	}
	if reasoning.Len() > 0 {
		if _, err := app.repo.CreateChatEvent(ctx, repo.CreateChatEventArgs{
			ChatID:  chatID,
			UUID:    uuid.Must(uuid.NewV7()).String(),
			Kind:    "other.reasoning",
			Content: util.Must(json.Marshal(map[string]any{"content": reasoning.String()})),
		}); err != nil {
			fmt.Printf("error creating chat event: %v\n", err)
			return
		}
	}
	if completion.Len() > 0 {
		if _, err := app.repo.CreateChatEvent(ctx, repo.CreateChatEventArgs{
			ChatID:  chatID,
			UUID:    uuid.Must(uuid.NewV7()).String(),
			Kind:    "message.assistant",
			Content: util.Must(json.Marshal(map[string]any{"content": completion.String()})),
		}); err != nil {
			fmt.Printf("error creating chat event: %v\n", err)
			return
		}
	}
}
