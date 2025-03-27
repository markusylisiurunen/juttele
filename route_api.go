package juttele

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type apiRequest_Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type apiRequest struct {
	Model    string               `json:"model"`
	System   string               `json:"system"`
	Messages []apiRequest_Message `json:"messages"`
}

type apiResponse struct {
	Message string `json:"message"`
}

func (app *App) apiRouteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var in apiRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, fmt.Sprintf("error decoding request: %v", err), http.StatusBadRequest)
		return
	}
	var model Model
	for _, m := range app.models {
		if m.GetModelInfo().Name == in.Model {
			model = m
			break
		}
	}
	if model == nil {
		http.Error(w, fmt.Sprintf("unknown model: %q", in.Model), http.StatusBadRequest)
		return
	}
	messages := []Message{}
	if in.System != "" {
		messages = append(messages, NewSystemMessage(in.System))
	}
	for _, m := range in.Messages {
		switch m.Role {
		case "user":
			messages = append(messages, NewUserMessage(m.Content))
		case "assistant":
			messages = append(messages, NewAssistantMessage(m.Content))
		default:
			http.Error(w, fmt.Sprintf("unknown role: %q", m.Role), http.StatusBadRequest)
			return
		}
	}
	events := model.StreamCompletion(ctx, messages, GenerationConfig{Tools: NewToolCatalog()})
	var last *AssistantMessage
	for event := range events {
		if event.Err != nil {
			http.Error(w, fmt.Sprintf("error streaming completion: %v", event.Err), http.StatusInternalServerError)
			return
		}
		if v, ok := event.Val.(*AssistantMessage); ok {
			last = v
		}
	}
	var v apiResponse
	v.Message = last.Content
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
	}
}
