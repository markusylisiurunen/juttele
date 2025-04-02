package juttele

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func streamWithTools(
	ctx context.Context,
	tools *ToolCatalog,
	history *[]Message,
	request func() <-chan Result[Message],
) <-chan Result[Message] {
	out := make(chan Result[Message])
	go func() {
		defer close(out)
	llm:
		var last *AssistantMessage
		for event := range request() {
			if event.Err != nil {
				out <- Err[Message](event.Err)
				return
			}
			if v, ok := event.Val.(*AssistantMessage); ok {
				last = v
			}
			out <- event
		}
		if last == nil || len(last.ToolCalls) == 0 {
			return
		}
		*history = append(*history, last)
		for _, t := range last.ToolCalls {
			result, err := tools.Call(ctx, t.FuncName, t.FuncArgs)
			if err != nil {
				msg := NewToolMessage(t.CallID)
				msg.SetError(-32603, err.Error())
				out <- Ok[Message](msg)
				*history = append(*history, msg)
			} else {
				msg := NewToolMessage(t.CallID)
				msg.SetResult(result)
				out <- Ok[Message](msg)
				*history = append(*history, msg)
			}
		}
		goto llm
	}()
	return out
}

//--------------------------------------------------------------------------------------------------

func streamAnthropic(resp *http.Response) <-chan Result[Message] {
	type toolBufferItem struct {
		ID       string
		FuncName string
		FuncArgs string
	}
	type respContentBlockStart struct {
		Type         string `json:"type"`
		Index        int    `json:"index"`
		ContentBlock struct {
			Type string `json:"type"`
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"content_block"`
	}
	type respContentBlockDelta struct {
		Type  string `json:"type"`
		Index int    `json:"index"`
		Delta struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			Thinking    string `json:"thinking"`
			Signature   string `json:"signature"`
			PartialJSON string `json:"partial_json"`
		} `json:"delta"`
	}
	out := make(chan Result[Message])
	go func() {
		defer close(out)
		events := streamServerSentEvents(resp)
		msg := NewAssistantMessage("")
		out <- Ok[Message](msg)
		toolBuffer := make([]*toolBufferItem, 64)
		stopReceived := false
		for event := range events {
			if event.Err != nil {
				out <- Err[Message](event.Err)
				return
			}
			if stopReceived {
				continue
			}
			if event.Val.T1 == "content_block_start" {
				var b respContentBlockStart
				if err := json.Unmarshal([]byte(event.Val.T2), &b); err != nil {
					out <- Err[Message](err)
					for range events {
						// NOTE: drain the channel to prevent blocking
					}
					return
				}
				if b.ContentBlock.Type == "tool_use" {
					if len(toolBuffer) <= b.Index {
						out <- Err[Message](errors.New("tool call index out of range, expected < 64"))
						for range events {
							// NOTE: drain the channel to prevent blocking
						}
						return
					}
					toolBuffer[b.Index] = &toolBufferItem{
						ID:       b.ContentBlock.ID,
						FuncName: b.ContentBlock.Name,
						FuncArgs: "",
					}
					msg.ClearToolCalls()
					for _, t := range toolBuffer {
						if t == nil {
							continue
						}
						msg.AppendToolCall(t.ID, t.FuncName, t.FuncArgs)
					}
					out <- Ok[Message](msg)
				}
				continue
			}
			if event.Val.T1 == "content_block_delta" {
				var b respContentBlockDelta
				if err := json.Unmarshal([]byte(event.Val.T2), &b); err != nil {
					out <- Err[Message](err)
					for range events {
						// NOTE: drain the channel to prevent blocking
					}
					return
				}
				if b.Delta.Thinking != "" {
					msg.AppendThinking(b.Delta.Thinking)
					out <- Ok[Message](msg)
				}
				if b.Delta.Signature != "" {
					msg.SetTransientMeta("signature", b.Delta.Signature)
					out <- Ok[Message](msg)
				}
				if b.Delta.Text != "" {
					msg.AppendContent(b.Delta.Text)
					out <- Ok[Message](msg)
				}
				if b.Delta.PartialJSON != "" {
					if len(toolBuffer) <= b.Index {
						out <- Err[Message](errors.New("tool call index out of range, expected < 64"))
						for range events {
							// NOTE: drain the channel to prevent blocking
						}
						return
					}
					toolBuffer[b.Index].FuncArgs += b.Delta.PartialJSON
					msg.ClearToolCalls()
					for _, t := range toolBuffer {
						if t == nil {
							continue
						}
						msg.AppendToolCall(t.ID, t.FuncName, t.FuncArgs)
					}
					out <- Ok[Message](msg)
				}
				continue
			}
			if event.Val.T1 == "message_stop" {
				stopReceived = true
			}
		}
		if !stopReceived {
			out <- Err[Message](errors.New("streaming ended without 'message_stop'"))
			return
		}
		if len(msg.ToolCalls) > 0 {
			for i := range msg.ToolCalls {
				if msg.ToolCalls[i].FuncArgs == "" {
					msg.ToolCalls[i].FuncArgs = "{}"
				}
			}
			out <- Ok[Message](msg)
		}
	}()
	return out
}

//--------------------------------------------------------------------------------------------------

func streamOpenAI(resp *http.Response) <-chan Result[Message] {
	type respSchema_toolCall_Function struct {
		Name string `json:"name"`
		Args string `json:"arguments"`
	}
	type respSchema_toolCall struct {
		Index    int64                        `json:"index"`
		ID       string                       `json:"id"`
		Type     string                       `json:"type"`
		Function respSchema_toolCall_Function `json:"function"`
	}
	type respSchema struct {
		Error *struct {
			Code     int             `json:"code"`
			Message  string          `json:"message"`
			Metadata json.RawMessage `json:"metadata"`
		} `json:"error"`
		Choices []struct {
			Delta struct {
				Reasoning        string                `json:"reasoning"`
				ReasoningContent string                `json:"reasoning_content"`
				Content          string                `json:"content"`
				ToolCalls        []respSchema_toolCall `json:"tool_calls"`
			} `json:"delta"`
		} `json:"choices"`
	}
	out := make(chan Result[Message])
	go func() {
		defer close(out)
		events := streamServerSentEvents(resp)
		msg := NewAssistantMessage("")
		out <- Ok[Message](msg)
		toolBuffer := make([]*respSchema_toolCall, 64)
		doneReceived := false
		for event := range events {
			if event.Err != nil {
				out <- Err[Message](event.Err)
				return
			}
			if doneReceived || event.Val.T1 != "message" {
				continue
			}
			if string(event.Val.T2) == "[DONE]" {
				doneReceived = true
				continue
			}
			var b respSchema
			if err := json.Unmarshal(event.Val.T2, &b); err != nil {
				out <- Err[Message](err)
				for range events {
					// NOTE: drain the channel to prevent blocking
				}
				return
			}
			if b.Error != nil {
				out <- Err[Message](fmt.Errorf("%s: %s", b.Error.Message, b.Error.Metadata))
				for range events {
					// NOTE: drain the channel to prevent blocking
				}
				return
			}
			if len(b.Choices) == 0 {
				continue
			}
			delta := b.Choices[0].Delta
			if delta.Reasoning != "" || delta.ReasoningContent != "" {
				var reasoning string
				if delta.Reasoning != "" {
					reasoning = delta.Reasoning
				}
				if delta.ReasoningContent != "" {
					reasoning = delta.ReasoningContent
				}
				msg.AppendThinking(reasoning)
				out <- Ok[Message](msg)
			}
			if delta.Content != "" {
				msg.AppendContent(delta.Content)
				out <- Ok[Message](msg)
			}
			for _, t := range delta.ToolCalls {
				if len(toolBuffer) <= int(t.Index) {
					out <- Err[Message](errors.New("tool call index out of range, expected < 64"))
					for range events {
						// NOTE: drain the channel to prevent blocking
					}
					return
				}
				if toolBuffer[t.Index] == nil {
					toolBuffer[t.Index] = &respSchema_toolCall{
						Index: t.Index,
						ID:    t.ID,
						Type:  t.Type,
						Function: respSchema_toolCall_Function{
							Name: t.Function.Name,
							Args: "",
						},
					}
				}
				toolBuffer[t.Index].Function.Args += t.Function.Args
			}
			if len(delta.ToolCalls) > 0 {
				msg.ClearToolCalls()
				for _, t := range toolBuffer {
					if t == nil || t.Type != "function" {
						continue
					}
					msg.AppendToolCall(t.ID, t.Function.Name, t.Function.Args)
				}
				out <- Ok[Message](msg)
			}
		}
		if !doneReceived {
			out <- Err[Message](errors.New("streaming ended without [DONE]"))
			return
		}
	}()
	return out
}

//--------------------------------------------------------------------------------------------------

func streamServerSentEvents(resp *http.Response) <-chan Result[Tuple[string, []byte]] {
	out := make(chan Result[Tuple[string, []byte]])
	go func() {
		defer close(out)
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		var (
			currentEvent string
			currentData  []string
		)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				if len(currentData) > 0 {
					event := currentEvent
					if event == "" {
						event = "message"
					}
					data := []byte(strings.Join(currentData, "\n"))
					out <- Ok(NewTuple(event, data))
					currentEvent = ""
					currentData = nil
				}
				continue
			}
			if i := strings.IndexRune(line, ':'); i > 0 {
				field := strings.ToLower(strings.TrimSpace(line[:i]))
				value := strings.TrimSpace(line[i+1:])
				switch field {
				case "event":
					currentEvent = value
				case "data":
					currentData = append(currentData, value)
				}
			}
		}
		if len(currentData) > 0 {
			event := currentEvent
			if event == "" {
				event = "message"
			}
			data := []byte(strings.Join(currentData, "\n"))
			out <- Ok(NewTuple(event, data))
		}
		if err := scanner.Err(); err != nil {
			out <- Err[Tuple[string, []byte]](err)
		}
	}()
	return out
}
