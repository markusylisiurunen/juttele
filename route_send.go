package juttele

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/markusylisiurunen/juttele/internal/repo"
	"github.com/markusylisiurunen/juttele/internal/util/jsonrpc"
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

func writeWSError(proxy *webSocketProxy, message string, err error) {
	errMsg := message
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", message, err)
	}
	resp := jsonrpc.NewNotification("error", map[string]any{"message": errMsg})
	proxy.write(resp)
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
	proxy := newWebSocketProxy(conn)
	defer proxy.close()
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		writeWSError(proxy, "error reading initial message", err)
		return
	}
	go proxy.readLoop()
	var v sendRequest
	if err := json.Unmarshal(msg, &v); err != nil {
		writeWSError(proxy, "error decoding request", err)
		return
	}
	if chatID <= 0 || v.ModelID == "" || v.PersonalityID == "" || v.Content == "" {
		writeWSError(proxy, "chat ID, model ID, personality ID, and content must be provided", nil)
		return
	}
	modelIdx := slices.IndexFunc(app.models, func(model Model) bool { return model.GetModelInfo().ID == v.ModelID })
	if modelIdx == -1 {
		writeWSError(proxy, fmt.Sprintf("model with ID %q not found", v.ModelID), nil)
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
		writeWSError(proxy, fmt.Sprintf("personality with ID %q not found", v.PersonalityID), nil)
		return
	}
	events, err := app.repo.ListChatEvents(ctx, repo.ListChatEventsArgs{ChatID: chatID})
	if err != nil {
		writeWSError(proxy, "error listing chat events", err)
		return
	}
	history := make([]ChatEvent, 0, 1+len(events.Items))
	for _, i := range events.Items {
		if !strings.HasPrefix(i.Kind, "message.") {
			continue
		}
		event, err := parseChatEvent(i.CreatedAt, i.UUID, i.Kind, i.Content)
		if err != nil {
			writeWSError(proxy, "error parsing chat event", err)
			return
		}
		history = append(history, event)
	}
	history = append(history, NewUserMessageChatEvent(v.Content))
	if err := app.upsertChatEvent(ctx, chatID, NewUserMessageChatEvent(v.Content)); err != nil {
		writeWSError(proxy, "error upserting chat event", err)
		return
	}
	opts := StreamCompletionOpts{
		SystemPrompt: *systemPrompt,
		Tools:        NewToolCatalog(),
		UseTools:     v.UseTools,
	}
	if opts.UseTools {
		for _, j := range app.tools {
			opts.Tools.Register(j)
		}
		for _, j := range v.Tools {
			opts.Tools.Register(newClientTool(proxy, j.Name, j.Spec))
		}
	}
	out := model.StreamCompletion(r.Context(), history, opts)
	for i := range out {
		if i.Err != nil {
			writeWSError(proxy, "error streaming completion", i.Err)
			return
		}
		switch i := i.Val.(type) {
		case *AssistantMessageChatEvent:
			if err := app.upsertChatEvent(ctx, chatID, i); err != nil {
				writeWSError(proxy, "error upserting chat event", err)
				return
			}
			if len(i.reasoning) > 0 {
				msg := jsonrpc.NewNotification("block", map[string]any{
					"id":      i.uuid + "_thinking",
					"type":    "thinking",
					"content": i.reasoning,
				})
				if err := proxy.write(msg); err != nil {
					writeWSError(proxy, "error writing block message", err)
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
				if err := proxy.write(msg); err != nil {
					writeWSError(proxy, "error writing block message", err)
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
				if err := proxy.write(msg); err != nil {
					writeWSError(proxy, "error writing block message", err)
					return
				}
			}
			continue
		case *ToolMessageChatEvent:
			if err := app.upsertChatEvent(ctx, chatID, i); err != nil {
				writeWSError(proxy, "error upserting chat event", err)
				return
			}
		}
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
