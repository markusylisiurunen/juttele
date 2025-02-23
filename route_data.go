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

type dataResponseHistoryItemToolCallFunction struct {
	Name string `json:"name"`
	Args string `json:"arguments"`
}
type dataResponseHistoryItemToolCall struct {
	ID       string                                  `json:"id"`
	Function dataResponseHistoryItemToolCallFunction `json:"function"`
}
type dataResponseHistoryItemMessage struct {
	Role      string                            `json:"role"`
	Content   string                            `json:"content"`
	ToolCalls []dataResponseHistoryItemToolCall `json:"tool_calls,omitempty"`
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
				reasoning := gjson.GetBytes(i.Content, "reasoning")
				if reasoning.Exists() {
					vi.History = append(vi.History, dataResponseHistoryItem{
						Kind: "reasoning",
						Data: dataResponseHistoryItemReasoning{
							Content: reasoning.String(),
						},
					})
				}
				itemData := dataResponseHistoryItemMessage{
					Role:    strings.TrimPrefix(i.Kind, "message."),
					Content: gjson.GetBytes(i.Content, "content").String(),
				}
				if gjson.GetBytes(i.Content, "tool_calls").Exists() {
					itemData.ToolCalls = make([]dataResponseHistoryItemToolCall, 0)
					for _, tc := range gjson.GetBytes(i.Content, "tool_calls").Array() {
						itemData.ToolCalls = append(itemData.ToolCalls, dataResponseHistoryItemToolCall{
							ID: tc.Get("id").String(),
							Function: dataResponseHistoryItemToolCallFunction{
								Name: tc.Get("function.name").String(),
								Args: tc.Get("function.arguments").String(),
							},
						})
					}
				}
				vi.History = append(vi.History, dataResponseHistoryItem{
					Kind: "message",
					Data: itemData,
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
