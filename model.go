package juttele

import (
	"context"
	"strconv"

	"github.com/cespare/xxhash/v2"
	"github.com/markusylisiurunen/juttele/internal/util"
)

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

type GenerationConfig struct {
	JSON        bool
	MaxTokens   int64
	Temperature *float64
	Think       bool
	Tools       *ToolCatalog
}

type Model interface {
	GetModelInfo() ModelInfo
	StreamCompletion(context.Context, []Message, GenerationConfig) <-chan Result[Message]
}

type model struct {
	displayName   string
	maxTokens     int64
	personalities []ModelPersonality
	temperature   float64
}

func (m *model) getModelInfo(id string) ModelInfo {
	personalities := make([]ModelPersonality, len(m.personalities))
	copy(personalities, m.personalities)
	for i := range personalities {
		pid := xxhash.New()
		util.Must(pid.WriteString(id))
		util.Must(pid.WriteString(personalities[i].Name))
		personalities[i].ID = strconv.FormatUint(pid.Sum64(), 10)
	}
	return ModelInfo{
		ID:            id,
		Name:          m.displayName,
		Personalities: personalities,
	}
}

type modelOption func(*model)

func WithDisplayName(displayName string) modelOption {
	return func(m *model) {
		m.displayName = displayName
	}
}

func WithMaxTokens(maxTokens int64) modelOption {
	return func(m *model) {
		m.maxTokens = maxTokens
	}
}

func WithTemperature(temperature float64) modelOption {
	return func(m *model) {
		m.temperature = temperature
	}
}

func WithPersonality(name, systemPrompt string) modelOption {
	return func(m *model) {
		m.personalities = append(m.personalities, ModelPersonality{
			Name:         name,
			SystemPrompt: systemPrompt,
		})
	}
}
