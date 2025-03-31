package juttele

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type ToolBundle interface {
	Tools() []Tool
}

type ToolCatalog struct {
	mux   sync.RWMutex
	tools map[string]Tool
	order []string
}

func NewToolCatalog() *ToolCatalog {
	return &ToolCatalog{tools: make(map[string]Tool), order: make([]string, 0)}
}

func (tc *ToolCatalog) Copy() *ToolCatalog {
	tc.mux.RLock()
	defer tc.mux.RUnlock()
	out := NewToolCatalog()
	for _, name := range tc.order {
		out.Register(tc.tools[name])
	}
	return out
}

func (tc *ToolCatalog) Register(tool Tool) error {
	tc.mux.Lock()
	defer tc.mux.Unlock()
	if _, ok := tc.tools[tool.Name()]; ok {
		return fmt.Errorf("tool %q already registered", tool.Name())
	}
	tc.tools[tool.Name()] = tool
	tc.order = append(tc.order, tool.Name())
	return nil
}

func (tc *ToolCatalog) Count() int {
	tc.mux.RLock()
	defer tc.mux.RUnlock()
	return len(tc.tools)
}

func (tc *ToolCatalog) List() []Tool {
	tc.mux.RLock()
	defer tc.mux.RUnlock()
	out := make([]Tool, 0, len(tc.tools))
	for _, name := range tc.order {
		out = append(out, tc.tools[name])
	}
	return out
}

func (tc *ToolCatalog) Call(ctx context.Context, name, args string) (string, error) {
	tc.mux.RLock()
	tool, ok := tc.tools[name]
	tc.mux.RUnlock()
	if !ok {
		return "", fmt.Errorf("tool %q not found", name)
	}
	return tool.Call(ctx, args)
}

type Tool interface {
	Name() string
	Spec() []byte
	Call(context.Context, string) (string, error)
}

type funcTool struct {
	name string
	spec []byte
	fn   func(context.Context, string) (string, error)
}

func newFuncTool(name string, spec []byte, fn func(context.Context, string) (string, error)) Tool {
	return &funcTool{name: name, spec: spec, fn: fn}
}

func (r *funcTool) Name() string {
	return r.name
}

func (r *funcTool) Spec() []byte {
	return r.spec
}

func (r *funcTool) Call(ctx context.Context, args string) (string, error) {
	return r.fn(ctx, args)
}

type clientTool struct {
	proxy *webSocketProxy
	name  string
	spec  []byte
}

func newClientTool(proxy *webSocketProxy, name string, spec []byte) Tool {
	return &clientTool{proxy: proxy, name: name, spec: spec}
}

func (r *clientTool) Name() string {
	return r.name
}

func (r *clientTool) Spec() []byte {
	return r.spec
}

func (r *clientTool) Call(ctx context.Context, args string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	type request struct {
		Name string `json:"name"`
		Args string `json:"args"`
	}
	req, err := json.Marshal(request{Name: r.name, Args: args})
	if err != nil {
		return "", err
	}
	res, err := r.proxy.rpc(ctx, "tool_call", req)
	if err != nil {
		return "", err
	}
	var v string
	if err := json.Unmarshal(res, &v); err != nil {
		return "", err
	}
	return v, nil
}
