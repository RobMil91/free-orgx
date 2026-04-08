package mocks_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RobMil91/free-orgx/internal/adapters/mocks"
	"github.com/RobMil91/free-orgx/internal/models"
	"github.com/RobMil91/free-orgx/internal/ports"
)

func TestRAM_Create(t *testing.T) {
	r := &mocks.RAM{}
	ctx := context.Background()

	err := r.Create(ctx, "testuser", "password", ports.RoleUser)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	user, ok := r.Users["testuser"]
	if !ok {
		t.Fatal("user not created in map")
	}
	if user.Name != "testuser" {
		t.Errorf("expected name 'testuser', got '%s'", user.Name)
	}
	if user.Password != "password" {
		t.Errorf("expected password 'password', got '%s'", user.Password)
	}
	if user.Role != ports.RoleUser {
		t.Errorf("expected role 'user', got '%s'", user.Role)
	}
}

func TestRAM_Create_ExistingUser(t *testing.T) {
	r := &mocks.RAM{
		Users: map[string]ports.User{
			"existing": {Name: "existing", Password: "pass", Role: ports.RoleUser},
		},
	}
	ctx := context.Background()

	err := r.Create(ctx, "existing", "newpass", ports.RoleUser)
	if err != nil {
		t.Fatalf("Create() should not fail for existing user: %v", err)
	}
}

func TestRAM_Delete(t *testing.T) {
	r := &mocks.RAM{
		Users: map[string]ports.User{
			"testuser": {Name: "testuser", Password: "password", Role: ports.RoleUser},
		},
	}
	ctx := context.Background()

	err := r.Delete(ctx, "testuser")
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	if _, ok := r.Users["testuser"]; ok {
		t.Error("user should be deleted")
	}
}

func TestRAM_Delete_NonExistent(t *testing.T) {
	r := &mocks.RAM{}
	ctx := context.Background()

	err := r.Delete(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Delete() should not fail for nonexistent user: %v", err)
	}
}

func TestRAM_Logout(t *testing.T) {
	r := &mocks.RAM{
		Users: map[string]ports.User{
			"testuser": {
				Name:     "testuser",
				Password: "password",
				Role:     ports.RoleUser,
				Cookie:   ports.SessionCookie{Value: "session-token"},
			},
		},
	}
	ctx := context.Background()

	err := r.Logout(ctx, "testuser")
	if err != nil {
		t.Fatalf("Logout() failed: %v", err)
	}

	if r.Users["testuser"].Cookie.Value != "" {
		t.Error("cookie should be cleared after logout")
	}
}

func TestRAM_Logout_NonExistent(t *testing.T) {
	r := &mocks.RAM{}
	ctx := context.Background()

	err := r.Logout(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Logout() should not fail: %v", err)
	}
}

func TestRAM_GetAdmin(t *testing.T) {
	r := &mocks.RAM{
		Users: map[string]ports.User{
			"admin": {Name: "admin", Password: "pass", Role: ports.RoleAdmin},
			"user":  {Name: "user", Password: "pass", Role: ports.RoleUser},
		},
	}
	ctx := context.Background()

	admin, err := r.GetAdmin(ctx)
	if err != nil {
		t.Fatalf("GetAdmin() failed: %v", err)
	}
	if admin.Name != "admin" {
		t.Errorf("expected admin 'admin', got '%s'", admin.Name)
	}
}

func TestRAM_GetAdmin_NotFound(t *testing.T) {
	r := &mocks.RAM{
		Users: map[string]ports.User{
			"user": {Name: "user", Password: "pass", Role: ports.RoleUser},
		},
	}
	ctx := context.Background()

	_, err := r.GetAdmin(ctx)
	if !errors.Is(err, ports.UserNotFound) {
		t.Errorf("expected UserNotFound, got: %v", err)
	}
}

func TestRAM_GetAdmin_Empty(t *testing.T) {
	r := &mocks.RAM{}
	ctx := context.Background()

	_, err := r.GetAdmin(ctx)
	if !errors.Is(err, ports.UserNotFound) {
		t.Errorf("expected UserNotFound, got: %v", err)
	}
}

func TestRAM_CountAdmins(t *testing.T) {
	r := &mocks.RAM{
		Users: map[string]ports.User{
			"admin1": {Name: "admin1", Password: "pass", Role: ports.RoleAdmin},
			"admin2": {Name: "admin2", Password: "pass", Role: ports.RoleAdmin},
			"user":   {Name: "user", Password: "pass", Role: ports.RoleUser},
		},
	}
	ctx := context.Background()

	count, err := r.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins() failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 admins, got %d", count)
	}
}

func TestRAM_CountAdmins_None(t *testing.T) {
	r := &mocks.RAM{
		Users: map[string]ports.User{
			"user": {Name: "user", Password: "pass", Role: ports.RoleUser},
		},
	}
	ctx := context.Background()

	count, err := r.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins() failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 admins, got %d", count)
	}
}

