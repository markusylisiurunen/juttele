package juttele

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/markusylisiurunen/juttele/internal/logger"
	"github.com/markusylisiurunen/juttele/internal/repo"
)

type (
	dataResponse_Chat struct {
		ID     int64   `json:"id"`
		Ts     string  `json:"ts"`
		Title  string  `json:"title"`
		Blocks []Block `json:"blocks"`
	}
	dataResponse struct {
		Chats []dataResponse_Chat `json:"chats"`
	}
)

func (app *App) dataRouteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var v dataResponse
	v.Chats = make([]dataResponse_Chat, 0)
	chats, err := app.repo.ListChats(ctx)
	if err != nil {
		logger.Get().Error(fmt.Sprintf("error listing chats: %v", err))
		http.Error(w, fmt.Sprintf("error listing chats: %v", err), http.StatusInternalServerError)
		return
	}
	for _, chat := range chats.Items {
		vv := dataResponse_Chat{
			ID:     chat.ID,
			Ts:     chat.CreatedAt.Format(time.RFC3339),
			Title:  chat.Title,
			Blocks: make([]Block, 0),
		}
		blocks, err := app.repo.ListChatEvents(ctx, repo.ListChatEventsArgs{
			ChatID:     chat.ID,
			KindPrefix: "block.",
		})
		if err != nil {
			logger.Get().Error(fmt.Sprintf("error listing chat events: %v", err))
			http.Error(w, fmt.Sprintf("error listing chat events: %v", err), http.StatusInternalServerError)
			return
		}
		for _, block := range blocks.Items {
			b, err := parseBlock(block.Content)
			if err != nil {
				logger.Get().Error(fmt.Sprintf("error parsing block: %v", err))
				http.Error(w, fmt.Sprintf("error parsing block: %v", err), http.StatusInternalServerError)
				return
			}
			vv.Blocks = append(vv.Blocks, b)
		}
		v.Chats = append(v.Chats, vv)
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
	}
}
