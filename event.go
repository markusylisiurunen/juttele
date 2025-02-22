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
	chatEventMessageUser      = "message.user"
)

//---

func parseChatEvent(ts time.Time, uuid string, kind string, content []byte) (ChatEvent, error) {
	switch kind {
	case chatEventMessageAssistant:
		var v AssistantMessageChatEvent
		v.ts = ts
		v.uuid = uuid
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

type AssistantMessageChatEvent struct {
	ts      time.Time
	uuid    string
	content string
}

func NewAssistantMessageChatEvent(content string) *AssistantMessageChatEvent {
	return &AssistantMessageChatEvent{
		time.Now(),
		uuid.Must(uuid.NewV7()).String(),
		content,
	}
}

func (e *AssistantMessageChatEvent) getChatEvent() (string, string, []byte) {
	type content struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	c := content{
		Role:    "assistant",
		Content: e.content,
	}
	return e.uuid, chatEventMessageAssistant, util.Must(json.Marshal(c))
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
