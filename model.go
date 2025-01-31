package juttele

import (
	"context"
)

type chunkType string

const (
	errChunkType      chunkType = "error"
	thinkingChunkType chunkType = "thinking"
	contentChunkType  chunkType = "content"
)

type chunk struct {
	t        chunkType
	err      error
	thinking string
	content  string
}

func (c *chunk) getChunk() *chunk { return c }

func ErrorChunk(v error) *chunk     { return &chunk{t: errChunkType, err: v} }
func ThinkingChunk(v string) *chunk { return &chunk{t: thinkingChunkType, thinking: v} }
func ContentChunk(v string) *chunk  { return &chunk{t: contentChunkType, content: v} }

type Chunk interface {
	getChunk() *chunk
}

type ModelPersonality struct {
	ID           string
	Name         string
	SystemPrompt string
}

type ModelInfo struct {
	ID            string
	Name          string
	Personalities []ModelPersonality
}

type Model interface {
	GetModelInfo() ModelInfo
	StreamCompletion(ctx context.Context, history []Message) <-chan Chunk
}
