package juttele

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/markusylisiurunen/juttele/internal/util"
)

type BlockType string

const (
	BlockTypeThinking BlockType = "thinking"
	BlockTypeText     BlockType = "text"
	BlockTypeTool     BlockType = "tool"
	BlockTypeError    BlockType = "error"
)

type Block interface {
	GetID() string
	GetTimestamp() time.Time
	GetHash() string
	GetType() BlockType
	MarshalJSON() ([]byte, error)
}

type BaseBlock struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"ts"`
	Hash      string    `json:"hash"`
	Type      BlockType `json:"type"`
}

func (b *BaseBlock) GetID() string {
	return b.ID
}

func (b *BaseBlock) GetTimestamp() time.Time {
	return b.Timestamp
}

func (b *BaseBlock) GetHash() string {
	return b.Hash
}

func (b *BaseBlock) GetType() BlockType {
	return b.Type
}

func newBaseBlock(blockType BlockType, contentHash string) BaseBlock {
	return BaseBlock{
		ID:        uuid.Must(uuid.NewV7()).String(),
		Timestamp: time.Now().UTC(),
		Hash:      contentHash,
		Type:      blockType,
	}
}

type ThinkingBlock struct {
	BaseBlock
	Content  string `json:"content"`
	Duration int64  `json:"duration"`
}

func NewThinkingBlock(content string, duration int64) *ThinkingBlock {
	return &ThinkingBlock{
		BaseBlock: newBaseBlock(BlockTypeThinking, calculateBlockHash(content)),
		Content:   content,
		Duration:  duration,
	}
}

func (b *ThinkingBlock) Update(content string, duration int64) {
	b.Content = content
	b.Duration = duration
	b.Hash = calculateBlockHash(content)
}

func (b *ThinkingBlock) MarshalJSON() ([]byte, error) {
	type Alias ThinkingBlock
	return json.Marshal((*Alias)(b))
}

type TextBlock struct {
	BaseBlock
	Role    string `json:"role"`
	Content string `json:"content"`
}

func NewTextBlock(role, content string) *TextBlock {
	return &TextBlock{
		BaseBlock: newBaseBlock(BlockTypeText, calculateBlockHash(role+content)),
		Role:      role,
		Content:   content,
	}
}

func (b *TextBlock) Update(content string) {
	b.Content = content
	b.Hash = calculateBlockHash(b.Role + content)
}

func (b *TextBlock) MarshalJSON() ([]byte, error) {
	type Alias TextBlock
	return json.Marshal((*Alias)(b))
}

type ToolBlock struct {
	BaseBlock
	Name   string  `json:"name"`
	Args   string  `json:"args"`
	Result *string `json:"result,omitempty"`
	Error  *struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewToolBlock(name, args string) *ToolBlock {
	return &ToolBlock{
		BaseBlock: newBaseBlock(BlockTypeTool, calculateBlockHash(name+args)),
		Name:      name,
		Args:      args,
	}
}

func (b *ToolBlock) Update(name, args string) {
	b.Name = name
	b.Args = args
	b.Hash = calculateBlockHash(name + args)
}

func (b *ToolBlock) SetResult(result string) {
	b.Result = &result
	b.Error = nil
	b.Hash = calculateBlockHash(b.Name + b.Args + result)
}

func (b *ToolBlock) SetError(code int64, message string) {
	b.Result = nil
	b.Error = &struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	}{
		Code:    code,
		Message: message,
	}
	b.Hash = calculateBlockHash(b.Name + b.Args + fmt.Sprintf("%d%s", code, message))
}

func (b *ToolBlock) MarshalJSON() ([]byte, error) {
	type Alias ToolBlock
	return json.Marshal((*Alias)(b))
}

type ErrorBlock struct {
	BaseBlock
	Error struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func NewErrorBlock(code int64, message string) *ErrorBlock {
	return &ErrorBlock{
		BaseBlock: newBaseBlock(BlockTypeError, calculateBlockHash(fmt.Sprintf("%d%s", code, message))),
		Error: struct {
			Code    int64  `json:"code"`
			Message string `json:"message"`
		}{
			Code:    code,
			Message: message,
		},
	}
}

func (b *ErrorBlock) MarshalJSON() ([]byte, error) {
	type Alias ErrorBlock
	return json.Marshal((*Alias)(b))
}

func calculateBlockHash(content string) string {
	h := xxhash.New()
	util.Must(h.WriteString(content))
	return fmt.Sprintf("%d", h.Sum64())
}

func parseBlock(data []byte) (Block, error) {
	var baseBlock struct {
		Type BlockType `json:"type"`
	}
	if err := json.Unmarshal(data, &baseBlock); err != nil {
		return nil, err
	}
	switch baseBlock.Type {
	case BlockTypeThinking:
		var block ThinkingBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, err
		}
		return &block, nil
	case BlockTypeText:
		var block TextBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, err
		}
		return &block, nil
	case BlockTypeTool:
		var block ToolBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, err
		}
		return &block, nil
	case BlockTypeError:
		var block ErrorBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, err
		}
		return &block, nil
	default:
		return nil, fmt.Errorf("unknown block type: %q", baseBlock.Type)
	}
}
