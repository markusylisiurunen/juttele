package juttele

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/markusylisiurunen/juttele/internal/util"
)

func streamWithTools(
	ctx context.Context, tools *ToolCatalog, history *[]Message, request func() <-chan Result[Message],
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

func streamOpenAI(resp *http.Response, forceThinking bool) <-chan Result[Message] {
	type respToolCall struct {
		Index    int64  `json:"index"`
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Name string `json:"name"`
			Args string `json:"arguments"`
		} `json:"function"`
	}
	type respSchema struct {
		Choices []struct {
			Delta struct {
				Reasoning        string         `json:"reasoning"`
				ReasoningContent string         `json:"reasoning_content"`
				Content          string         `json:"content"`
				ToolCalls        []respToolCall `json:"tool_calls"`
			} `json:"delta"`
		} `json:"choices"`
	}
	out := make(chan Result[Message])
	go func() {
		defer close(out)
		events := streamSSE(resp)
		msg := NewAssistantMessage("", "")

		// Create a thinking block if needed
		var thinkingContent string
		if forceThinking {
			thinkingContent = "Thinking, just a second... "
			msg.SetMeta("thinking", thinkingContent)
		}

		out <- Ok[Message](msg)
		toolBuffer := make([]*respToolCall, 64)

		for event := range events {
			if event.Err != nil {
				out <- Err[Message](event.Err)
				return
			}
			if event.Val.T1 != "message" {
				continue
			}
			if string(event.Val.T2) == "[DONE]" {
				break
			}

			var b respSchema
			if err := json.Unmarshal(event.Val.T2, &b); err != nil {
				out <- Err[Message](err)
				return
			}
			if len(b.Choices) == 0 {
				continue
			}

			delta := b.Choices[0].Delta

			// Handle reasoning/thinking content
			if !forceThinking && (delta.Reasoning != "" || delta.ReasoningContent != "") {
				var reasoning string
				for _, i := range []string{
					delta.Reasoning,
					delta.ReasoningContent,
				} {
					if reasoning == "" {
						reasoning = i
					}
				}

				thinkingContent += reasoning
				msg.SetMeta("thinking", thinkingContent)
				out <- Ok[Message](msg)
			}

			// Handle regular content
			if delta.Content != "" {
				msg.Content += delta.Content
				out <- Ok[Message](msg)
			}

			// Handle tool calls
			for _, t := range delta.ToolCalls {
				if len(toolBuffer) <= int(t.Index) {
					out <- Err[Message](errors.New("tool call index out of range"))
					return
				}
				if toolBuffer[t.Index] == nil {
					c := respToolCall{}
					c.Index = t.Index
					c.ID = t.ID
					c.Type = t.Type
					c.Function.Name = t.Function.Name
					c.Function.Args = t.Function.Args
					toolBuffer[t.Index] = &c
				}
				toolBuffer[t.Index].Function.Args += t.Function.Args
			}

			if len(delta.ToolCalls) > 0 {
				// Clear existing tool calls and regenerate them
				msg.ToolCalls = nil
				for _, t := range toolBuffer {
					if t == nil || t.Type != "function" {
						continue
					}
					msg.AppendToolCall(t.ID, t.Function.Name, t.Function.Args)
				}
				out <- Ok[Message](msg)
			}
		}
	}()
	return out
}

func streamAnthropic(resp *http.Response) <-chan Result[Message] {
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
			PartialJSON string `json:"partial_json"`
		} `json:"delta"`
	}
	out := make(chan Result[Message])
	go func() {
		defer close(out)
		events := streamSSE(resp)
		msg := NewAssistantMessage("", "")
		out <- Ok[Message](msg)

		type toolBufferItem struct {
			ID       string
			FuncName string
			FuncArgs string
		}
		toolBuffer := make([]*toolBufferItem, 64)
		var thinkingContent string

		for event := range events {
			if event.Err != nil {
				out <- Err[Message](event.Err)
				return
			}

			if event.Val.T1 == "content_block_start" {
				var b respContentBlockStart
				if err := json.Unmarshal([]byte(event.Val.T2), &b); err != nil {
					out <- Err[Message](err)
					return
				}
				if b.ContentBlock.Type == "tool_use" {
					if len(toolBuffer) <= b.Index {
						out <- Err[Message](errors.New("tool call index out of range"))
						return
					}
					toolBuffer[b.Index] = &toolBufferItem{
						ID:       b.ContentBlock.ID,
						FuncName: b.ContentBlock.Name,
						FuncArgs: "",
					}

					// Clear and rebuild tool calls
					msg.ToolCalls = nil
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
					return
				}

				// Handle thinking content
				if b.Delta.Thinking != "" {
					thinkingContent += b.Delta.Thinking
					msg.SetMeta("thinking", thinkingContent)
					out <- Ok[Message](msg)
				}

				// Handle regular text content
				if b.Delta.Text != "" {
					msg.Content += b.Delta.Text
					out <- Ok[Message](msg)
				}

				// Handle partial JSON for tool calls
				if b.Delta.PartialJSON != "" {
					if len(toolBuffer) <= b.Index {
						out <- Err[Message](errors.New("tool call index out of range"))
						return
					}
					toolBuffer[b.Index].FuncArgs += b.Delta.PartialJSON

					// Clear and rebuild tool calls
					msg.ToolCalls = nil
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
		}

		// Ensure tool calls have valid args
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

func streamSSE(resp *http.Response) <-chan Result[util.Tuple[string, []byte]] {
	out := make(chan Result[util.Tuple[string, []byte]])
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
					out <- Ok(util.NewTuple(event, data))
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
			out <- Ok(util.NewTuple(event, data))
		}
		if err := scanner.Err(); err != nil {
			out <- Err[util.Tuple[string, []byte]](err)
		}
	}()
	return out
}
