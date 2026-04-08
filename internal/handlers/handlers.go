package handlers

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/RobMil91/free-orgx/internal/models"
	"github.com/RobMil91/free-orgx/internal/ports"
)

const (
	SessionCookieID = "session_id"
	htmxPath        = "static/"
)

func LoginHandler(l *slog.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		l.Debug("reached login handler")
		tmpl := template.Must(template.ParseFiles(htmxPath + "login.html"))
		tmpl.Execute(w, nil)
	}
}

func LogoutHandler(l *slog.Logger, db ports.UserRepo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		l.Debug("reached logout handler")

		c, err := r.Cookie(SessionCookieID)
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
			Name:   SessionCookieID,
			Value:  "",
			MaxAge: -1,
		})

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func ProjectHandler(l *slog.Logger, db ports.UserRepo, projectRepo ports.ProjectRepo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(SessionCookieID)
		if err != nil {
			w.Write([]byte("Please login first"))
			return
		}

		user, err := db.IsValid(r.Context(), c.Value)
		if err != nil {
			w.Write([]byte("Session invalid, please login again"))
			return
		}

		projectResp := []models.Project{
			{
				ID:      "test",
				Name:    "holder",
				Owner:   "noone",
				Created: time.Now().String(),
			},
		}

		projects, err := projectRepo.GetProjectsByOwner(r.Context(), user.Name)
		if err != nil {
			l.Error("failed to get projects", "error", err)
		}

		if len(projects) > 0 {
			l.Debug("found projects")
			projectResp = projects
		}

		l.Debug("replying project site with %v", projectResp)
		tmpl := template.Must(template.ParseFiles(htmxPath + "project.html"))
		tmpl.Execute(w, struct {
			Username string
			Projects []models.Project
			IsAdmin  bool
		}{
			Username: user.Name,
			Projects: projectResp,
			IsAdmin:  user.Role == ports.RoleAdmin,
		})
	}
}

func LoginSubmit(
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
			Name:     SessionCookieID,
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

func CreateProjectHandler(l *slog.Logger, userRepo ports.UserRepo, projectRepo ports.ProjectRepo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		l.Debug("reached create project handler")

		c, err := r.Cookie(SessionCookieID)
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

		//TODO: send back a piece of html that resembles the project
		tmpl := template.Must(template.ParseFiles(htmxPath + "proj.html"))
		tmpl.Execute(w, struct {
			ID   string
			Name string
		}{
			ID:   project.ID,
			Name: project.Name,
		})

	}
}

func DeleteProjectHandler(l *slog.Logger, userRepo ports.UserRepo, projectRepo ports.ProjectRepo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		l.Debug("reached delete  project handler")

		c, err := r.Cookie(SessionCookieID)
		if err != nil {
			w.Write([]byte("Please login first"))
			return
		}

		user, err := userRepo.IsValid(r.Context(), c.Value)
		if err != nil {
			w.Write([]byte("Session invalid, please login again"))
			return
		}

		l.Debug("delete id " + r.PathValue("id"))

		err = projectRepo.DeleteProject(r.Context(), r.PathValue("id"))
		if err != nil {
			l.Error("failed to create project", "error", err, "user", user.Name)
			w.Write([]byte("Failed to create project"))
			return
		}
	}
}

func AdminUsersHandler(l *slog.Logger, db ports.UserRepo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		l.Debug("reached admin users handler")

		c, err := r.Cookie(SessionCookieID)
		if err != nil {
			w.Write([]byte("Please login first"))
			return
		}

		user, err := db.IsValid(r.Context(), c.Value)
		if err != nil {
			w.Write([]byte("Session invalid, please login again"))
			return
		}

		if user.Role != ports.RoleAdmin {
			w.Write([]byte("Access denied: admin only"))
			return
		}

		users, err := db.GetAll(r.Context())
		if err != nil {
			l.Error("failed to get users", "error", err)
			users = []ports.User{}
		}

		tmpl := template.Must(template.ParseFiles(htmxPath + "admin_users.html"))
		tmpl.Execute(w, struct {
			Users []ports.User
		}{
			Users: users,
		})
	}
}

func AdminCreateUserHandler(l *slog.Logger, db ports.UserRepo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		l.Debug("reached admin create user handler")

		c, err := r.Cookie(SessionCookieID)
		if err != nil {
			w.Write([]byte("Please login first"))
			return
		}

		user, err := db.IsValid(r.Context(), c.Value)
		if err != nil {
			w.Write([]byte("Session invalid, please login again"))
			return
		}

		if user.Role != ports.RoleAdmin {
			w.Write([]byte("Access denied: admin only"))
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		if username == "" || password == "" {
			w.Write([]byte("Username and password are required"))
			return
		}

		err = db.Create(r.Context(), username, password, ports.RoleUser)
		if err != nil {
			l.Error("failed to create user", "error", err)
			w.Write([]byte(fmt.Sprintf("Failed to create user: %s", err.Error())))
			return
		}

		l.Info(fmt.Sprintf("admin %s created user %s", user.Name, username))
		w.Write([]byte(fmt.Sprintf("User '%s' created successfully", username)))
	}
}
