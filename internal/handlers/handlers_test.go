package handlers_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/RobMil91/free-orgx/internal/adapters/database"
	"github.com/RobMil91/free-orgx/internal/handlers"
	"github.com/RobMil91/free-orgx/internal/models"
	"github.com/RobMil91/free-orgx/internal/ports"
)

func setupDBForHandlerTest(t *testing.T) (*database.SQLiteAdapter, func()) {
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

func TestLoginHandler_Real(t *testing.T) {
	if _, err := os.Stat("internal/adapters/htmlx/login.html"); os.IsNotExist(err) {
		t.Skip("skipping test: HTML templates not accessible from package directory")
	}

	_, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.LoginHandler(logger)

	req := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestLogoutHandler_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	ctx := context.Background()
	token, _ := db.GetUserToken(ctx, "firstadmin", "adminpass")

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.LogoutHandler(logger, db)

	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: token.Cookie.Value})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected status 302, got %d", w.Code)
	}

	for _, c := range w.Result().Cookies() {
		if c.Name == handlers.SessionCookieID && c.Value != "" {
			t.Error("session cookie should be cleared")
		}
	}
}

func TestLogoutHandler_NoCookie_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.LogoutHandler(logger, db)

	req := httptest.NewRequest("POST", "/logout", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestLogoutHandler_InvalidSession_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.LogoutHandler(logger, db)

	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: "invalid-token"})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestProjectHandler_Real(t *testing.T) {
	if _, err := os.Stat("internal/adapters/htmlx/project.html"); os.IsNotExist(err) {
		t.Skip("skipping test: HTML templates not accessible from package directory")
	}

	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	ctx := context.Background()
	token, _ := db.GetUserToken(ctx, "firstadmin", "adminpass")
	db.CreateProject(ctx, "TestProject", "firstadmin")

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.ProjectHandler(logger, db, db)

	req := httptest.NewRequest("GET", "/project", nil)
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: token.Cookie.Value})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "TestProject") {
		t.Errorf("expected TestProject in response, got: %s", w.Body.String())
	}
}

func TestProjectHandler_NoCookie_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.ProjectHandler(logger, db, db)

	req := httptest.NewRequest("GET", "/project", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "Please login first") {
		t.Errorf("expected login message, got: %s", w.Body.String())
	}
}

func TestProjectHandler_InvalidSession_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.ProjectHandler(logger, db, db)

	req := httptest.NewRequest("GET", "/project", nil)
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: "invalid-token"})
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "Session invalid") {
		t.Errorf("expected session invalid message, got: %s", w.Body.String())
	}
}

func TestLoginSubmit_FirstAdmin_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.LoginSubmit(logger, db)

	req := httptest.NewRequest("POST", "/submit", strings.NewReader("user=firstadmin&password=adminpass"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "Welcome") {
		t.Errorf("expected welcome message, got: %s", w.Body.String())
	}
}

func TestLoginSubmit_Success_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password", ports.RoleUser)

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.LoginSubmit(logger, db)

	req := httptest.NewRequest("POST", "/submit", strings.NewReader("user=testuser&password=password"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Body.Len() == 0 {
		t.Error("expected non-empty response")
	}

	cookieFound := false
	for _, c := range w.Result().Cookies() {
		if c.Name == handlers.SessionCookieID && c.Value != "" {
			cookieFound = true
		}
	}
	if !cookieFound {
		t.Error("session cookie should be set")
	}
}

func TestLoginSubmit_InvalidCredentials_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password", ports.RoleUser)

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.LoginSubmit(logger, db)

	req := httptest.NewRequest("POST", "/submit", strings.NewReader("user=testuser&password=wrongpassword"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "Login failed") {
		t.Errorf("expected login failed message, got: %s", w.Body.String())
	}
}

func TestCreateProjectHandler_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	ctx := context.Background()
	token, _ := db.GetUserToken(ctx, "firstadmin", "adminpass")

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.CreateProjectHandler(logger, db, db)

	req := httptest.NewRequest("POST", "/project/create", strings.NewReader("name=NewProject"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: token.Cookie.Value})
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "Created project") {
		t.Errorf("expected project created message, got: %s", w.Body.String())
	}
}

