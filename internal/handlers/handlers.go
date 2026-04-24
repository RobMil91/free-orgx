package handlers

import (
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/RobMil91/free-orgx/internal/models"
	"github.com/RobMil91/free-orgx/internal/ports"
)

const (
	SessionCookieID = "session_id"
	htmxPath        = "static/"
)

func TasksHandler(l *slog.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		l.Debug("reached tasks handler")
		tmpl := template.Must(template.ParseFiles(htmxPath + "taskboard.html"))
		tmpl.Execute(w, nil)
	}
}

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

		projects, err := projectRepo.GetProjectsByOwner(r.Context(), user.Name)
		if err != nil {
			l.Error("failed to get projects", "error", err)
		}

		tmpl := template.Must(template.ParseFiles(htmxPath + "project.html"))
		tmpl.Execute(w, struct {
			Username string
			Projects []models.Project
			IsAdmin  bool
		}{
			Username: user.Name,
			Projects: projects,
			IsAdmin:  user.Role == ports.RoleAdmin,
		})
	}
}

func auth(r *http.Request, u ports.UserRepo) (*ports.User, error) {
	c, err := r.Cookie(SessionCookieID)
	if err != nil {
		return nil, fmt.Errorf("invalid token in request %w", err)
	}

	user, err := u.IsValid(r.Context(), c.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid session  %w", err)
	}

	return user, nil
}

func ProjectTasksHandler(l *slog.Logger, u ports.UserRepo, p ports.ProjectRepo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := auth(r, u)
		if err != nil {
			l.DebugContext(r.Context(), "user login failed")
			w.Write([]byte("Session Validation failed"))
			return
		}

		//get id of project
		id := r.PathValue("id")

		l.DebugContext(r.Context(), "hello project: "+id)

		tmpl := template.Must(template.ParseFiles(htmxPath + "taskboard.html"))

		tmpl.Execute(w, struct {
			ID string
		}{
			ID: id,
		})

		//open web socket connection?
		//probably except in other handler
	}
}

func TasksTopicHandler(l *slog.Logger, u ports.UserRepo, p ports.ProjectRepo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := auth(r, u)
		if err != nil {
			l.DebugContext(r.Context(), "user login failed")
			w.Write([]byte("Session Validation failed"))
			return
		}

		l.DebugContext(r.Context(), fmt.Sprintf("user %s attempt ws connect", user.Name))

		id := r.PathValue("id")
		l.DebugContext(r.Context(), fmt.Sprintf("attempt to subscribe to task events for project %s", id))

		//TODO: ws connection is dual -> need to pass consumer and producer
		var upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			l.ErrorContext(r.Context(), err.Error())
			return
		}
		l.DebugContext(r.Context(), "successfull websocket connection")

		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				l.ErrorContext(r.Context(), err.Error())
				break
			}

			resp := map[string]any{
				"echo": string(msg),
			}

			conn.WriteJSON(resp)
		}
	}
}

type WebSocketMsg struct {
	Authorization string
	Content       io.Reader
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

type TaskHandler struct {
}

func (t *TaskHandler) HandleCreateTask() {

}

func (t *TaskHandler) HandleUpdateTask() {

}