func TestRAM_CountAdmins_Empty(t *testing.T) {
	r := &mocks.RAM{}
	ctx := context.Background()

	count, err := r.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins() failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 admins, got %d", count)
	}
}

func TestRAM_CreateProject(t *testing.T) {
	r := &mocks.RAM{}
	ctx := context.Background()

	project, err := r.CreateProject(ctx, "MyProject", "owner")
	if err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	if project.Name != "MyProject" {
		t.Errorf("expected name 'MyProject', got '%s'", project.Name)
	}
	if project.Owner != "owner" {
		t.Errorf("expected owner 'owner', got '%s'", project.Owner)
	}
	if project.ID == "" {
		t.Error("project ID should not be empty")
	}
	if project.Created == "" {
		t.Error("project Created should not be empty")
	}

	if len(r.Projects["owner"]) != 1 {
		t.Errorf("expected 1 project in map, got %d", len(r.Projects["owner"]))
	}
}

func TestRAM_GetProjectsByOwner(t *testing.T) {
	r := &mocks.RAM{
		Projects: map[string][]models.Project{
			"owner1": {
				{ID: "1", Name: "Project1", Owner: "owner1"},
				{ID: "2", Name: "Project2", Owner: "owner1"},
			},
			"owner2": {
				{ID: "3", Name: "Project3", Owner: "owner2"},
			},
		},
	}
	ctx := context.Background()

	projects, err := r.GetProjectsByOwner(ctx, "owner1")
	if err != nil {
		t.Fatalf("GetProjectsByOwner() failed: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

func TestRAM_GetProjectsByOwner_Empty(t *testing.T) {
	r := &mocks.RAM{}
	ctx := context.Background()

	projects, err := r.GetProjectsByOwner(ctx, "owner")
	if err != nil {
		t.Fatalf("GetProjectsByOwner() failed: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestRAM_DeleteProject(t *testing.T) {
	r := &mocks.RAM{
		Projects: map[string][]models.Project{
			"owner": {
				{ID: "1", Name: "Project1", Owner: "owner"},
				{ID: "2", Name: "Project2", Owner: "owner"},
			},
		},
	}
	ctx := context.Background()

	err := r.DeleteProject(ctx, "1")
	if err != nil {
		t.Fatalf("DeleteProject() failed: %v", err)
	}

	if len(r.Projects["owner"]) != 1 {
		t.Errorf("expected 1 project remaining, got %d", len(r.Projects["owner"]))
	}
	if r.Projects["owner"][0].ID == "1" {
		t.Error("project 1 should be deleted")
	}
}

func TestRAM_DeleteProject_NotFound(t *testing.T) {
	r := &mocks.RAM{
		Projects: map[string][]models.Project{
			"owner": {
				{ID: "1", Name: "Project1", Owner: "owner"},
			},
		},
	}
	ctx := context.Background()

	err := r.DeleteProject(ctx, "nonexistent")
	if err == nil {
		t.Error("DeleteProject() should fail for nonexistent project")
	}
}

func TestRAM_GetUserToken_ExistingUser(t *testing.T) {
	r := &mocks.RAM{
		Users: map[string]ports.User{
			"testuser": {Name: "testuser", Password: "password", Role: ports.RoleUser},
		},
	}
	ctx := context.Background()

	result, err := r.GetUserToken(ctx, "testuser", "password")
	if err != nil {
		t.Fatalf("GetUserToken() failed: %v", err)
	}

	if result.User.Name != "testuser" {
		t.Errorf("expected user 'testuser', got '%s'", result.User.Name)
	}
	if result.IsNewUser {
		t.Error("existing user should not be marked as new")
	}
	if result.Cookie.Value == "" {
		t.Error("cookie value should not be empty")
	}
}

func TestRAM_GetUserToken_WrongPassword(t *testing.T) {
	r := &mocks.RAM{
		Users: map[string]ports.User{
			"testuser": {Name: "testuser", Password: "password", Role: ports.RoleUser},
		},
	}
	ctx := context.Background()

	_, err := r.GetUserToken(ctx, "testuser", "wrongpassword")
	if err == nil {
		t.Error("GetUserToken() should fail with wrong password")
	}
}

func TestRAM_GetUserToken_UserNotFound(t *testing.T) {
	r := &mocks.RAM{
		Users: map[string]ports.User{
			"testuser": {Name: "testuser", Password: "password", Role: ports.RoleUser},
		},
	}
	ctx := context.Background()

	_, err := r.GetUserToken(ctx, "nonexistent", "password")
	if !errors.Is(err, ports.UserNotFound) {
		t.Errorf("expected UserNotFound, got: %v", err)
	}
}

func TestRAM_GetUserToken_FirstAdmin(t *testing.T) {
	r := &mocks.RAM{}
	ctx := context.Background()

	result, err := r.GetUserToken(ctx, "firstadmin", "password")
	if err != nil {
		t.Fatalf("GetUserToken() failed: %v", err)
	}

	if !result.IsNewUser {
		t.Error("first user should be marked as new")
	}
	if result.User.Role != ports.RoleAdmin {
		t.Errorf("first user should be admin, got: %s", result.User.Role)
	}
	if result.User.Name != "firstadmin" {
		t.Errorf("expected name 'firstadmin', got '%s'", result.User.Name)
	}
}

func TestRAM_CountUsers(t *testing.T) {
	r := &mocks.RAM{
		Users: map[string]ports.User{
			"user1": {Name: "user1", Password: "pass", Role: ports.RoleUser},
			"user2": {Name: "user2", Password: "pass", Role: ports.RoleUser},
		},
	}
	ctx := context.Background()

	count, err := r.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers() failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 users, got %d", count)
	}
}

func TestRAM_CountUsers_Empty(t *testing.T) {
	r := &mocks.RAM{}
	ctx := context.Background()

	count, err := r.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers() failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 users, got %d", count)
	}
}

func TestRAM_IsValid(t *testing.T) {
	r := &mocks.RAM{
		Users: map[string]ports.User{
			"testuser": {
				Name:     "testuser",
				Password: "password",
				Role:     ports.RoleUser,
				Cookie: ports.SessionCookie{
					Value:      "valid-token",
					CreateTime: time.Now(),
				},
			},
		},
	}
	ctx := context.Background()

	user, err := r.IsValid(ctx, "valid-token")
	if err != nil {
		t.Fatalf("IsValid() failed: %v", err)
	}
	if user.Name != "testuser" {
		t.Errorf("expected user 'testuser', got '%s'", user.Name)
	}
}

func TestRAM_IsValid_ExpiredToken(t *testing.T) {
	r := &mocks.RAM{
		Users: map[string]ports.User{
			"testuser": {
				Name:     "testuser",
				Password: "password",
				Role:     ports.RoleUser,
				Cookie: ports.SessionCookie{
					Value:      "expired-token",
					CreateTime: time.Now().Add(-10 * time.Minute),
				},
			},
		},
	}
	ctx := context.Background()

	_, err := r.IsValid(ctx, "expired-token")
	if err == nil {
		t.Error("IsValid() should fail for expired token")
	}
}

func TestRAM_IsValid_InvalidToken(t *testing.T) {
	r := &mocks.RAM{
		Users: map[string]ports.User{
			"testuser": {
				Name:     "testuser",
				Password: "password",
				Role:     ports.RoleUser,
				Cookie: ports.SessionCookie{
					Value:      "valid-token",
					CreateTime: time.Now(),
				},
			},
		},
	}
	ctx := context.Background()

	_, err := r.IsValid(ctx, "invalid-token")
	if err == nil {
		t.Error("IsValid() should fail for invalid token")
	}
}

func TestRAM_IsValid_EmptyUsers(t *testing.T) {
	r := &mocks.RAM{}
	ctx := context.Background()

	_, err := r.IsValid(ctx, "any-token")
	if err == nil {
		t.Error("IsValid() should fail when no users exist")
	}
}

func TestRAM_GetAll(t *testing.T) {
	r := &mocks.RAM{
		Users: map[string]ports.User{
			"user1": {Name: "user1", Password: "pass", Role: ports.RoleUser},
			"user2": {Name: "user2", Password: "pass", Role: ports.RoleAdmin},
		},
	}
	ctx := context.Background()

	users, err := r.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestRAM_GetAll_Empty(t *testing.T) {
	r := &mocks.RAM{}
	ctx := context.Background()

	users, err := r.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() failed: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}
}

func TestRAM_ChangePassword(t *testing.T) {
	r := &mocks.RAM{
		Users: map[string]ports.User{
			"testuser": {Name: "testuser", Password: "oldpass", Role: ports.RoleUser},
		},
	}
	ctx := context.Background()

	err := r.ChangePassword(ctx, "testuser", "newpass")
	if err != nil {
		t.Fatalf("ChangePassword() failed: %v", err)
	}

	if r.Users["testuser"].Password != "newpass" {
		t.Error("password should be updated")
	}
}

func TestRAM_ChangePassword_UserNotFound(t *testing.T) {
	r := &mocks.RAM{}
	ctx := context.Background()

	err := r.ChangePassword(ctx, "nonexistent", "newpass")
	if !errors.Is(err, ports.UserNotFound) {
		t.Errorf("expected UserNotFound, got: %v", err)
	}
}

func TestCreateRandStr(t *testing.T) {
	str1, err := mocks.CreateRandStr(32)
	if err != nil {
		t.Fatalf("CreateRandStr() failed: %v", err)
	}
	if *str1 == "" {
		t.Error("string should not be empty")
	}

	str2, err := mocks.CreateRandStr(32)
	if err != nil {
		t.Fatalf("CreateRandStr() failed: %v", err)
	}
	if *str1 == *str2 {
		t.Error("two calls should produce different strings")
	}

	shortStr, err := mocks.CreateRandStr(8)
	if err != nil {
		t.Fatalf("CreateRandStr(8) failed: %v", err)
	}
	if *shortStr == "" {
		t.Error("short string should not be empty")
	}
}
