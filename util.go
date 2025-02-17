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

func streamOpenAICompatibleCompletion(
	ctx context.Context,
	sendRequest func(context.Context) (*http.Response, error),
	processDataChunk func([]byte) (Chunk, error),
) <-chan Chunk {
	out, _ := util.SafeGo(ctx, func(ctx context.Context, vs chan<- Chunk, errs chan<- error) {
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
			if v.Err != nil {
				ch <- ErrorChunk(v.Err)
				continue
			}
			ch <- v.Val
		}
	}()
	return ch
}
