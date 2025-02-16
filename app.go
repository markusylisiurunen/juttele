package juttele

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/markusylisiurunen/juttele/internal/db"
	_ "github.com/mattn/go-sqlite3"
)

type App struct {
	dbfolder string
	db       *db.DB
	token    string
	models   []Model
	mux      *http.ServeMux
}

type option func(*App)

func WithDatabaseFolder(folder string) option {
	return func(app *App) {
		app.dbfolder = folder
	}
}

func WithModel(model Model) option {
	return func(app *App) {
		app.models = append(app.models, model)
	}
}

func New(token string, opts ...option) *App {
	app := new(App)
	app.dbfolder = "./.data"
	app.token = token
	app.models = make([]Model, 0)
	app.mux = http.NewServeMux()
	for _, opt := range opts {
		opt(app)
	}
	return app
}

func (app *App) ListenAndServe(ctx context.Context) error {
	// init database
	if err := app.initDatabase(); err != nil {
		return err
	}
	// run migrations
	if err := db.Migrate(ctx, app.db.DB); err != nil {
		return err
	}
	// validate models
	for _, model := range app.models {
		info := model.GetModelInfo()
		if len(info.Personalities) == 0 {
			return fmt.Errorf("model %q has no personalities", info.ID)
		}
	}
	// mount routes
	if err := app.mountRoutes(); err != nil {
		return err
	}
	// start server
	server := &http.Server{
		Addr:    "0.0.0.0:8765",
		Handler: app.corsMiddleware(app.mux),
	}
	go func() {
		<-ctx.Done()
		server.Shutdown(ctx)
	}()
	return server.ListenAndServe()
}

func (app *App) initDatabase() error {
	client, err := sql.Open("sqlite3", fmt.Sprintf("%s/juttele.db", app.dbfolder))
	if err != nil {
		return err
	}
	app.db = db.New(client)
	return nil
}

func (app *App) mountRoutes() error {
	type mountable struct {
		pattern string
		handler http.HandlerFunc
	}
	mountables := []mountable{
		{"GET /models", app.handleModelsRoute},
		{"GET /chats/{id}", app.handleChatRoute},
		{"POST /stream", app.handleStreamRoute},
	}
	for _, m := range mountables {
		app.mux.Handle(m.pattern,
			app.authMiddleware(
				m.handler,
			),
		)
	}
	return nil
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

func (app *App) authMiddleware(next http.Handler) http.Handler {
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
