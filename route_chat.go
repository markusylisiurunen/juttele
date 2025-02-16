package juttele

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/markusylisiurunen/juttele/internal/db"
)

func (app *App) handleChatRoute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chatID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || chatID <= 0 {
		http.Error(w, "invalid chat ID", http.StatusBadRequest)
		return
	}
	// list the chat events
	events, err := app.db.ListChatEvents(ctx, db.ListChatEventsArgs{ChatID: chatID})
	if err != nil {
		http.Error(w, fmt.Sprintf("error listing chat events: %v", err), http.StatusInternalServerError)
		return
	}
	// construct the response
	type respHistoryItem struct {
		Kind string          `json:"kind"`
		Data json.RawMessage `json:"data"`
	}
	type resp struct {
		History []respHistoryItem `json:"history"`
	}
	vv := resp{History: make([]respHistoryItem, 0, len(events.Items))}
	for _, i := range events.Items {
		if !strings.HasPrefix(i.Kind, "message.") {
			continue
		}
		type Content struct {
			Content string `json:"content"`
		}
		var content Content
		if err := json.Unmarshal(i.Content, &content); err != nil {
			http.Error(w, fmt.Sprintf("error decoding chat event: %v", err), http.StatusInternalServerError)
			return
		}
		vv.History = append(vv.History, respHistoryItem{
			Kind: "message",
			Data: must(json.Marshal(struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{
				Role:    strings.TrimPrefix(i.Kind, "message."),
				Content: content.Content,
			})),
		})
	}
	// send the response
	if err := json.NewEncoder(w).Encode(vv); err != nil {
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
	}
}
