package juttele

import (
	"context"
	"encoding/json"
)

type Tool interface {
	Name() string
	Spec() []byte
	Call(context.Context, string) (string, error)
}

//---

func toAnthropicToolSpec(spec []byte) ([]byte, error) {
	type OpenAIToolSpec struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Parameters  struct {
			Type       string `json:"type"`
			Properties map[string]struct {
				Type        string `json:"type"`
				Description string `json:"description"`
			} `json:"properties"`
			Required             []string `json:"required"`
			AdditionalProperties bool     `json:"additionalProperties"`
		} `json:"parameters"`
	}
	type AnthropicToolSpec struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		InputSchema struct {
			Type       string `json:"type"`
			Properties map[string]struct {
				Type        string `json:"type"`
				Description string `json:"description"`
			} `json:"properties"`
			Required []string `json:"required"`
		} `json:"input_schema"`
	}
	var openAITool OpenAIToolSpec
	if err := json.Unmarshal(spec, &openAITool); err != nil {
		return nil, err
	}
	var anthropicTool AnthropicToolSpec
	anthropicTool.Name = openAITool.Name
	anthropicTool.Description = openAITool.Description
	anthropicTool.InputSchema.Type = openAITool.Parameters.Type
	anthropicTool.InputSchema.Properties = make(map[string]struct {
		Type        string `json:"type"`
		Description string `json:"description"`
	})
	for k, v := range openAITool.Parameters.Properties {
		anthropicTool.InputSchema.Properties[k] = struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		}{
			Type:        v.Type,
			Description: v.Description,
		}
	}
	anthropicTool.InputSchema.Required = openAITool.Parameters.Required
	return json.Marshal(anthropicTool)
}

//---

type funcTool struct {
	name string
	spec []byte
	fn   func(context.Context, string) (string, error)
}

func NewFuncTool(name string, spec []byte, fn func(context.Context, string) (string, error)) Tool {
	return &funcTool{name: name, spec: spec, fn: fn}
}

func (f *funcTool) Name() string {
	return f.name
}

func (f *funcTool) Spec() []byte {
	return f.spec
}

func (f *funcTool) Call(ctx context.Context, input string) (string, error) {
	return f.fn(ctx, input)
}
