package juttele

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type apiGenerateRequest_Model struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type apiGenerateRequest_Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type apiGenerateRequest_GenerationConfig struct {
	Temperature *float64 `json:"temperature"`
	Think       *bool    `json:"think"`
}
type apiGenerateRequest struct {
	Model            apiGenerateRequest_Model            `json:"model"`
	System           *string                             `json:"system"`
	Messages         []apiGenerateRequest_Message        `json:"messages"`
	GenerationConfig apiGenerateRequest_GenerationConfig `json:"generation_config"`
}

type apiGenerateResponse struct {
	Message string `json:"message"`
}

func (app *App) apiGenerateRouteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// decode and validate the request
	var request apiGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("error decoding request: %v", err), http.StatusBadRequest)
		return
	}
	if request.Model.ID == "" && request.Model.Name == "" {
		http.Error(w, "model ID or name is required", http.StatusBadRequest)
		return
	}
	if request.GenerationConfig.Temperature != nil && *request.GenerationConfig.Temperature < 0 {
		http.Error(w, "temperature must be non-negative", http.StatusBadRequest)
		return
	}
	if len(request.Messages) == 0 {
		http.Error(w, "at least one message is required", http.StatusBadRequest)
		return
	}
	// find the requested model
	var model Model
	if request.Model.ID != "" {
		for _, m := range app.models {
			if m.GetModelInfo().ID == request.Model.ID {
				model = m
				break
			}
		}
	} else {
		for _, m := range app.models {
			if m.GetModelInfo().Name == request.Model.Name {
				model = m
				break
			}
		}
	}
	if model == nil {
		http.Error(w, "unknown model", http.StatusBadRequest)
		return
	}
	// construct the message history
	messages := []Message{}
	if request.System != nil {
		messages = append(messages, NewSystemMessage(*request.System))
	}
	for _, m := range request.Messages {
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
	// create the generation config
	generationConfig := GenerationConfig{
		Temperature: nil,
		Think:       false,
		Tools:       nil,
	}
	if request.GenerationConfig.Temperature != nil {
		generationConfig.Temperature = request.GenerationConfig.Temperature
	}
	if request.GenerationConfig.Think != nil {
		generationConfig.Think = *request.GenerationConfig.Think
	}
	// stream the completion
	events := model.StreamCompletion(ctx, messages, generationConfig)
	var lastAssistantMessage *AssistantMessage
	for event := range events {
		if event.Err != nil {
			http.Error(w, fmt.Sprintf("error streaming completion: %v", event.Err), http.StatusInternalServerError)
			return
		}
		if v, ok := event.Val.(*AssistantMessage); ok {
			lastAssistantMessage = v
		}
	}
	// construct the response
	var response apiGenerateResponse
	if lastAssistantMessage != nil {
		response.Message = lastAssistantMessage.Content
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
		return
	}
}
