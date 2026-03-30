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

func projectHandler(l *slog.Logger, db ports.UserRepo) func(w http.ResponseWriter, r *http.Request) {
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

		l.Debug("replying project site")
		tmpl := template.Must(template.ParseFiles("internal/adapters/htmlx/project.html"))
		tmpl.Execute(w, fmt.Sprintf("hello %s", user.Name))
	}
}

func loginSubmit(
	l *slog.Logger,
	db ports.UserRepo,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		l.Debug("login submit request")
		l.Debug(r.FormValue("user"))

		token, err := db.GetUserToken(r.Context(), r.FormValue("user"), r.FormValue("password"))
		if err != nil {
			w.Write([]byte("Login failed: invalid username or password"))
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieID,
			Value:    token.Value,
			HttpOnly: true,
			Path:     "/",
		})

		w.Write([]byte(token.Value))
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

	http.HandleFunc("/project", projectHandler(logger, adapters.UserRep))
	http.HandleFunc("/login", loginHandler(logger))
	http.HandleFunc("/submit", loginSubmit(logger, adapters.UserRep))
	http.HandleFunc("/logout", logoutHandler(logger, adapters.UserRep))
	http.Handle("/", http.FileServer(http.Dir("./static")))

	portStr := fmt.Sprintf(":%s", cfg.Port)
	logger.Info("started free orgx on port" + portStr)
	log.Fatal(http.ListenAndServe(portStr, nil))
}
