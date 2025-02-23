package juttele

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/markusylisiurunen/juttele/internal/util"
	"github.com/tidwall/gjson"
)

type ChatEvent interface {
	getChatEvent() (uuid string, kind string, content []byte)
}

const (
	chatEventMessageAssistant = "message.assistant"
	chatEventMessageTool      = "message.tool"
	chatEventMessageUser      = "message.user"
)

//---

func parseChatEvent(ts time.Time, uuid string, kind string, content []byte) (ChatEvent, error) {
	switch kind {
	case chatEventMessageAssistant:
		var v AssistantMessageChatEvent
		v.ts = ts
		v.uuid = uuid
		v.reasoning = gjson.GetBytes(content, "reasoning").String()
		v.content = gjson.GetBytes(content, "content").String()
		if gjson.GetBytes(content, "tool_calls").Exists() {
			toolCalls := gjson.GetBytes(content, "tool_calls").Array()
			v.toolCalls = make([]assistantMessageToolCall, len(toolCalls))
			for i, t := range toolCalls {
				v.toolCalls[i] = assistantMessageToolCall{
					ID:       t.Get("id").String(),
					Type:     t.Get("type").String(),
					FuncName: t.Get("function.name").String(),
					FuncArgs: t.Get("function.arguments").String(),
				}
			}
		}
		return &v, nil
	case chatEventMessageTool:
		var v ToolMessageChatEvent
		v.ts = ts
		v.uuid = uuid
		v.callID = gjson.GetBytes(content, "tool_call_id").String()
		v.content = gjson.GetBytes(content, "content").String()
		return &v, nil
	case chatEventMessageUser:
		var v UserMessageChatEvent
		v.ts = ts
		v.uuid = uuid
		v.content = gjson.GetBytes(content, "content").String()
		return &v, nil
	default:
		return nil, fmt.Errorf("unknown chat event kind: %s", kind)
	}
}

//---

type assistantMessageToolCall struct {
	ID       string
	Type     string
	FuncName string
	FuncArgs string
}

type AssistantMessageChatEvent struct {
	ts        time.Time
	uuid      string
	reasoning string
	content   string
	toolCalls []assistantMessageToolCall
}

func NewAssistantMessageChatEvent(content string) *AssistantMessageChatEvent {
	return &AssistantMessageChatEvent{
		time.Now(),
		uuid.Must(uuid.NewV7()).String(),
		"",
		content,
		nil,
	}
}

func (e *AssistantMessageChatEvent) getChatEvent() (string, string, []byte) {
	type toolCallFunc struct {
		Name string `json:"name"`
		Args string `json:"arguments"`
	}
	type toolCall struct {
		ID       string        `json:"id"`
		Type     string        `json:"type"`
		Function *toolCallFunc `json:"function,omitempty"`
	}
	type content struct {
		Role      string     `json:"role"`
		Reasoning string     `json:"reasoning,omitzero"`
		Content   string     `json:"content"`
		ToolCalls []toolCall `json:"tool_calls,omitempty"`
	}
	c := content{
		Role:      "assistant",
		Reasoning: e.reasoning,
		Content:   e.content,
	}
	if len(e.toolCalls) > 0 {
		toolCalls := make([]toolCall, len(e.toolCalls))
		for i, t := range e.toolCalls {
			toolCalls[i] = toolCall{
				ID:   t.ID,
				Type: t.Type,
				Function: &toolCallFunc{
					Name: t.FuncName,
					Args: t.FuncArgs,
				},
			}
		}
		c.ToolCalls = toolCalls
	}
	return e.uuid, chatEventMessageAssistant, util.Must(json.Marshal(c))
}

//---

type ToolMessageChatEvent struct {
	ts      time.Time
	uuid    string
	callID  string
	content string
}

func NewToolMessageChatEvent(callID string, content string) *ToolMessageChatEvent {
	return &ToolMessageChatEvent{
		time.Now(),
		uuid.Must(uuid.NewV7()).String(),
		callID,
		content,
	}
}

func (e *ToolMessageChatEvent) getChatEvent() (string, string, []byte) {
	type content struct {
		Role       string `json:"role"`
		Content    string `json:"content"`
		ToolCallID string `json:"tool_call_id"`
	}
	c := content{
		Role:       "tool",
		Content:    e.content,
		ToolCallID: e.callID,
	}
	return e.uuid, chatEventMessageTool, util.Must(json.Marshal(c))
}

//---

type UserMessageChatEvent struct {
	ts      time.Time
	uuid    string
	content string
}

func NewUserMessageChatEvent(content string) *UserMessageChatEvent {
	return &UserMessageChatEvent{
		time.Now(),
		uuid.Must(uuid.NewV7()).String(),
		content,
	}
}

func (e *UserMessageChatEvent) getChatEvent() (string, string, []byte) {
	type content struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	c := content{
		Role:    "user",
		Content: e.content,
	}
	return e.uuid, chatEventMessageUser, util.Must(json.Marshal(c))
}
