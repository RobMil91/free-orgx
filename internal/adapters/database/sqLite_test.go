package database_test

import (
	"context"
	"os"
	"testing"

	"github.com/RobMil91/free-orgx/internal/adapters/database"
	"github.com/RobMil91/free-orgx/internal/ports"
)

func setupTestDB(t *testing.T) (*database.SQLiteAdapter, func()) {
	os.Setenv("PASSWORD_PEPPER", "test-pepper-"+t.Name())

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

func TestSQLite_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	err := db.Create(ctx, "testuser", "password123")
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	users, err := db.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() failed: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
	if users[0].Name != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", users[0].Name)
	}
}

func TestSQLite_Create_DuplicateUser(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password123")

	err := db.Create(ctx, "testuser", "password456")
	if err == nil {
		t.Error("Create() should fail for duplicate user")
	}
}

func TestSQLite_GetUserToken(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password123")

	token, err := db.GetUserToken(ctx, "testuser", "password123")
	if err != nil {
		t.Fatalf("GetUserToken() failed: %v", err)
	}

	if token == nil {
		t.Fatal("GetUserToken() returned nil token")
	}
	if token.Value == "" {
		t.Error("GetUserToken() returned empty token value")
	}
}

func TestSQLite_GetUserToken_WrongPassword(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password123")

	_, err := db.GetUserToken(ctx, "testuser", "wrongpassword")
	if err == nil {
		t.Error("GetUserToken() should fail with wrong password")
	}
}

func TestSQLite_GetUserToken_UserNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := db.GetUserToken(ctx, "nonexistent", "password")
	if err == nil {
		t.Error("GetUserToken() should fail for nonexistent user")
	}
}

func TestSQLite_IsValid(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password123")

	token, _ := db.GetUserToken(ctx, "testuser", "password123")

	user, err := db.IsValid(ctx, token.Value)
	if err != nil {
		t.Fatalf("IsValid() failed: %v", err)
	}

	if user == nil {
		t.Fatal("IsValid() returned nil user")
	}
	if user.Name != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", user.Name)
	}
}

func TestSQLite_IsValid_InvalidToken(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := db.IsValid(ctx, "invalid-token")
	if err == nil {
		t.Error("IsValid() should fail for invalid token")
	}
}

func TestSQLite_ChangePassword(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password123")

	err := db.ChangePassword(ctx, "testuser", "newpassword456")
	if err != nil {
		t.Fatalf("ChangePassword() failed: %v", err)
	}

	_, err = db.GetUserToken(ctx, "testuser", "password123")
	if err == nil {
		t.Error("old password should no longer work")
	}

	token, err := db.GetUserToken(ctx, "testuser", "newpassword456")
	if err != nil {
		t.Errorf("new password should work: %v", err)
	}
	if token == nil {
		t.Error("GetUserToken() returned nil after password change")
	}
}

func TestSQLite_ChangePassword_UserNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	err := db.ChangePassword(ctx, "nonexistent", "newpassword")
	if err == nil {
		t.Error("ChangePassword() should fail for nonexistent user")
	}
}

func TestSQLite_Logout(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password123")

	token, _ := db.GetUserToken(ctx, "testuser", "password123")

	err := db.Logout(ctx, "testuser")
	if err != nil {
		t.Fatalf("Logout() failed: %v", err)
	}

	_, err = db.IsValid(ctx, token.Value)
	if err == nil {
		t.Error("token should be invalid after logout")
	}
}

func TestSQLite_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password123")

	err := db.Delete(ctx, "testuser")
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	users, _ := db.GetAll(ctx)
	if len(users) != 0 {
		t.Errorf("expected 0 users after delete, got %d", len(users))
	}
}

func TestSQLite_GetAll(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "user1", "password1")
	db.Create(ctx, "user2", "password2")

	users, err := db.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() failed: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestSQLite_GetAll_Empty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	users, err := db.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() failed: %v", err)
	}

	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}
}

func TestSQLite_CreateProject(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	project, err := db.CreateProject(ctx, "My Project", "testuser")
	if err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	if project.Name != "My Project" {
		t.Errorf("expected project name 'My Project', got '%s'", project.Name)
	}
	if project.ID == "" {
		t.Error("project ID should not be empty")
	}
	if project.Owner != "testuser" {
		t.Errorf("expected owner 'testuser', got '%s'", project.Owner)
	}
}

func TestSQLite_GetProjectsByOwner(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.CreateProject(ctx, "Project1", "testuser")
	db.CreateProject(ctx, "Project2", "testuser")
	db.CreateProject(ctx, "OtherProject", "otheruser")

	projects, err := db.GetProjectsByOwner(ctx, "testuser")
	if err != nil {
		t.Fatalf("GetProjectsByOwner() failed: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

func TestSQLite_GetProjectsByOwner_Empty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	projects, err := db.GetProjectsByOwner(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetProjectsByOwner() failed: %v", err)
	}

	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestSQLite_DeleteProject(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	project, _ := db.CreateProject(ctx, "My Project", "testuser")

	err := db.DeleteProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("DeleteProject() failed: %v", err)
	}
}

func TestSQLite_DeleteProject_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	err := db.DeleteProject(ctx, "nonexistent-id")
	if err == nil {
		t.Error("DeleteProject() should fail for nonexistent project")
	}
}

func TestSQLite_NoPepper(t *testing.T) {
	os.Unsetenv("PASSWORD_PEPPER")

	db, err := database.NewSQLite()
	if err != nil {
		t.Fatalf("NewSQLite() failed: %v", err)
	}
	defer func() {
		db.Conn.Close()
		os.Remove("orgxdb.db")
	}()

	ctx := context.Background()
	err = db.Create(ctx, "testuser", "password")
	if err == nil {
		t.Error("Create() should fail when PASSWORD_PEPPER is not set")
	}

	_, err = db.GetUserToken(ctx, "testuser", "password")
	if err == nil {
		t.Error("GetUserToken() should fail when PASSWORD_PEPPER is not set")
	}

	_ = db.ChangePassword(ctx, "testuser", "newpass")
}

var _ ports.UserRepo = (*database.SQLiteAdapter)(nil)
var _ ports.ProjectRepo = (*database.SQLiteAdapter)(nil)
