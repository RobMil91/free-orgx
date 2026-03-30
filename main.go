package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/RobMil91/free-orgx/config"
	"github.com/RobMil91/free-orgx/internal/models"
	"github.com/RobMil91/free-orgx/internal/ports"
	"github.com/RobMil91/free-orgx/internal/setup"
)

const (
	sessionCookieID = "session_id"
)

func loginHandler(l *slog.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		l.Debug("reached login handler")
		tmpl := template.Must(template.ParseFiles("internal/adapters/htmlx/login.html"))
		tmpl.Execute(w, nil)
	}
}

func logoutHandler(l *slog.Logger, db ports.UserRepo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		l.Debug("reached logout handler")

		c, err := r.Cookie(sessionCookieID)
		if err != nil {
			http.Error(w, err.Error(), 401)
			return
		}

		user, err := db.IsValid(r.Context(), c.Value)
		if err != nil {
			http.Error(w, err.Error(), 403)
			return
		}

		if err := db.Logout(r.Context(), user.Name); err != nil {
			l.Error("logout failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:   sessionCookieID,
			Value:  "",
			MaxAge: -1,
		})

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func projectHandler(l *slog.Logger, db ports.UserRepo, projectRepo ports.ProjectRepo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		l.Debug("reached project site")
		c, err := r.Cookie(sessionCookieID)
		if err != nil {
			w.Write([]byte("Please login first"))
			return
		}

		user, err := db.IsValid(r.Context(), c.Value)
		if err != nil {
			w.Write([]byte("Session invalid, please login again"))
			return
		}

		projects, err := projectRepo.GetProjectsByOwner(r.Context(), user.Name)
		if err != nil {
			l.Error("failed to get projects", "error", err)
			projects = []models.Project{}
		}

		l.Debug("replying project site")
		tmpl := template.Must(template.ParseFiles("internal/adapters/htmlx/project.html"))
		tmpl.Execute(w, struct {
			Username string
			Projects []models.Project
		}{
			Username: user.Name,
			Projects: projects,
		})
	}
}

func loginSubmit(
	l *slog.Logger,
	db ports.UserRepo,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		l.Debug("login submit request")
		l.Debug(r.FormValue("user"))

		result, err := db.GetUserToken(r.Context(), r.FormValue("user"), r.FormValue("password"))
		if err != nil {
			w.Write([]byte("Login failed: invalid username or password"))
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieID,
			Value:    result.Cookie.Value,
			HttpOnly: true,
			Path:     "/",
		})

		if result.IsNewUser {
			l.Info(fmt.Sprintf("first admin registered: %s", result.User.Name))
			w.Write([]byte(fmt.Sprintf("Welcome! You are the first admin (%s).", result.User.Name)))
			return
		}

		w.Write([]byte(result.Cookie.Value))
	}
}

func createProjectHandler(l *slog.Logger, userRepo ports.UserRepo, projectRepo ports.ProjectRepo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		l.Debug("reached create project handler")

		c, err := r.Cookie(sessionCookieID)
		if err != nil {
			w.Write([]byte("Please login first"))
			return
		}

		user, err := userRepo.IsValid(r.Context(), c.Value)
		if err != nil {
			w.Write([]byte("Session invalid, please login again"))
			return
		}

		projectName := r.FormValue("name")
		if projectName == "" {
			w.Write([]byte("Project name is required"))
			return
		}

		project, err := projectRepo.CreateProject(r.Context(), projectName, user.Name)
		if err != nil {
			l.Error("failed to create project", "error", err, "user", user.Name)
			w.Write([]byte("Failed to create project"))
			return
		}

		w.Write([]byte(fmt.Sprintf("Created project: %s (ID: %s)", project.Name, project.ID)))
	}
}

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

	http.HandleFunc("/project", projectHandler(logger, adapters.UserRep, adapters.ProjectRep))
	http.HandleFunc("/login", loginHandler(logger))
	http.HandleFunc("/submit", loginSubmit(logger, adapters.UserRep))
	http.HandleFunc("/logout", logoutHandler(logger, adapters.UserRep))
	http.HandleFunc("/project/create", createProjectHandler(logger, adapters.UserRep, adapters.ProjectRep))
	http.Handle("/", http.FileServer(http.Dir("./static")))

	portStr := fmt.Sprintf(":%s", cfg.Port)
	logger.Info("started free orgx on port" + portStr)
	log.Fatal(http.ListenAndServe(portStr, nil))
}
