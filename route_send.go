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
	"github.com/markusylisiurunen/juttele/internal/logger"
	"github.com/markusylisiurunen/juttele/internal/repo"
	"github.com/markusylisiurunen/juttele/internal/util"
	"github.com/markusylisiurunen/juttele/internal/util/jsonrpc"
)

type sendRequestTool struct {
	Name string          `json:"name"`
	Spec json.RawMessage `json:"spec"`
}

type sendRequest struct {
	Method string `json:"method"`
	Params struct {
		ModelID       string            `json:"model_id"`
		PersonalityID string            `json:"personality_id"`
		Content       string            `json:"content"`
		Tools         []sendRequestTool `json:"tools"`
		UseTools      bool              `json:"use_tools"`
		Think         bool              `json:"think"`
	} `json:"params"`
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
	logger.Get().Error(errMsg)
	resp := jsonrpc.NewNotification("error", map[string]any{"message": errMsg})
	proxy.write(resp)
}

func (app *App) sendRouteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chatID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		logger.Get().Error(fmt.Sprintf("error parsing chat ID: %v", err))
		http.Error(w, fmt.Sprintf("error parsing chat ID: %v", err), http.StatusBadRequest)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Get().Error(fmt.Sprintf("error upgrading to websocket: %v", err))
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
	if v.Method != "generate" {
		writeWSError(proxy, "invalid method", nil)
		return
	}
	if chatID <= 0 || v.Params.ModelID == "" || v.Params.PersonalityID == "" || v.Params.Content == "" {
		writeWSError(proxy, "chat ID, model ID, personality ID, and content must be provided", nil)
		return
	}
	modelIdx := slices.IndexFunc(app.models, func(model Model) bool { return model.GetModelInfo().ID == v.Params.ModelID })
	if modelIdx == -1 {
		writeWSError(proxy, fmt.Sprintf("model with ID %q not found", v.Params.ModelID), nil)
		return
	}
	model := app.models[modelIdx]
	var systemPrompt *string
	for _, i := range model.GetModelInfo().Personalities {
		if i.ID == v.Params.PersonalityID {
			v := i.SystemPrompt
			systemPrompt = &v
			break
		}
	}
	if systemPrompt == nil {
		writeWSError(proxy, fmt.Sprintf("personality with ID %q not found", v.Params.PersonalityID), nil)
		return
	}
	isFirst, err := app.isFirstUserMessage(ctx, chatID)
	if err != nil {
		logger.Get().Error(fmt.Sprintf("error checking if first message: %v", err))
		isFirst = false
	}
	var titleChan chan string
	if isFirst {
		titleChan = app.generateChatTitle(ctx, v.Params.Content)
	}
	if err := app.upsertMessage(ctx, chatID, NewUserMessage(v.Params.Content)); err != nil {
		writeWSError(proxy, "error upserting user message", err)
		return
	}
	if err := app.upsertBlock(ctx, chatID, NewTextBlock("user", v.Params.Content)); err != nil {
		writeWSError(proxy, "error upserting user block", err)
		return
	}
	events, err := app.repo.ListChatEvents(ctx, repo.ListChatEventsArgs{
		ChatID:     chatID,
		KindPrefix: "message.",
	})
	if err != nil {
		writeWSError(proxy, "error listing chat events", err)
		return
	}
	history := make([]Message, 0, 1+len(events.Items))
	history = append(history, NewSystemMessage(*systemPrompt))
	for _, i := range events.Items {
		message, err := parseMessage(i.Content)
		if err != nil {
			writeWSError(proxy, "error parsing message", err)
			return
		}
		history = append(history, message)
	}
	// history = append(history, NewUserMessage(v.Content))
	opts := GenerationConfig{
		Tools: NewToolCatalog(),
		Think: v.Params.Think,
	}
	if v.Params.UseTools {
		for _, j := range app.tools {
			opts.Tools.Register(j)
		}
		for _, j := range v.Params.Tools {
			opts.Tools.Register(newClientTool(proxy, j.Name, j.Spec))
		}
	}
	out := model.StreamCompletion(r.Context(), history, opts)
	out2 := app.streamBlocks(ctx, chatID, out, titleChan, isFirst)
	for i := range out2 {
		msg := jsonrpc.NewNotification("block", i)
		if err := proxy.write(msg); err != nil {
			writeWSError(proxy, "error writing block message", err)
			return
		}
	}
}

