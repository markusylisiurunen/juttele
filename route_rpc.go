package juttele

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/markusylisiurunen/juttele/internal/repo"
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
