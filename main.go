package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/RobMil91/free-orgx/config"
	"github.com/RobMil91/free-orgx/internal/handlers"
	"github.com/RobMil91/free-orgx/internal/setup"
)

//go:embed static/*
var staticContent embed.FS

func main() {
	staticFS, err := fs.Sub(staticContent, "static")
	if err != nil {
		panic(err)
	}
	cfg := config.Config{}

	port := flag.String("port", "8080", "Port to listen on")
	decsionDB := flag.Bool("ram", false, "Decide wether to use a ram db or std db")

	flag.Parse()
	if port != nil {
		cfg.Port = *port
	}

	if decsionDB != nil {
		cfg.RAMDB = *decsionDB
	}

	adapters, err := setup.Setup(cfg)
	if err != nil {
		panic(err)
	}

	level := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))

	mux := http.NewServeMux()

	projectHandler, err := handlers.NewProjectHandler(
		staticFS,
		logger,
		adapters.UserRep,
		adapters.ProjectRep,
		adapters.TaskRepo,

		adapters.EventRepo,
		adapters.EventsTranslator,
		map[string]func(w http.ResponseWriter, r *http.Request){},
	)

	if err != nil {
		panic(err)
	}

	userHandler := handlers.NewUserHandler(staticFS, logger, adapters.UserRep)

	projectHandler.EndpointMapping = map[string]func(w http.ResponseWriter, r *http.Request){
		"/project":             projectHandler.HandleGetProjects,
		"/project/create":      projectHandler.CreateProjectHandler,
		"/project/delete/{id}": projectHandler.DeleteProjectHandler,

		"/projects/{id}/tasks":        projectHandler.ProjectTasksHandler,
		"/projects/{id}/tasks/create": projectHandler.CreateTaskForm,

		"/projects/{id}/ws": projectHandler.WebsocketHandler,

		"/login":             userHandler.Login,
		"/submit":            userHandler.Submit,
		"/users":             userHandler.AdminUsersPage,
		"/admin/user/create": userHandler.AdminCreateUser,
		"/logout":            userHandler.Logout,
	}

	for k := range projectHandler.EndpointMapping {
		mux.Handle(k, projectHandler)
	}

	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	portStr := fmt.Sprintf(":%s", cfg.Port)
	logger.Info("started free orgx on port" + portStr)
	log.Fatal(http.ListenAndServe(portStr, mux))
}