func TestCreateProjectHandler_EmptyName_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	ctx := context.Background()
	token, _ := db.GetUserToken(ctx, "firstadmin", "adminpass")

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.CreateProjectHandler(logger, db, db)

	req := httptest.NewRequest("POST", "/project/create", strings.NewReader("name="))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: token.Cookie.Value})
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "Project name is required") {
		t.Errorf("expected name required message, got: %s", w.Body.String())
	}
}

func TestCreateProjectHandler_NoCookie_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.CreateProjectHandler(logger, db, db)

	req := httptest.NewRequest("POST", "/project/create", strings.NewReader("name=Test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "Please login first") {
		t.Errorf("expected login message, got: %s", w.Body.String())
	}
}

func TestCreateProjectHandler_InvalidSession_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.CreateProjectHandler(logger, db, db)

	req := httptest.NewRequest("POST", "/project/create", strings.NewReader("name=Test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: "invalid-token"})
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "Session invalid") {
		t.Errorf("expected session invalid message, got: %s", w.Body.String())
	}
}

func TestAdminUsersHandler_Real(t *testing.T) {
	if _, err := os.Stat("internal/adapters/htmlx/admin_users.html"); os.IsNotExist(err) {
		t.Skip("skipping test: HTML templates not accessible from package directory")
	}

	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	ctx := context.Background()
	token, _ := db.GetUserToken(ctx, "firstadmin", "adminpass")

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.AdminUsersHandler(logger, db)

	req := httptest.NewRequest("GET", "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: token.Cookie.Value})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "firstadmin") {
		t.Errorf("expected firstadmin in response, got: %s", w.Body.String())
	}
}

func TestAdminUsersHandler_NonAdmin_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "regularuser", "password", ports.RoleUser)
	token, _ := db.GetUserToken(ctx, "regularuser", "password")

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.AdminUsersHandler(logger, db)

	req := httptest.NewRequest("GET", "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: token.Cookie.Value})
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "Access denied") {
		t.Errorf("expected access denied message, got: %s", w.Body.String())
	}
}

func TestAdminUsersHandler_NoCookie_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.AdminUsersHandler(logger, db)

	req := httptest.NewRequest("GET", "/admin/users", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "Please login first") {
		t.Errorf("expected login message, got: %s", w.Body.String())
	}
}

func TestAdminUsersHandler_InvalidSession_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.AdminUsersHandler(logger, db)

	req := httptest.NewRequest("GET", "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: "invalid-token"})
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "Session invalid") {
		t.Errorf("expected session invalid message, got: %s", w.Body.String())
	}
}

func TestAdminCreateUserHandler_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	ctx := context.Background()
	token, _ := db.GetUserToken(ctx, "firstadmin", "adminpass")

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.AdminCreateUserHandler(logger, db)

	req := httptest.NewRequest("POST", "/admin/user/create", strings.NewReader("username=newuser&password=newpass"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: token.Cookie.Value})
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "created successfully") {
		t.Errorf("expected success message, got: %s", w.Body.String())
	}
}

func TestAdminCreateUserHandler_EmptyFields_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	ctx := context.Background()
	token, _ := db.GetUserToken(ctx, "firstadmin", "adminpass")

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.AdminCreateUserHandler(logger, db)

	req := httptest.NewRequest("POST", "/admin/user/create", strings.NewReader("username=&password="))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: token.Cookie.Value})
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "required") {
		t.Errorf("expected required message, got: %s", w.Body.String())
	}
}

