package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/RobMil91/free-orgx/config"
	"github.com/RobMil91/free-orgx/internal/handlers"
	"github.com/RobMil91/free-orgx/internal/setup"
)

const (
	htmxPath = "static/"
)

func main() {
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

	handler, err := handlers.NewProjectHandler(htmxPath,
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

	handler.EndpointMapping = map[string]func(w http.ResponseWriter, r *http.Request){
		"/project":             handler.HandleGetProjects,
		"/project/create":      handler.CreateProjectHandler,
		"/project/delete/{id}": handler.DeleteProjectHandler,

		"/projects/{id}/tasks":        handler.ProjectTasksHandler,
		"/projects/{id}/tasks/create": handler.CreateTaskForm,

		"/projects/{id}/ws": handler.WebsocketHandler,
	}

	for k := range handler.EndpointMapping {
		mux.Handle(k, handler)
	}

	mux.HandleFunc("/login", handlers.LoginHandler(logger))
	mux.HandleFunc("/submit", handlers.LoginSubmit(logger, adapters.UserRep))
	mux.HandleFunc("/logout", handlers.LogoutHandler(logger, adapters.UserRep))
	mux.HandleFunc("/users", handlers.AdminUsersHandler(logger, adapters.UserRep))
	mux.HandleFunc("/admin/user/create", handlers.AdminCreateUserHandler(logger, adapters.UserRep))

	mux.Handle("/", http.FileServer(http.Dir("./static")))

	portStr := fmt.Sprintf(":%s", cfg.Port)
	logger.Info("started free orgx on port" + portStr)
	log.Fatal(http.ListenAndServe(portStr, mux))
}
