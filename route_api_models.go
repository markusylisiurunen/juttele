package juttele

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type apiModelsRequest_Model struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type apiModelsResponse struct {
	Models []apiModelsRequest_Model `json:"models"`
}

func (app *App) apiModelsRouteHandler(w http.ResponseWriter, r *http.Request) {
	var response apiModelsResponse
	for _, m := range app.models {
		modelInfo := m.GetModelInfo()
		response.Models = append(response.Models, apiModelsRequest_Model{
			ID:   modelInfo.ID,
			Name: modelInfo.Name,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
		return
	}
}