func TestAdminCreateUserHandler_NonAdmin_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "regularuser", "password", ports.RoleUser)
	token, _ := db.GetUserToken(ctx, "regularuser", "password")

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.AdminCreateUserHandler(logger, db)

	req := httptest.NewRequest("POST", "/admin/user/create", strings.NewReader("username=test&password=test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: token.Cookie.Value})
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "Access denied") {
		t.Errorf("expected access denied message, got: %s", w.Body.String())
	}
}

func TestAdminCreateUserHandler_NoCookie_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.AdminCreateUserHandler(logger, db)

	req := httptest.NewRequest("POST", "/admin/user/create", strings.NewReader("username=test&password=test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "Please login first") {
		t.Errorf("expected login message, got: %s", w.Body.String())
	}
}

func TestAdminCreateUserHandler_InvalidSession_Real(t *testing.T) {
	db, cleanup := setupDBForHandlerTest(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.AdminCreateUserHandler(logger, db)

	req := httptest.NewRequest("POST", "/admin/user/create", strings.NewReader("username=test&password=test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: "invalid-token"})
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "Session invalid") {
		t.Errorf("expected session invalid message, got: %s", w.Body.String())
	}
}

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
	return models.Project{ID: "test", Name: name, Owner: owner}, nil
}

func (m *mockProjectRepo) GetProjectsByOwner(ctx context.Context, owner string) ([]models.Project, error) {
	return m.projects, nil
}

func (m *mockProjectRepo) DeleteProject(ctx context.Context, id string) error {
	return nil
}

func TestProjectHandler_WithProjects(t *testing.T) {
	if _, err := os.Stat("internal/adapters/htmlx/project.html"); os.IsNotExist(err) {
		t.Skip("skipping test: HTML templates not accessible from package directory")
	}

	userRepo := &mockUserRepo{
		users: map[string]ports.User{
			"testuser": {
				Name: "testuser",
				Role: ports.RoleUser,
				Cookie: ports.SessionCookie{
					Value:      "valid-token",
					CreateTime: time.Now(),
				},
			},
		},
	}
	projectRepo := &mockProjectRepo{
		projects: []models.Project{
			{ID: "1", Name: "Project1", Owner: "testuser"},
			{ID: "2", Name: "Project2", Owner: "testuser"},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.ProjectHandler(logger, userRepo, projectRepo)

	req := httptest.NewRequest("GET", "/project", nil)
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: "valid-token"})
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "Project1") {
		t.Errorf("expected Project1 in response, got: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Project2") {
		t.Errorf("expected Project2 in response, got: %s", w.Body.String())
	}
}

func TestCreateProjectHandler_WithMock(t *testing.T) {
	userRepo := &mockUserRepo{
		users: map[string]ports.User{
			"testuser": {
				Name: "testuser",
				Role: ports.RoleUser,
				Cookie: ports.SessionCookie{
					Value:      "valid-token",
					CreateTime: time.Now(),
				},
			},
		},
	}
	projectRepo := &mockProjectRepo{}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.CreateProjectHandler(logger, userRepo, projectRepo)

	req := httptest.NewRequest("POST", "/project/create", strings.NewReader("name=NewProject"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: "valid-token"})
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "NewProject") {
		t.Errorf("expected NewProject in response, got: %s", w.Body.String())
	}
}

