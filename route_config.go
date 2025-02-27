package juttele

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type configResponsePersonality struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type configResponseModel struct {
	ID            string                      `json:"id"`
	Name          string                      `json:"name"`
	Personalities []configResponsePersonality `json:"personalities"`
}
type configResponse struct {
	Models []configResponseModel `json:"models"`
}

func (app *App) configRouteHandler(w http.ResponseWriter, r *http.Request) {
	var v configResponse
	v.Models = make([]configResponseModel, 0)
	for _, model := range app.models {
		info := model.GetModelInfo()
		personalities := make([]configResponsePersonality, 0)
		for _, personality := range info.Personalities {
			personalities = append(personalities, configResponsePersonality{
				ID:   personality.ID,
				Name: personality.Name,
			})
		}
		v.Models = append(v.Models, configResponseModel{
			ID:            info.ID,
			Name:          info.Name,
			Personalities: personalities,
		})
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
	}
}
