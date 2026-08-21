package handlers

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/RobMil91/free-orgx/internal/core/event"
	"github.com/RobMil91/free-orgx/internal/models"
	"github.com/RobMil91/free-orgx/internal/ports"
)

const (
	SessionCookieID = "session_id"
)

type Project struct {
	Files  fs.FS
	Logger *slog.Logger

	UserRepo    ports.UserRepo
	ProjectRepo ports.ProjectRepo
	TasksRepo   ports.TasksRepo

	EventsRepo ports.EventStore

	EventTranslator ports.EventTranslator

	TasksSubjects map[string]event.TaskSubject //project_id -> subject

	EndpointMapping map[string]func(w http.ResponseWriter, r *http.Request)
}

func NewProjectHandler(files fs.FS,
	l *slog.Logger,
	u ports.UserRepo,
	p ports.ProjectRepo,
	t ports.TasksRepo,
	e ports.EventStore,
	ev ports.EventTranslator,
	endpoints map[string]func(w http.ResponseWriter, r *http.Request)) (*Project, error) {

	return &Project{
		Files:       files,
		Logger:      l,
		UserRepo:    u,
		ProjectRepo: p,
		TasksRepo:   t,

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

	projects, err := p.ProjectRepo.GetAllProjects(r.Context())
	if err != nil {
		p.Logger.Error("failed to get projects", "error", err)
		return
	}

	p.Logger.DebugContext(r.Context(), "show project")

	tmpl, err := template.ParseFS(p.Files, "project.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
	users, err := p.UserRepo.GetAll(r.Context())
	if err != nil {
		p.Logger.ErrorContext(r.Context(), fmt.Sprintf("could not get users %s", err.Error()))
		http.Error(w, fmt.Sprintf("could not get users %s", err.Error()), http.StatusInternalServerError)
		return
	}

	var userHTML []struct {
		ShortName string
		Name      string
	}

	for _, u := range users {
		userHTML = append(userHTML, struct {
			ShortName string
			Name      string
		}{
			Name:      u.Name,
			ShortName: u.Name[0:1],
		})
	}

	p.Logger.DebugContext(r.Context(), fmt.Sprintf("loaded tasks %+v", tasks))

	data := map[string]any{
		"ID":    id,
		"Name":  project.Name,
		"Users": userHTML,
	}
	if len(tasks) == 0 {
		data["Tasks"] = nil
	}

	tmpl, err := template.ParseFS(p.Files, "taskboard.html", "create_card.html")
	if err != nil {
		p.Logger.ErrorContext(r.Context(), err.Error())
		http.Error(w, "could locad: templates", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		p.Logger.ErrorContext(r.Context(), err.Error())
		http.Error(w, "could not append ID to template id: "+id, http.StatusInternalServerError)
		return
	}

}

type UsersHTML struct {
	Name      string
	ShortName string
}

func (p *Project) UsersHTML(ctx context.Context) ([]UsersHTML, error) {
	users, err := p.UserRepo.GetAll(ctx)
	if err != nil {
		p.Logger.ErrorContext(ctx, fmt.Sprintf("could not get users %s", err.Error()))
		return nil, err
	}

	var userHTML []UsersHTML

	for _, u := range users {
		userHTML = append(userHTML, UsersHTML{
			Name:      u.Name,
			ShortName: u.Name[0:1],
		})
	}

	return userHTML, nil
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

	tmpl, err := template.ParseFS(p.Files, "create_task.html")
	if err != nil {
		p.Logger.ErrorContext(r.Context(), err.Error())
		http.Error(w, "could not load templates", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		p.Logger.ErrorContext(r.Context(), err.Error())
		http.Error(w, "could not append ID to template id: "+id, http.StatusInternalServerError)
		return
	}
}

func LoginSubmit(
	l *slog.Logger,
	db ports.UserRepo,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

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

	tmpl, err := template.ParseFS(p.Files, "proj.html")
	if err != nil {
		p.Logger.ErrorContext(r.Context(), err.Error())

		return
	}

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