func TestAdminUsersHandler_WithMock(t *testing.T) {
	if _, err := os.Stat("internal/adapters/htmlx/admin_users.html"); os.IsNotExist(err) {
		t.Skip("skipping test: HTML templates not accessible from package directory")
	}

	userRepo := &mockUserRepo{
		users: map[string]ports.User{
			"admin": {
				Name: "admin",
				Role: ports.RoleAdmin,
				Cookie: ports.SessionCookie{
					Value:      "admin-token",
					CreateTime: time.Now(),
				},
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.AdminUsersHandler(logger, userRepo)

	req := httptest.NewRequest("GET", "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: "admin-token"})
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "admin") {
		t.Errorf("expected admin in response, got: %s", w.Body.String())
	}
}

func TestLoginSubmit_WithMock(t *testing.T) {
	userRepo := &mockUserRepoWithError{}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.LoginSubmit(logger, userRepo)

	req := httptest.NewRequest("POST", "/submit", strings.NewReader("user=test&password=test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "Login failed") {
		t.Errorf("expected login failed message, got: %s", w.Body.String())
	}
}

type mockUserRepoWithError struct{}

func (m *mockUserRepoWithError) GetUserToken(ctx context.Context, name, password string) (*ports.LoginResult, error) {
	return nil, ports.UserNotFound
}

func (m *mockUserRepoWithError) CountUsers(ctx context.Context) (int, error) {
	return 1, nil
}

func (m *mockUserRepoWithError) IsValid(ctx context.Context, c string) (*ports.User, error) {
	return nil, ports.UserNotFound
}

func (m *mockUserRepoWithError) Create(ctx context.Context, name, password, role string) error {
	return nil
}

func (m *mockUserRepoWithError) Delete(ctx context.Context, name string) error {
	return nil
}

func (m *mockUserRepoWithError) GetAll(ctx context.Context) ([]ports.User, error) {
	return nil, nil
}

func (m *mockUserRepoWithError) ChangePassword(ctx context.Context, name, newPassword string) error {
	return nil
}

func (m *mockUserRepoWithError) Logout(ctx context.Context, name string) error {
	return nil
}

func (m *mockUserRepoWithError) GetAdmin(ctx context.Context) (*ports.User, error) {
	return nil, ports.UserNotFound
}

func (m *mockUserRepoWithError) CountAdmins(ctx context.Context) (int, error) {
	return 0, nil
}

var _ ports.UserRepo = (*mockUserRepoWithError)(nil)

type mockProjectRepoWithError struct{}

func (m *mockProjectRepoWithError) CreateProject(ctx context.Context, name, owner string) (models.Project, error) {
	return models.Project{}, ports.DatabaseError
}

func (m *mockProjectRepoWithError) GetProjectsByOwner(ctx context.Context, owner string) ([]models.Project, error) {
	return nil, ports.DatabaseError
}

func (m *mockProjectRepoWithError) DeleteProject(ctx context.Context, id string) error {
	return nil
}

var _ ports.ProjectRepo = (*mockProjectRepoWithError)(nil)

func TestProjectHandler_GetProjectsError(t *testing.T) {
	if _, err := os.Stat("internal/adapters/htmlx/project.html"); os.IsNotExist(err) {
		t.Skip("skipping test: HTML templates not accessible from package directory")
	}

	userRepo := &mockUserRepo{
		users: map[string]ports.User{
			"testuser": {
				Name: "testuser",
				Role: ports.RoleUser,
				Cookie: ports.SessionCookie{
					Value:      "valid-token",
					CreateTime: time.Now(),
				},
			},
		},
	}
	projectRepo := &mockProjectRepoWithError{}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.ProjectHandler(logger, userRepo, projectRepo)

	req := httptest.NewRequest("GET", "/project", nil)
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: "valid-token"})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 even with error, got %d", w.Code)
	}
}

func TestCreateProjectHandler_ProjectRepoError(t *testing.T) {
	userRepo := &mockUserRepo{
		users: map[string]ports.User{
			"testuser": {
				Name: "testuser",
				Role: ports.RoleUser,
				Cookie: ports.SessionCookie{
					Value:      "valid-token",
					CreateTime: time.Now(),
				},
			},
		},
	}
	projectRepo := &mockProjectRepoWithError{}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.CreateProjectHandler(logger, userRepo, projectRepo)

	req := httptest.NewRequest("POST", "/project/create", strings.NewReader("name=NewProject"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: "valid-token"})
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "Failed to create project") {
		t.Errorf("expected failed message, got: %s", w.Body.String())
	}
}

type mockUserRepoWithGetAllError struct {
	mockUserRepo
}

func (m *mockUserRepoWithGetAllError) GetAll(ctx context.Context) ([]ports.User, error) {
	return nil, ports.DatabaseError
}

