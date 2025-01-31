package juttele

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

type result[T any] struct {
	val T
	err error
}

func safeGo[T any](
	ctx context.Context,
	fn func(ctx context.Context, vs chan<- T, errs chan<- error),
) (<-chan result[T], context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	out := make(chan result[T], 1)
	go func() {
		defer close(out)
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}
				select {
				case out <- result[T]{err: fmt.Errorf("panic: %w; stack: %s", err, stack)}:
				case <-time.After(5 * time.Second):
				}
			}
		}()
		vs, errs := make(chan T), make(chan error)
		go func() {
			defer close(vs)
			defer close(errs)
			defer func() {
				if r := recover(); r != nil {
					stack := debug.Stack()
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}
					select {
					case errs <- fmt.Errorf("panic: %w; stack: %s", err, stack):
					case <-time.After(5 * time.Second):
					}
				}
			}()
			fn(ctx, vs, errs)
		}()
		var vClosed, errClosed bool
		for !vClosed || !errClosed {
			select {
			case v, ok := <-vs:
				if !ok {
					vClosed = true
					continue
				}
				out <- result[T]{val: v}
			case err, ok := <-errs:
				if !ok {
					errClosed = true
					continue
				}
				out <- result[T]{err: err}
			}
		}
	}()
	return out, cancel
}

func streamOpenAICompatibleCompletion(
	ctx context.Context,
	sendRequest func(context.Context) (*http.Response, error),
	processDataChunk func([]byte) (Chunk, error),
) <-chan Chunk {
	out, _ := safeGo(ctx, func(ctx context.Context, vs chan<- Chunk, errs chan<- error) {
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
			chunk, err := processDataChunk([]byte(line))
			if err != nil {
				errs <- fmt.Errorf("process error: %w", err)
				return
			}
			vs <- chunk
		}
	})
	ch := make(chan Chunk)
	go func() {
		defer close(ch)
		for v := range out {
			if v.err != nil {
				ch <- ErrorChunk(v.err)
				continue
			}
			ch <- v.val
		}
	}()
	return ch
}
