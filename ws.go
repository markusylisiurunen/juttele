package juttele

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/markusylisiurunen/juttele/internal/util/jsonrpc"
)

type webSocketProxy struct {
	conn *websocket.Conn

	reqID atomic.Uint64

	pending   map[uint64]chan<- Result[json.RawMessage]
	pendingMu sync.RWMutex

	closeOnce sync.Once
	closeChan chan struct{}
}

func newWebSocketProxy(conn *websocket.Conn) *webSocketProxy {
	ws := &webSocketProxy{
		conn:      conn,
		pending:   make(map[uint64]chan<- Result[json.RawMessage]),
		closeChan: make(chan struct{}),
	}
	return ws
}

func (ws *webSocketProxy) readLoop() {
	for {
		select {
		case <-ws.closeChan:
			return
		default:
			var res jsonrpc.Response
			if err := ws.conn.ReadJSON(&res); err != nil {
				continue
			}
			ws.pendingMu.RLock()
			ch, ok := ws.pending[res.ID]
			ws.pendingMu.RUnlock()
			if !ok {
				continue
			}
			if res.ErrorCode != nil && res.ErrorMessage != nil {
				ch <- Err[json.RawMessage](fmt.Errorf("%d: %s", *res.ErrorCode, *res.ErrorMessage))
				continue
			}
			v, ok := (*res.Result).(json.RawMessage)
			if !ok {
				ch <- Err[json.RawMessage](fmt.Errorf("invalid result type: %T", *res.Result))
				continue
			}
			ch <- Ok(v)
		}
	}
}

func (ws *webSocketProxy) write(v any) error {
	return ws.conn.WriteJSON(v)
}

func (ws *webSocketProxy) rpc(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
	id := ws.reqID.Add(1)
	ch := make(chan Result[json.RawMessage], 1)
	ws.pendingMu.Lock()
	ws.pending[id] = ch
	ws.pendingMu.Unlock()
	defer func() {
		ws.pendingMu.Lock()
		delete(ws.pending, id)
		ws.pendingMu.Unlock()
		close(ch)
	}()
	req := jsonrpc.NewRequest(id, method, params)
	if err := ws.conn.WriteJSON(req); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		return res.Val, nil
	}
}

func (ws *webSocketProxy) close() {
	ws.closeOnce.Do(func() {
		close(ws.closeChan)
		ws.conn.Close()
		ws.pendingMu.Lock()
		for _, ch := range ws.pending {
			ch <- Err[json.RawMessage](context.Canceled)
			close(ch)
		}
		ws.pending = nil
		ws.pendingMu.Unlock()
	})
}
