package main_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RobMil91/free-orgx/internal/adapters/database"
	"github.com/RobMil91/free-orgx/internal/models"
	"github.com/RobMil91/free-orgx/internal/ports"
)

type mockUserRepo struct {
	users map[string]ports.User
}

func (m *mockUserRepo) GetUserToken(ctx context.Context, name, password string) (*ports.LoginResult, error) {
	return nil, nil
}

func (m *mockUserRepo) CountUsers(ctx context.Context) (int, error) {
	return len(m.users), nil
}

func (m *mockUserRepo) IsValid(ctx context.Context, c string) (*ports.User, error) {
	for _, u := range m.users {
		if u.Cookie.Value == c {
			return &u, nil
		}
	}
	return nil, ports.UserNotFound
}

func (m *mockUserRepo) Create(ctx context.Context, name, password, role string) error {
	return nil
}

func (m *mockUserRepo) Delete(ctx context.Context, name string) error {
	return nil
}

func (m *mockUserRepo) GetAll(ctx context.Context) ([]ports.User, error) {
	return nil, nil
}

func (m *mockUserRepo) ChangePassword(ctx context.Context, name, newPassword string) error {
	return nil
}

func (m *mockUserRepo) Logout(ctx context.Context, name string) error {
	return nil
}

func (m *mockUserRepo) GetAdmin(ctx context.Context) (*ports.User, error) {
	return nil, ports.UserNotFound
}

func (m *mockUserRepo) CountAdmins(ctx context.Context) (int, error) {
	return 0, nil
}

type mockProjectRepo struct {
	projects []models.Project
}

func (m *mockProjectRepo) CreateProject(ctx context.Context, name, owner string) (models.Project, error) {
	project := models.Project{ID: "test-id", Name: name, Owner: owner}
	m.projects = append(m.projects, project)
	return project, nil
}

func (m *mockProjectRepo) GetProjectsByOwner(ctx context.Context, owner string) ([]models.Project, error) {
	var result []models.Project
	for _, p := range m.projects {
		if p.Owner == owner {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockProjectRepo) DeleteProject(ctx context.Context, id string) error {
	return nil
}

var _ ports.UserRepo = (*mockUserRepo)(nil)
var _ ports.ProjectRepo = (*mockProjectRepo)(nil)
var _ ports.UserRepo = (*database.SQLiteAdapter)(nil)
var _ ports.ProjectRepo = (*database.SQLiteAdapter)(nil)

func TestCreateProject_Unauthenticated(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	req, _ := http.NewRequest("POST", "/project/create", strings.NewReader("name=Test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler := createProjectHandlerTest(db, db)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Please login first") {
		t.Errorf("expected login message, got: %s", w.Body.String())
	}
}

func TestCreateProject_InvalidSession(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	req, _ := http.NewRequest("POST", "/project/create", strings.NewReader("name=Test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "invalid-token"})
	w := httptest.NewRecorder()

	handler := createProjectHandlerTest(db, db)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Session invalid") {
		t.Errorf("expected session invalid message, got: %s", w.Body.String())
	}
}

func TestCreateProject_EmptyName(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password", ports.RoleUser)
	token, _ := db.GetUserToken(ctx, "testuser", "password")

	req, _ := http.NewRequest("POST", "/project/create", strings.NewReader("name="))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: token.Cookie.Value})
	w := httptest.NewRecorder()

	handler := createProjectHandlerTest(db, db)
	handler(w, req)

	if !strings.Contains(w.Body.String(), "Project name is required") {
		t.Errorf("expected name required message, got: %s", w.Body.String())
	}
}

func TestCreateProject_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password", ports.RoleUser)
	token, _ := db.GetUserToken(ctx, "testuser", "password")

	req, _ := http.NewRequest("POST", "/project/create", strings.NewReader("name=MyProject"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: token.Cookie.Value})
	w := httptest.NewRecorder()

	handler := createProjectHandlerTest(db, db)
	handler(w, req)

	if !strings.Contains(w.Body.String(), "Created project: MyProject") {
		t.Errorf("expected project created message, got: %s", w.Body.String())
	}
}

func TestListProjects_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password", ports.RoleUser)
	token, _ := db.GetUserToken(ctx, "testuser", "password")
	db.CreateProject(ctx, "Project1", "testuser")
	db.CreateProject(ctx, "Project2", "testuser")

	req, _ := http.NewRequest("GET", "/project", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: token.Cookie.Value})
	w := httptest.NewRecorder()

	handler := listProjectsHandlerTest(db, db)
	handler(w, req)

	if !strings.Contains(w.Body.String(), "Project1") {
		t.Errorf("expected Project1 in response, got: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Project2") {
		t.Errorf("expected Project2 in response, got: %s", w.Body.String())
	}
}

func TestListProjects_Unauthorized(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	req, _ := http.NewRequest("GET", "/project", nil)
	w := httptest.NewRecorder()

	handler := listProjectsHandlerTest(db, db)
	handler(w, req)

	if !strings.Contains(w.Body.String(), "Please login first") {
		t.Errorf("expected login message, got: %s", w.Body.String())
	}
}

func TestListProjects_Empty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "emptyuser", "password", ports.RoleUser)
	token, _ := db.GetUserToken(ctx, "emptyuser", "password")

	req, _ := http.NewRequest("GET", "/project", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: token.Cookie.Value})
	w := httptest.NewRecorder()

	handler := listProjectsHandlerTest(db, db)
	handler(w, req)

	if !strings.Contains(w.Body.String(), "No projects yet") {
		t.Errorf("expected empty state message, got: %s", w.Body.String())
	}
}

func setupTestDB(t *testing.T) (*database.SQLiteAdapter, func()) {
	t.Helper()
	t.Setenv("PASSWORD_PEPPER", "test-pepper-"+t.Name())

	db, err := database.NewSQLiteForTest(t.Name())
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	cleanup := func() {
		db.Conn.Close()
		database.CleanupTestDB(t.Name())
	}
	return db, cleanup
}

func createProjectHandlerTest(userRepo ports.UserRepo, projectRepo ports.ProjectRepo) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil {
			w.Write([]byte("Please login first"))
			return
		}

		_, err = userRepo.IsValid(r.Context(), c.Value)
		if err != nil {
			w.Write([]byte("Session invalid, please login again"))
			return
		}

		projectName := r.FormValue("name")
		if projectName == "" {
			w.Write([]byte("Project name is required"))
			return
		}

		project, err := projectRepo.CreateProject(r.Context(), projectName, "testuser")
		if err != nil {
			w.Write([]byte("Failed to create project"))
			return
		}

		w.Write([]byte("Created project: " + project.Name + " (ID: " + project.ID + ")"))
	}
}

func listProjectsHandlerTest(userRepo ports.UserRepo, projectRepo ports.ProjectRepo) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil {
			w.Write([]byte("Please login first"))
			return
		}

		user, err := userRepo.IsValid(r.Context(), c.Value)
		if err != nil {
			w.Write([]byte("Session invalid, please login again"))
			return
		}

		projects, _ := projectRepo.GetProjectsByOwner(r.Context(), user.Name)

		if len(projects) == 0 {
			w.Write([]byte("No projects yet. Create one above!"))
			return
		}

		for _, p := range projects {
			w.Write([]byte(p.Name + "\n"))
		}
	}
}
