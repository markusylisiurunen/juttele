package juttele

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/markusylisiurunen/juttele/internal/repo"
	"github.com/markusylisiurunen/juttele/internal/util/jsonrpc"
	"github.com/tidwall/gjson"
)

type sendRequestTool struct {
	Name string          `json:"name"`
	Spec json.RawMessage `json:"spec"`
}

type sendRequest struct {
	ModelID       string            `json:"model_id"`
	PersonalityID string            `json:"personality_id"`
	Content       string            `json:"content"`
	Tools         []sendRequestTool `json:"tools"`
	UseTools      bool              `json:"use_tools"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func writeWSError(conn *websocket.Conn, message string, err error) {
	errMsg := message
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", message, err)
	}
	resp := jsonrpc.NewNotification("error", map[string]any{"message": errMsg})
	conn.WriteJSON(resp)
}

func (app *App) sendRouteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chatID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("error parsing chat ID: %v", err), http.StatusBadRequest)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("error upgrading to websocket: %v", err), http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return
	}
	var v sendRequest
	if err := json.Unmarshal(msg, &v); err != nil {
		writeWSError(conn, "error decoding request", err)
		return
	}
	if chatID <= 0 || v.ModelID == "" || v.PersonalityID == "" || v.Content == "" {
		writeWSError(conn, "chat ID, model ID, personality ID, and content must be provided", nil)
		return
	}
	modelIdx := slices.IndexFunc(app.models, func(model Model) bool { return model.GetModelInfo().ID == v.ModelID })
	if modelIdx == -1 {
		writeWSError(conn, fmt.Sprintf("model with ID %q not found", v.ModelID), nil)
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
		writeWSError(conn, fmt.Sprintf("personality with ID %q not found", v.PersonalityID), nil)
		return
	}
	events, err := app.repo.ListChatEvents(ctx, repo.ListChatEventsArgs{ChatID: chatID})
	if err != nil {
		writeWSError(conn, "error listing chat events", err)
		return
	}
	history := make([]ChatEvent, 0, 1+len(events.Items))
	for _, i := range events.Items {
		if !strings.HasPrefix(i.Kind, "message.") {
			continue
		}
		event, err := parseChatEvent(i.CreatedAt, i.UUID, i.Kind, i.Content)
		if err != nil {
			writeWSError(conn, "error parsing chat event", err)
			return
		}
		history = append(history, event)
	}
	history = append(history, NewUserMessageChatEvent(v.Content))
	if err := app.upsertChatEvent(ctx, chatID, NewUserMessageChatEvent(v.Content)); err != nil {
		writeWSError(conn, "error upserting chat event", err)
		return
	}
	opts := CompletionOpts{}
	opts.UseTools = v.UseTools
	if len(v.Tools) > 0 {
		opts.ClientTools = make([]Tool, len(v.Tools))
		for i, j := range v.Tools {
			// FIXME: this is absolute garbage
			opts.ClientTools[i] = NewFuncTool(j.Name, j.Spec, func(ctx context.Context, args string) (string, error) {
				id := int64(math.Round(rand.Float64() * 256))
				req := jsonrpc.NewRequest(int(id), "tool_call", map[string]any{
					"name": j.Name,
					"args": args,
				})
				if err := conn.WriteJSON(req); err != nil {
					return "", err
				}
				_, msg, err := conn.ReadMessage()
				if err != nil {
					return "", err
				}
				if gjson.GetBytes(msg, "id").Int() != id {
					return "", fmt.Errorf("tool call failed: invalid ID %d", id)
				}
				result := gjson.GetBytes(msg, "result")
				if !result.Exists() {
					return "", fmt.Errorf("tool call failed: %s", msg)
				}
				return result.String(), nil
			})
		}
	}
	out := model.StreamCompletion(r.Context(), *systemPrompt, history, opts)
	for i := range out {
		if i == nil {
			continue
		}
		switch i := i.(type) {
		case *AssistantMessageChatEvent:
			if err := app.upsertChatEvent(ctx, chatID, i); err != nil {
				writeWSError(conn, "error upserting chat event", err)
				return
			}
			if len(i.reasoning) > 0 {
				msg := jsonrpc.NewNotification("block", map[string]any{
					"id":      i.uuid + "_thinking",
					"type":    "thinking",
					"content": i.reasoning,
				})
				if err := conn.WriteJSON(msg); err != nil {
					return
				}
			}
			if len(i.content) > 0 {
				msg := jsonrpc.NewNotification("block", map[string]any{
					"id":      i.uuid,
					"type":    "text",
					"role":    "assistant",
					"content": i.content,
				})
				if err := conn.WriteJSON(msg); err != nil {
					return
				}
			}
			for _, i := range i.toolCalls {
				msg := jsonrpc.NewNotification("block", map[string]any{
					"id":   i.ID,
					"type": "tool_call",
					"name": i.FuncName,
					"args": i.FuncArgs,
				})
				if err := conn.WriteJSON(msg); err != nil {
					return
				}
			}
			continue
		case *ToolMessageChatEvent:
			if err := app.upsertChatEvent(ctx, chatID, i); err != nil {
				writeWSError(conn, "error upserting chat event", err)
				return
			}
		}
	}
	if err := conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	); err != nil {
		fmt.Printf("error writing close message: %v", err)
	}
}

func (app *App) upsertChatEvent(ctx context.Context, id int64, event ChatEvent) error {
	uuid, kind, content := event.getChatEvent()
	if _, err := app.repo.CreateChatEvent(ctx, repo.CreateChatEventArgs{
		ChatID:  id,
		UUID:    uuid,
		Kind:    kind,
		Content: content,
	}); err != nil {
		return err
	}
	return nil
}
