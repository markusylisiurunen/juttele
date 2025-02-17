package juttele

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type rpcRequest struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

func (app *App) rpcRouteHandler(w http.ResponseWriter, r *http.Request) {
	var v rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		http.Error(w, fmt.Sprintf("error decoding request: %v", err), http.StatusBadRequest)
		return
	}
	if v.Name == "" {
		http.Error(w, "name must be provided", http.StatusBadRequest)
		return
	}
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
