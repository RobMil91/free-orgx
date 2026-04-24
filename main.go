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

	http.HandleFunc("/project", handlers.ProjectHandler(logger, adapters.UserRep, adapters.ProjectRep))
	http.HandleFunc("/login", handlers.LoginHandler(logger))
	http.HandleFunc("/submit", handlers.LoginSubmit(logger, adapters.UserRep))
	http.HandleFunc("/logout", handlers.LogoutHandler(logger, adapters.UserRep))
	http.HandleFunc("/project/create", handlers.CreateProjectHandler(logger, adapters.UserRep, adapters.ProjectRep))
	http.HandleFunc("/project/delete/{id}", handlers.DeleteProjectHandler(logger, adapters.UserRep, adapters.ProjectRep))
	http.HandleFunc("/users", handlers.AdminUsersHandler(logger, adapters.UserRep))
	http.HandleFunc("/users/create", handlers.AdminCreateUserHandler(logger, adapters.UserRep))

	http.HandleFunc("/projects/{id}/tasks", handlers.ProjectTasksHandler(logger, adapters.UserRep, adapters.ProjectRep))
	http.HandleFunc("/projects/{id}/ws", handlers.TasksTopicHandler(logger, adapters.UserRep, nil))

	http.Handle("/", http.FileServer(http.Dir("./static")))

	portStr := fmt.Sprintf(":%s", cfg.Port)
	logger.Info("started free orgx on port" + portStr)
	log.Fatal(http.ListenAndServe(portStr, nil))
}
