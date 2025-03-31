package juttele

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/markusylisiurunen/juttele/internal/middleware"
	"github.com/markusylisiurunen/juttele/internal/repo"
	_ "github.com/mattn/go-sqlite3"
)

type App struct {
	// config options
	configToken      string
	configDataFolder string

	// runtime state
	db     *sql.DB
	repo   *repo.Repository
	router *http.ServeMux
	models []Model
	tools  []Tool
}

type appOption func(*App)

func WithDataFolder(folder string) appOption {
	return func(app *App) {
		app.configDataFolder = folder
	}
}

func WithModel(model Model) appOption {
	return func(app *App) {
		app.models = append(app.models, model)
	}
}

func WithToolBundle(tools ToolBundle) appOption {
	return func(app *App) {
		app.tools = append(app.tools, tools.Tools()...)
	}
}

func New(token string, opts ...appOption) *App {
	app := new(App)
	app.configDataFolder = "./.data"
	app.configToken = token
	app.router = http.NewServeMux()
	app.models = make([]Model, 0)
	app.tools = make([]Tool, 0)
	for _, opt := range opts {
		opt(app)
	}
	return app
}

func (app *App) ListenAndServe(ctx context.Context) error {
	type initFunc = func(ctx context.Context) error
	initFuncs := []initFunc{
		app.initModels,
		app.initDatabase,
		app.initRoutes,
	}
	for _, initFunc := range initFuncs {
		if err := initFunc(ctx); err != nil {
			return err
		}
	}
	server := &http.Server{
		Addr:    "0.0.0.0:8765",
		Handler: middleware.Cors()(app.router),
	}
	go func() {
		<-ctx.Done()
		server.Shutdown(ctx)
		app.db.Close()
	}()
	err := server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// ---

func (app *App) initModels(ctx context.Context) error {
	for _, model := range app.models {
		info := model.GetModelInfo()
		if len(info.Personalities) == 0 {
			return fmt.Errorf("model %q has no personalities", info.ID)
		}
	}
	return nil
}

func (app *App) initDatabase(ctx context.Context) error {
	client, err := sql.Open("sqlite3",
		fmt.Sprintf("file:%s/juttele.db?_fk=1", app.configDataFolder))
	if err != nil {
		return err
	}
	app.db = client
	if err := repo.Migrate(ctx, client); err != nil {
		return err
	}
	app.repo = repo.New(client)
	return nil
}

func (app *App) initRoutes(ctx context.Context) error {
	type mountable struct {
		pattern string
		handler http.HandlerFunc
	}
	mountables := []mountable{
		{"GET /api/models", app.apiModelsRouteHandler},
		{"POST /api/generate", app.apiGenerateRouteHandler},

		{"GET /config", app.configRouteHandler},
		{"GET /data", app.dataRouteHandler},
		{"POST /rpc", app.rpcRouteHandler},
		{"GET /chats/{id}", app.sendRouteHandler},
	}
	for _, i := range mountables {
		app.router.Handle(i.pattern,
			middleware.Log()(
				middleware.Auth(app.repo, app.configToken)(
					i.handler,
				),
			),
		)
	}
	return nil
}
