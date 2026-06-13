package handlers

import (
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"github.com/RobMil91/free-orgx/internal/core/event"
	"github.com/RobMil91/free-orgx/internal/models"
	"github.com/RobMil91/free-orgx/internal/ports"
)

const (
	SessionCookieID = "session_id"
	htmxPath        = "static/"
)

type Project struct {
	TemplatePath string
	Logger       *slog.Logger

	UserRepo    ports.UserRepo
	ProjectRepo ports.ProjectRepo
	TasksRepo   ports.TasksRepo

	EventsRepo ports.EventStore

	EventTranslator ports.EventTranslator

	// InChannel  chan (ports.TaskEventRequest)
	// OutChannel chan (ports.Task)

	TasksSubjects map[string]event.TaskSubject //project_id -> subject
	// TasksObservers map[string]event.TaskObserver //project_id -> observer?

	EndpointMapping map[string]func(w http.ResponseWriter, r *http.Request)
}

func NewProjectHandler(path string,
	l *slog.Logger,
	u ports.UserRepo,
	p ports.ProjectRepo,
	t ports.TasksRepo,
	e ports.EventStore,
	ev ports.EventTranslator,
	endpoints map[string]func(w http.ResponseWriter, r *http.Request)) (*Project, error) {

	return &Project{
		TemplatePath: path,
		Logger:       l,
		UserRepo:     u,
		ProjectRepo:  p,
		TasksRepo:    t,

		EventTranslator: ev,
		EventsRepo:      e,

		TasksSubjects: map[string]event.TaskSubject{},

		EndpointMapping: endpoints,
	}, nil
}

func (p *Project) authUser(r *http.Request) (*ports.User, error) {
	c, err := r.Cookie(SessionCookieID)
	if err != nil {
		return nil, errors.New("Please Login first")
	}

	user, err := p.UserRepo.IsValid(r.Context(), c.Value)
	if err != nil {
		return nil, errors.New("Session invalid, please login again")
	}

	return user, nil
}

func (p *Project) HandleGetProjects(w http.ResponseWriter, r *http.Request) {
	user, err := p.authUser(r)
	if err != nil {
		http.Error(w, err.Error(), 401)
		return
	}

	projects, err := p.ProjectRepo.GetProjectsByOwner(r.Context(), user.Name)
	if err != nil {
		p.Logger.Error("failed to get projects", "error", err)
		return
	}

	p.Logger.DebugContext(r.Context(), "show project")

	template := template.Must(template.ParseFiles(htmxPath + "project.html"))
	template.Execute(w, struct {
		Username string
		Projects []models.Project
		IsAdmin  bool
	}{
		Username: user.Name,
		Projects: projects,
		IsAdmin:  user.Role == ports.RoleAdmin,
	})
}

func (p *Project) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL == nil {
		http.Error(w, "no url", 400)
		return
	}

	endpoint := r.URL.Path

	id := r.PathValue("id")
	if id != "" {
		endpoint = strings.Replace(endpoint, id, "{id}", 1)
	}

	p.Logger.DebugContext(r.Context(), endpoint)

	//todo: this string compare is bullshit. i kind of need sub routes.
	f, ok := p.EndpointMapping[endpoint]
	if !ok {
		p.Logger.DebugContext(r.Context(), fmt.Sprintf("did not get url %s", r.URL.String()))
		http.Error(w, "no such endpoint", 404)
		return
	}

	f(w, r)
}

func LoginHandler(l *slog.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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

func (p *Project) ProjectTasksHandler(w http.ResponseWriter, r *http.Request) {
	_, err := p.authUser(r)
	if err != nil {
		p.Logger.ErrorContext(r.Context(), err.Error())
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	p.Logger.DebugContext(r.Context(), "hello project: "+id)

	project, err := p.ProjectRepo.GetProject(r.Context(), id)
	if err != nil {
		p.Logger.ErrorContext(r.Context(), "could not load project for id: "+id)
		http.Error(w, "could not load project for id: "+id, http.StatusInternalServerError)
		return
	}

	tasks, err := p.TasksRepo.LoadSnapshot(r.Context(), id)
	if err != nil {
		p.Logger.ErrorContext(r.Context(), "could not retrieve snapshot for id: "+id)
		http.Error(w, "could not retrieve snapshot for id: "+id, http.StatusInternalServerError)
		return
	}

	p.Logger.DebugContext(r.Context(), fmt.Sprintf("loaded tasks %+v", tasks))

	data := map[string]any{
		"ID":   id,
		"Name": project.Name,
	}
	if len(tasks) == 0 {
		data["Tasks"] = nil
	}

	tmpl := template.Must(template.ParseFiles(p.TemplatePath + "taskboard.html"))

	err = tmpl.Execute(w, data)
	if err != nil {
		p.Logger.ErrorContext(r.Context(), err.Error())
		http.Error(w, "could not append ID to template id: "+id, http.StatusInternalServerError)
		return
	}

}

func (p *Project) CreateTaskForm(w http.ResponseWriter, r *http.Request) {
	_, err := p.authUser(r)
	if err != nil {
		p.Logger.ErrorContext(r.Context(), err.Error())
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	id := r.PathValue("id")

	p.Logger.DebugContext(r.Context(), "hello create task on project: "+id)

	data := map[string]any{
		"ID": id,
	}

	tmpl := template.Must(template.ParseFiles(p.TemplatePath + "create_task.html"))

	err = tmpl.Execute(w, data)
	if err != nil {
		p.Logger.ErrorContext(r.Context(), err.Error())
		http.Error(w, "could not append ID to template id: "+id, http.StatusInternalServerError)
		return
	}
}

type TaskBoardValues struct {
	ID    string
	Tasks []models.Task
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
			// w.Write([]byte(fmt.Sprintf("Welcome! You are the first admin (%s).", result.User.Name)))
			return
		}

		w.Write([]byte(result.Cookie.Value))
	}
}

func (p *Project) CreateProjectHandler(w http.ResponseWriter, r *http.Request) {
	p.Logger.DebugContext(r.Context(), "reached create project handler")

	user, err := p.authUser(r)
	if err != nil {
		http.Error(w, err.Error(), 401)
		return
	}

	projectName := r.FormValue("name")
	if projectName == "" {
		w.Write([]byte("Project name is required"))
		return
	}

	project, err := p.ProjectRepo.CreateProject(r.Context(), projectName, user.Name)
	if err != nil {
		p.Logger.Error("failed to create project", "error", err, "user", user.Name)
		w.Write([]byte("Failed to create project"))
		return
	}

	tmpl := template.Must(template.ParseFiles(htmxPath + "proj.html"))
	tmpl.Execute(w, struct {
		ID   string
		Name string
	}{
		ID:   project.ID,
		Name: project.Name,
	})
}

func (p *Project) DeleteProjectHandler(w http.ResponseWriter, r *http.Request) {
	p.Logger.Debug("reached delete  project handler")

	user, err := p.authUser(r)
	if err != nil {
		http.Error(w, err.Error(), 401)
		return
	}

	p.Logger.Debug("command: delete id " + r.PathValue("id"))

	err = p.ProjectRepo.DeleteProject(r.Context(), r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		p.Logger.Error("failed to create project", "error", err, "user", user.Name)
		return
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
