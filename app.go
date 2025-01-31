package juttele

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type App struct {
	token  string
	models []Model
	mux    *http.ServeMux
}

type option func(*App)

func WithModel(model Model) option {
	return func(app *App) {
		app.models = append(app.models, model)
	}
}

func New(token string, opts ...option) *App {
	app := new(App)
	app.token = token
	app.models = make([]Model, 0)
	app.mux = http.NewServeMux()
	app.mux.Handle("/models", app.corsMiddleware(app.authMiddleware(app.handleModelsRoute)))
	app.mux.Handle("/stream", app.corsMiddleware(app.authMiddleware(app.handleStreamRoute)))
	for _, opt := range opts {
		opt(app)
	}
	return app
}

func (app *App) ListenAndServe(ctx context.Context) error {
	// validate models
	for _, model := range app.models {
		info := model.GetModelInfo()
		if len(info.Personalities) == 0 {
			return fmt.Errorf("model %q has no personalities", info.ID)
		}
	}
	// start server
	server := &http.Server{
		Addr:    "0.0.0.0:8765",
		Handler: app.mux,
	}
	go func() {
		<-ctx.Done()
		server.Shutdown(ctx)
	}()
	return server.ListenAndServe()
}

func (app *App) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (app *App) authMiddleware(nextFunc http.HandlerFunc) http.Handler {
	next := http.HandlerFunc(nextFunc)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		if token != app.token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
