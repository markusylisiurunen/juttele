package juttele

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

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

type CompletionOpts struct {
	UseTools    bool
	ClientTools []Tool
}

type Model interface {
	GetModelInfo() ModelInfo
	StreamCompletion(ctx context.Context, systemPrompt string, history []ChatEvent, opts CompletionOpts) <-chan ChatEvent
}

func callOpenAICompatibleAPI(
	ctx context.Context,
	sendRequest func(context.Context) (*http.Response, error),
) <-chan util.Result[[]byte] {
	out, _ := util.SafeGo(ctx, func(ctx context.Context, vs chan<- []byte, errs chan<- error) {
		resp, err := sendRequest(ctx)
		if err != nil {
			errs <- fmt.Errorf("request error: %w", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			errs <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			return
		}
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				errs <- fmt.Errorf("read error: %w", err)
				return
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			line = strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			if line == "[DONE]" {
				continue
			}
			vs <- []byte(line)
		}
	})
	return out
}

func callAnySSEAPI(
	ctx context.Context,
	sendRequest func(context.Context) (*http.Response, error),
) <-chan util.Result[util.Tuple[string, []byte]] {
	out, _ := util.SafeGo(ctx, func(ctx context.Context, vs chan<- util.Tuple[string, []byte], errs chan<- error) {
		resp, err := sendRequest(ctx)
		if err != nil {
			errs <- fmt.Errorf("request error: %w", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errs <- fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
			return
		}
		var event *string
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				errs <- fmt.Errorf("read error: %w", err)
				return
			}
			if strings.HasPrefix(line, "event: ") {
				line = strings.TrimSpace(strings.TrimPrefix(line, "event: "))
				event = &line
				continue
			}
			if event != nil {
				if strings.HasPrefix(line, "data: ") {
					line = strings.TrimSpace(strings.TrimPrefix(line, "data: "))
					vs <- util.NewTuple(*event, []byte(line))
				}
				event = nil
			} else {
				if strings.HasPrefix(line, "data: ") {
					line = strings.TrimSpace(strings.TrimPrefix(line, "data: "))
					vs <- util.NewTuple("", []byte(line))
				}
			}
		}
	})
	return out
}
