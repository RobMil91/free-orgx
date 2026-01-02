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
	"github.com/RobMil91/free-orgx/internal/adapters/mocks"
	"github.com/RobMil91/free-orgx/internal/ports"
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

func projectHandler(l *slog.Logger, db ports.UserRepo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		l.Debug("reached project site")
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

			w.Write([]byte(err.Error()))
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieID,
			Value:    token.Value,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
		})

		w.Write([]byte(token.Value))
	}
}

func main() {
	cfg := config.Config{}

	port := flag.String("port", "8080", "Port to listen on")
	flag.Parse()
	if port != nil {
		cfg.Port = *port
	}

	setup.
		level := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))

	ramDB := mocks.RAM{
		Users: map[string]ports.User{
			"tom": {
				Name:     "tom",
				Password: "test",
			},
		},
	}

	http.HandleFunc("/project", projectHandler(logger, &ramDB))
	http.HandleFunc("/login", loginHandler(logger))
	http.HandleFunc("/submit", loginSubmit(logger, &ramDB))
	http.Handle("/", http.FileServer(http.Dir("./static")))

	portStr := fmt.Sprintf(":%s", cfg.Port)
	logger.Info("started free orgx on port" + portStr)
	log.Fatal(http.ListenAndServe(portStr, nil))
}