func (app *App) streamBlocks(
	ctx context.Context, chatID int64, in <-chan Result[Message], titleChan chan string, isFirst bool,
) <-chan Block {
	begin := time.Now()
	out1 := make(chan Block)
	go func() {
		defer close(out1)
		var done bool
		blocks := map[string]Block{}
		toolBlocks := map[string]*ToolBlock{}
		for i := range in {
			if done {
				continue
			}
			if i.Err != nil {
				logger.Get().Error(fmt.Sprintf("error in stream: %v", i.Err))
				done = true
				out1 <- NewErrorBlock(-32603, i.Err.Error())
			} else {
				if err := app.upsertMessage(ctx, chatID, i.Val); err != nil {
					logger.Get().Error(fmt.Sprintf("error upserting message: %v", err))
					done = true
					out1 <- NewErrorBlock(-32603, fmt.Sprintf("error upserting message: %v", err))
					continue
				}
				switch i := i.Val.(type) {
				case *AssistantMessage:
					if i.Thinking != "" {
						id := i.GetID() + "_thinking"
						block, ok := blocks[id].(*ThinkingBlock)
						if !ok {
							block = NewThinkingBlock("", 0)
							blocks[id] = block
						}
						block.Update(i.Thinking, int64(time.Since(begin).Milliseconds()))
						out1 <- block
					}
					if i.Content != "" {
						id := i.GetID()
						block, ok := blocks[id].(*TextBlock)
						if !ok {
							block = NewTextBlock("assistant", "")
							blocks[id] = block
						}
						block.Update(i.Content)
						out1 <- block
					}
					if len(i.ToolCalls) > 0 {
						for _, j := range i.ToolCalls {
							id := i.GetID() + j.CallID
							block, ok := blocks[id].(*ToolBlock)
							if !ok {
								block = NewToolBlock("", "")
								blocks[id] = block
								toolBlocks[j.CallID] = block
							}
							block.Update(j.FuncName, j.FuncArgs)
							out1 <- block
						}
					}
				case *ToolMessage:
					id := i.CallID
					block, ok := toolBlocks[id]
					if !ok {
						continue
					}
					if i.Error != nil {
						block.SetError(i.Error.Code, i.Error.Message)
						out1 <- block
						continue
					}
					if i.Result != nil {
						block.SetResult(*i.Result)
						out1 <- block
						continue
					}
				default:
					logger.Get().Error(fmt.Sprintf("unknown message type: %T", i))
					done = true
					out1 <- NewErrorBlock(-32603, fmt.Sprintf("unknown message type: %T", i))
				}
			}
		}
	}()
	out2 := make(chan Block)
	go func() {
		defer close(out2)
		for i := range out1 {
			if err := app.upsertBlock(ctx, chatID, i); err != nil {
				logger.Get().Error(fmt.Sprintf("error upserting block: %v", err))
			}
			out2 <- i
		}
		if isFirst && titleChan != nil {
			title := <-titleChan
			if title == "" {
				logger.Get().Error("generated title is empty")
			} else {
				if err := app.repo.UpdateChat(ctx, repo.UpdateChatArgs{
					ID:    chatID,
					Title: title,
				}); err != nil {
					logger.Get().Error(fmt.Sprintf("error updating chat title: %v", err))
				}
			}
		}
	}()
	return out2
}

func (app *App) upsertMessage(ctx context.Context, chatID int64, message Message) error {
	return app.upsertChatEvent(ctx,
		chatID,
		message.GetID(),
		fmt.Sprintf("message.%s", message.GetType()),
		util.Must(message.MarshalJSON()),
	)
}

func (app *App) upsertBlock(ctx context.Context, chatID int64, block Block) error {
	return app.upsertChatEvent(ctx,
		chatID,
		block.GetID(),
		fmt.Sprintf("block.%s", block.GetType()),
		util.Must(block.MarshalJSON()),
	)
}

func (app *App) upsertChatEvent(
	ctx context.Context, chatID int64, eventUUID string, eventKind string, eventContent []byte,
) error {
	if _, err := app.repo.CreateChatEvent(ctx, repo.CreateChatEventArgs{
		ChatID:  chatID,
		UUID:    eventUUID,
		Kind:    eventKind,
		Content: eventContent,
	}); err != nil {
		return err
	}
	return nil
}

func (app *App) isFirstUserMessage(ctx context.Context, chatID int64) (bool, error) {
	events, err := app.repo.ListChatEvents(ctx, repo.ListChatEventsArgs{
		ChatID:     chatID,
		KindPrefix: "message.user",
	})
	if err != nil {
		return false, err
	}
	return len(events.Items) == 0, nil
}

func (app *App) generateChatTitle(ctx context.Context, content string) chan string {
	result := make(chan string, 1)
	go func() {
		defer close(result)
		// TODO: allow setting the "small but capable" model in the config
		titleModel := app.getSmallButCapableModel()
		if titleModel == nil {
			logger.Get().Error("no model found for title generation")
			return
		}
		systemPrompt := "You are a helpful assistant that generates concise, descriptive titles based on the user's first chat message. " +
			"IMPORTANT: Use sentence case, NOT title case. This means only capitalize the first word and proper nouns like names of people, places, or brands. " +
			"Create a short chat title (max 80 characters) that captures the essence of the user's initial message and the chat's context. " +
			"DO NOT include emojis, special characters or punctuation at the end of the title. " +
			"Respond with just the title, no quotes or explanations. Respond in the same language as the message. " +
			"Your response will be used directly as the chat title." +
			"\n\nExamples of CORRECT titles (sentence case):\n" +
			"- \"How to make a perfect omelette\" (NOT \"How To Make A Perfect Omelette\")\n" +
			"- \"Elokuvasuosituksia draaman ystäville\" (NOT \"Elokuvasuosituksia Draaman Ystäville\")\n" +
			"- \"Z-value spike implications\" (NOT \"Z-Value Spike Implications\")\n" +
			"- \"Planning a trip to London\" (NOT \"Planning A Trip To London\")\n" +
			"- \"Ways to improve productivity at work\" (NOT \"Ways To Improve Productivity At Work\")\n" +
			"\n\nRemember: Only capitalize the first word and proper nouns. Every other word should be lowercase."
		history := []Message{
			NewSystemMessage(systemPrompt),
			NewUserMessage("<initial_user_message>\n" + content + "\n<initial_user_message>"),
		}
		temp := 0.3
		opts := GenerationConfig{
			MaxTokens:   50,
			Temperature: &temp,
		}
		titleStream := titleModel.StreamCompletion(ctx, history, opts)
		var title string
		for res := range titleStream {
			if res.Err != nil {
				logger.Get().Error(fmt.Sprintf("error generating title: %v", res.Err))
				continue
			}
			if msg, ok := res.Val.(*AssistantMessage); ok && msg.Content != "" {
				title = msg.Content
			}
		}
		title = strings.TrimSpace(title)
		if title == "" {
			logger.Get().Error("generated title is empty")
			return
		}
		result <- title
	}()
	return result
}
