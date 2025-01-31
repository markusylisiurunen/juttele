package juttele

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (app *App) handleModelsRoute(w http.ResponseWriter, r *http.Request) {
	type respPersonality struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	type respModel struct {
		ID            string            `json:"id"`
		Name          string            `json:"name"`
		Personalities []respPersonality `json:"personalities"`
	}
	type resp struct {
		Models []respModel `json:"models"`
	}
	v := resp{make([]respModel, 0)}
	for _, model := range app.models {
		info := model.GetModelInfo()
		personalities := make([]respPersonality, 0)
		for _, personality := range info.Personalities {
			personalities = append(personalities, respPersonality{
				ID:   personality.ID,
				Name: personality.Name,
			})
		}
		v.Models = append(v.Models, respModel{
			ID:            info.ID,
			Name:          info.Name,
			Personalities: personalities,
		})
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
	}
}