var _ ports.UserRepo = (*mockUserRepoWithGetAllError)(nil)

func TestAdminUsersHandler_GetAllError(t *testing.T) {
	if _, err := os.Stat("internal/adapters/htmlx/admin_users.html"); os.IsNotExist(err) {
		t.Skip("skipping test: HTML templates not accessible from package directory")
	}

	userRepo := &mockUserRepoWithGetAllError{
		mockUserRepo: mockUserRepo{
			users: map[string]ports.User{
				"admin": {
					Name: "admin",
					Role: ports.RoleAdmin,
					Cookie: ports.SessionCookie{
						Value:      "admin-token",
						CreateTime: time.Now(),
					},
				},
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.AdminUsersHandler(logger, userRepo)

	req := httptest.NewRequest("GET", "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: "admin-token"})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 even with error, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "admin") {
		t.Errorf("expected admin in response, got: %s", w.Body.String())
	}
}

type mockUserRepoWithCreateError struct {
	mockUserRepo
}

func (m *mockUserRepoWithCreateError) Create(ctx context.Context, name, password, role string) error {
	return ports.DatabaseError
}

var _ ports.UserRepo = (*mockUserRepoWithCreateError)(nil)

func TestAdminCreateUserHandler_CreateError(t *testing.T) {
	userRepo := &mockUserRepoWithCreateError{
		mockUserRepo: mockUserRepo{
			users: map[string]ports.User{
				"admin": {
					Name: "admin",
					Role: ports.RoleAdmin,
					Cookie: ports.SessionCookie{
						Value:      "admin-token",
						CreateTime: time.Now(),
					},
				},
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.AdminCreateUserHandler(logger, userRepo)

	req := httptest.NewRequest("POST", "/admin/user/create", strings.NewReader("username=test&password=test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: "admin-token"})
	w := httptest.NewRecorder()

	handler(w, req)

	if !strings.Contains(w.Body.String(), "Failed to create user") {
		t.Errorf("expected failed message, got: %s", w.Body.String())
	}
}

type mockUserRepoWithLogoutError struct{}

func (m *mockUserRepoWithLogoutError) GetUserToken(ctx context.Context, name, password string) (*ports.LoginResult, error) {
	return nil, nil
}

func (m *mockUserRepoWithLogoutError) CountUsers(ctx context.Context) (int, error) {
	return 0, nil
}

func (m *mockUserRepoWithLogoutError) IsValid(ctx context.Context, c string) (*ports.User, error) {
	return &ports.User{Name: "testuser", Role: ports.RoleUser}, nil
}

func (m *mockUserRepoWithLogoutError) Create(ctx context.Context, name, password, role string) error {
	return nil
}

func (m *mockUserRepoWithLogoutError) Delete(ctx context.Context, name string) error {
	return nil
}

func (m *mockUserRepoWithLogoutError) GetAll(ctx context.Context) ([]ports.User, error) {
	return nil, nil
}

func (m *mockUserRepoWithLogoutError) ChangePassword(ctx context.Context, name, newPassword string) error {
	return nil
}

func (m *mockUserRepoWithLogoutError) Logout(ctx context.Context, name string) error {
	return ports.DatabaseError
}

func (m *mockUserRepoWithLogoutError) GetAdmin(ctx context.Context) (*ports.User, error) {
	return nil, ports.UserNotFound
}

func (m *mockUserRepoWithLogoutError) CountAdmins(ctx context.Context) (int, error) {
	return 0, nil
}

var _ ports.UserRepo = (*mockUserRepoWithLogoutError)(nil)

func TestLogoutHandler_LogoutError(t *testing.T) {
	userRepo := &mockUserRepoWithLogoutError{}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := handlers.LogoutHandler(logger, userRepo)

	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{Name: handlers.SessionCookieID, Value: "some-token"})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

var _ ports.UserRepo = (*mockUserRepo)(nil)
var _ ports.ProjectRepo = (*mockProjectRepo)(nil)
