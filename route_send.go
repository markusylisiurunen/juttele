package juttele

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/markusylisiurunen/juttele/internal/repo"
	"github.com/markusylisiurunen/juttele/internal/util"
)

func (app *App) handleStreamRoute(w http.ResponseWriter, r *http.Request) {
	type req struct {
		ModelID            string `json:"model_id"`
		ModelPersonalityID string `json:"model_personality_id"`
		History            []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"history"`
	}
	var v req
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		http.Error(w, fmt.Sprintf("error decoding request: %v", err), http.StatusBadRequest)
		return
	}
	if v.ModelID == "" || v.ModelPersonalityID == "" || len(v.History) == 0 {
		http.Error(w, "model ID, personality ID, and history are required", http.StatusBadRequest)
		return
	}
	// find the requested model
	modelIdx := slices.IndexFunc(app.models, func(model Model) bool {
		return model.GetModelInfo().ID == v.ModelID
	})
	if modelIdx == -1 {
		http.Error(w, fmt.Sprintf("model with ID %q not found", v.ModelID), http.StatusNotFound)
		return
	}
	model := app.models[modelIdx]
	// find the requested personality's system prompt
	var systemPrompt *string
	for _, i := range model.GetModelInfo().Personalities {
		if i.ID == v.ModelPersonalityID {
			v := i.SystemPrompt
			systemPrompt = &v
			break
		}
	}
	if systemPrompt == nil {
		http.Error(w, fmt.Sprintf("personality with ID %q not found", v.ModelPersonalityID), http.StatusNotFound)
		return
	}
	// construct the full conversation history
	history := make([]Message, 0, 1+len(v.History))
	// append the system prompt if defined
	if *systemPrompt != "" {
		history = append(history, SystemMessage(*systemPrompt))
	}
	// append the user and assistant messages
	for _, i := range v.History {
		switch i.Role {
		case "user":
			history = append(history, UserMessage(i.Content))
		case "assistant":
			history = append(history, AssistantMessage("", i.Content))
		default:
			http.Error(w, fmt.Sprintf("invalid role %q", i.Role), http.StatusBadRequest)
			return
		}
	}
	// send the headers
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	// stream the completion
	var (
		reasoning  strings.Builder
		completion strings.Builder
	)
	chunks := model.StreamCompletion(r.Context(), history)
	for i := range chunks {
		if i == nil {
			continue
		}
		chunk := i.getChunk()
		if chunk.t == errChunkType {
			data := map[string]string{"error": chunk.err.Error()}
			fmt.Fprintf(w, "data: %s\n\n", util.Must(json.Marshal(data)))
			flusher.Flush()
			return
		}
		if chunk.t == thinkingChunkType {
			reasoning.WriteString(chunk.thinking)
			data := map[string]string{"thinking": chunk.thinking}
			fmt.Fprintf(w, "data: %s\n\n", util.Must(json.Marshal(data)))
			flusher.Flush()
			continue
		}
		if chunk.t == contentChunkType {
			completion.WriteString(chunk.content)
			data := map[string]string{"content": chunk.content}
			fmt.Fprintf(w, "data: %s\n\n", util.Must(json.Marshal(data)))
			flusher.Flush()
			continue
		}
	}
	// FIXME: this should definitely not be here
	ctx := r.Context()
	chatID, err := app.repo.CreateChat(ctx, repo.CreateChatArgs{
		Title: time.Now().UTC().Format("2006-01-02 15:04:05"),
	})
	if err != nil {
		fmt.Printf("error creating chat: %v\n", err)
		return
	}
	// append events from the conversation history
	for _, i := range history {
		var createEventErr error
		switch role := i.GetRole(); role {
		case SystemRole:
			_, createEventErr = app.repo.CreateChatEvent(ctx, repo.CreateChatEventArgs{
				ChatID:  chatID,
				Kind:    "message.system",
				Content: util.Must(json.Marshal(map[string]any{"content": i.GetContent()})),
			})
		case UserRole:
			_, createEventErr = app.repo.CreateChatEvent(ctx, repo.CreateChatEventArgs{
				ChatID:  chatID,
				Kind:    "message.user",
				Content: util.Must(json.Marshal(map[string]any{"content": i.GetContent()})),
			})
		case AssistantRole:
			_, createEventErr = app.repo.CreateChatEvent(ctx, repo.CreateChatEventArgs{
				ChatID:  chatID,
				Kind:    "message.assistant",
				Content: util.Must(json.Marshal(map[string]any{"content": i.GetContent()})),
			})
		}
		if createEventErr != nil {
			fmt.Printf("error creating chat event: %v\n", createEventErr)
			return
		}
	}
	// append reasoning and completion
	if reasoning.Len() > 0 {
		if _, err := app.repo.CreateChatEvent(ctx, repo.CreateChatEventArgs{
			ChatID:  chatID,
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
			Kind:    "message.assistant",
			Content: util.Must(json.Marshal(map[string]any{"content": completion.String()})),
		}); err != nil {
			fmt.Printf("error creating chat event: %v\n", err)
			return
		}
	}
}
