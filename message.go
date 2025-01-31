package juttele

type Role string

const (
	SystemRole    Role = "system"
	AssistantRole Role = "assistant"
	UserRole      Role = "user"
)

type Message struct {
	role     Role
	thinking string
	content  string
}

func SystemMessage(text string) Message              { return Message{SystemRole, "", text} }
func AssistantMessage(thinking, text string) Message { return Message{AssistantRole, thinking, text} }
func UserMessage(text string) Message                { return Message{UserRole, "", text} }

func (m Message) GetRole() Role      { return m.role }
func (m Message) GetContent() string { return m.content }
