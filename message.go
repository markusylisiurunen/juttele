package juttele

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type MessageType string

const (
	MessageTypeSystem    MessageType = "system"
	MessageTypeAssistant MessageType = "assistant"
	MessageTypeUser      MessageType = "user"
	MessageTypeTool      MessageType = "tool"
)

type Message interface {
	GetID() string
	GetType() MessageType
	SetMeta(key, value string)
	GetMeta(key string) (string, bool)
	MarshalJSON() ([]byte, error)
}

type BaseMessage struct {
	ID   string            `json:"id"`
	Type MessageType       `json:"type"`
	Meta map[string]string `json:"meta,omitempty"`
}

func newBaseMessage(messageType MessageType) BaseMessage {
	return BaseMessage{
		ID:   uuid.Must(uuid.NewV7()).String(),
		Type: messageType,
	}
}

func (m *BaseMessage) GetID() string {
	return m.ID
}

func (m *BaseMessage) GetType() MessageType {
	return m.Type
}

func (m *BaseMessage) SetMeta(key, value string) {
	if m.Meta == nil {
		m.Meta = make(map[string]string)
	}
	m.Meta[key] = value
}

func (m *BaseMessage) GetMeta(key string) (string, bool) {
	if m.Meta == nil {
		return "", false
	}
	value, ok := m.Meta[key]
	return value, ok
}

type SystemMessage struct {
	BaseMessage
	Content string `json:"content"`
}

func NewSystemMessage(content string) *SystemMessage {
	return &SystemMessage{
		BaseMessage: newBaseMessage(MessageTypeSystem),
		Content:     content,
	}
}

func (m *SystemMessage) MarshalJSON() ([]byte, error) {
	type Alias SystemMessage
	return json.Marshal((*Alias)(m))
}

type AssistantMessageToolCall struct {
	CallID   string `json:"call_id"`
	FuncName string `json:"func_name"`
	FuncArgs string `json:"func_args"`
}

type AssistantMessage struct {
	BaseMessage
	Thinking  string                     `json:"thinking,omitempty"`
	Content   string                     `json:"content"`
	ToolCalls []AssistantMessageToolCall `json:"tool_calls,omitempty"`
}

func NewAssistantMessage(content string) *AssistantMessage {
	return &AssistantMessage{
		BaseMessage: newBaseMessage(MessageTypeAssistant),
		Thinking:    "",
		Content:     content,
	}
}

func (m *AssistantMessage) AppendThinking(thinking string) {
	m.Thinking += thinking
}

func (m *AssistantMessage) AppendContent(content string) {
	m.Content += content
}

func (m *AssistantMessage) ClearToolCalls() {
	m.ToolCalls = nil
}

func (m *AssistantMessage) AppendToolCall(callID, funcName, funcArgs string) {
	m.ToolCalls = append(m.ToolCalls, AssistantMessageToolCall{
		CallID:   callID,
		FuncName: funcName,
		FuncArgs: funcArgs,
	})
}

func (m *AssistantMessage) MarshalJSON() ([]byte, error) {
	type Alias AssistantMessage
	return json.Marshal((*Alias)(m))
}

type UserMessage struct {
	BaseMessage
	Content string `json:"content"`
}

func NewUserMessage(content string) *UserMessage {
	return &UserMessage{
		BaseMessage: newBaseMessage(MessageTypeUser),
		Content:     content,
	}
}

func (m *UserMessage) MarshalJSON() ([]byte, error) {
	type Alias UserMessage
	return json.Marshal((*Alias)(m))
}

type ToolMessage struct {
	BaseMessage
	CallID string  `json:"call_id"`
	Result *string `json:"result,omitempty"`
	Error  *struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewToolMessage(callID string) *ToolMessage {
	return &ToolMessage{
		BaseMessage: newBaseMessage(MessageTypeTool),
		CallID:      callID,
	}
}

func (m *ToolMessage) SetResult(result string) {
	m.Result = &result
	m.Error = nil
}

func (m *ToolMessage) SetError(code int64, message string) {
	m.Result = nil
	m.Error = &struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	}{
		Code:    code,
		Message: message,
	}
}

func (m *ToolMessage) MarshalJSON() ([]byte, error) {
	type Alias ToolMessage
	return json.Marshal((*Alias)(m))
}

func parseMessage(data []byte) (Message, error) {
	var baseMessage struct {
		Type MessageType `json:"type"`
	}
	if err := json.Unmarshal(data, &baseMessage); err != nil {
		return nil, err
	}
	switch baseMessage.Type {
	case MessageTypeSystem:
		var message SystemMessage
		if err := json.Unmarshal(data, &message); err != nil {
			return nil, err
		}
		return &message, nil
	case MessageTypeAssistant:
		var message AssistantMessage
		if err := json.Unmarshal(data, &message); err != nil {
			return nil, err
		}
		return &message, nil
	case MessageTypeUser:
		var message UserMessage
		if err := json.Unmarshal(data, &message); err != nil {
			return nil, err
		}
		return &message, nil
	case MessageTypeTool:
		var message ToolMessage
		if err := json.Unmarshal(data, &message); err != nil {
			return nil, err
		}
		return &message, nil
	default:
		return nil, fmt.Errorf("unknown message type: %q", baseMessage.Type)
	}
}
