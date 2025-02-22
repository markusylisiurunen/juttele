package juttele

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/markusylisiurunen/juttele/internal/repo"
	"github.com/tidwall/gjson"
)

type dataResponseHistoryItemMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type dataResponseHistoryItemReasoning struct {
	Content string `json:"content"`
}
type dataResponseHistoryItem struct {
	Kind string `json:"kind"`
	Data any    `json:"data"`
}
type dataResponseChat struct {
	ID        int64                     `json:"id"`
	CreatedAt string                    `json:"created_at"`
	Title     string                    `json:"title"`
	History   []dataResponseHistoryItem `json:"history"`
}
type dataResponse struct {
	Chats []dataResponseChat `json:"chats"`
}

func (app *App) dataRouteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var v dataResponse
	v.Chats = make([]dataResponseChat, 0)
	chats, err := app.repo.ListChats(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("error listing chats: %v", err), http.StatusInternalServerError)
		return
	}
	for _, chat := range chats.Items {
		events, err := app.repo.ListChatEvents(ctx, repo.ListChatEventsArgs{ChatID: chat.ID})
		if err != nil {
			http.Error(w, fmt.Sprintf("error listing chat events: %v", err), http.StatusInternalServerError)
			return
		}
		var vi dataResponseChat
		vi.ID = chat.ID
		vi.CreatedAt = chat.CreatedAt.Format(time.RFC3339Nano)
		vi.Title = chat.Title
		vi.History = make([]dataResponseHistoryItem, 0, len(events.Items))
		for _, i := range events.Items {
			if strings.HasPrefix(i.Kind, "message.") {
				vi.History = append(vi.History, dataResponseHistoryItem{
					Kind: "message",
					Data: dataResponseHistoryItemMessage{
						Role:    strings.TrimPrefix(i.Kind, "message."),
						Content: gjson.GetBytes(i.Content, "content").String(),
					},
				})
			}
			if i.Kind == "other.reasoning" {
				vi.History = append(vi.History, dataResponseHistoryItem{
					Kind: "reasoning",
					Data: dataResponseHistoryItemReasoning{
						Content: gjson.GetBytes(i.Content, "content").String(),
					},
				})
			}
		}
		v.Chats = append(v.Chats, vi)
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
	}
}
