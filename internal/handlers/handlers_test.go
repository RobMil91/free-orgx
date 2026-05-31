package handlers_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RobMil91/free-orgx/internal/adapters/mocks"
	"github.com/RobMil91/free-orgx/internal/handlers"
	"github.com/RobMil91/free-orgx/internal/models"
	"github.com/RobMil91/free-orgx/internal/ports"
)

type mockTasksRepo struct {
	tasks []models.Task
	err   error
}

func (m *mockTasksRepo) LoadSnapshot(_ context.Context, _ string) ([]models.Task, error) {
	return m.tasks, m.err
}

func (m *mockTasksRepo) CreateSnapshot(_ context.Context, _ string, _ []models.Task) error {
	return nil
}

func TestProject_ProjectTasksHandler(t *testing.T) {
	ram := &mocks.RAM{
		Projects: make(map[string][]models.Project),
	}
	if err := ram.Create(context.Background(), "testuser", "pass", ports.RoleUser); err != nil {
		t.Fatal(err)
	}
	loginResult, err := ram.GetUserToken(context.Background(), "testuser", "pass")
	if err != nil {
		t.Fatal(err)
	}
	validCookie := loginResult.Cookie.Value

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "taskboard.html"), []byte("taskboard content"), 0644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name        string
		cookieVal   string
		projectID   string
		tasksRepo   *mockTasksRepo
		projectRepo []models.Project
		wantStatus  int
		wantBody    string
	}{
		{
			name:      "no cookie returns forbidden",
			cookieVal: "",
			projectID: "proj-1",
			tasksRepo: &mockTasksRepo{},
			projectRepo: []models.Project{
				{
					ID:   "proj-1",
					Name: "testProjectName",
				},
			},
			wantStatus: http.StatusForbidden,
			wantBody:   "Please Login first\n",
		},
		{
			name:      "invalid cookie returns forbidden",
			cookieVal: "invalid-cookie",
			projectID: "proj-1",
			tasksRepo: &mockTasksRepo{},
			projectRepo: []models.Project{
				{
					ID:   "proj-1",
					Name: "testProjectName",
				},
			},
			wantStatus: http.StatusForbidden,
			wantBody:   "Session invalid, please login again\n",
		},
		{
			name:      "load snapshot error returns 500",
			cookieVal: validCookie,
			projectID: "proj-1",
			tasksRepo: &mockTasksRepo{err: errors.New("db error")},
			projectRepo: []models.Project{
				{
					ID:   "proj-1",
					Name: "testProjectName",
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   "could not retrieve snapshot for id: proj-1\n",
		},
		{
			name:      "empty tasks executes template with nil",
			cookieVal: validCookie,
			projectID: "proj-1",
			tasksRepo: &mockTasksRepo{tasks: []models.Task{}},
			projectRepo: []models.Project{
				{
					ID:   "proj-1",
					Name: "testProjectName",
				},
			},
			wantStatus: http.StatusOK,
			wantBody:   "taskboard content",
		},
		{
			name:      "non-empty tasks executes template",
			cookieVal: validCookie,
			projectID: "proj-2",
			tasksRepo: &mockTasksRepo{
				tasks: []models.Task{
					{NewTask: models.NewTask{Title: "Task 1", Description: "Desc 1"}, TaskID: "proj-2"},
				},
			},
			projectRepo: []models.Project{
				{
					ID:   "proj-2",
					Name: "testProjectName",
				},
			},
			wantStatus: http.StatusOK,
			wantBody:   "taskboard content",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ram.Projects["testuser"] = append(ram.Projects["testuser"], tt.projectRepo...)
			p, err := handlers.NewProjectHandler(
				tmpDir+"/",
				logger,
				ram,
				ram,
				tt.tasksRepo,
				nil,
				nil,
			)

			if err != nil {
				t.Log(err.Error())
				t.FailNow()
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.cookieVal != "" {
				r.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: tt.cookieVal})
			}
			r.SetPathValue("id", tt.projectID)

			p.ProjectTasksHandler(w, r)

			resp := w.Result()
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("got status %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			body := w.Body.String()
			if !strings.Contains(body, tt.wantBody) {
				t.Errorf("got body %q, want to contain %q", body, tt.wantBody)
			}
		})
	}
}
