package database_test

import (
	"context"
	"errors"
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
	err := db.Create(ctx, "testuser", "password123", ports.RoleUser)
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
	db.Create(ctx, "testuser", "password123", ports.RoleUser)

	err := db.Create(ctx, "testuser", "password456", ports.RoleUser)
	if err == nil {
		t.Error("Create() should fail for duplicate user")
	}
}

func TestSQLite_GetUserToken(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password123", ports.RoleUser)

	token, err := db.GetUserToken(ctx, "testuser", "password123")
	if err != nil {
		t.Fatalf("GetUserToken() failed: %v", err)
	}

	if token == nil {
		t.Fatal("GetUserToken() returned nil token")
	}
	if token.Cookie.Value == "" {
		t.Error("GetUserToken() returned empty token value")
	}
}

func TestSQLite_GetUserToken_WrongPassword(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password123", ports.RoleUser)

	_, err := db.GetUserToken(ctx, "testuser", "wrongpassword")
	if err == nil {
		t.Error("GetUserToken() should fail with wrong password")
	}
}

func TestSQLite_GetUserToken_UserNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "existinguser", "password", ports.RoleUser)

	_, err := db.GetUserToken(ctx, "nonexistent", "password")
	if err == nil {
		t.Error("GetUserToken() should fail for nonexistent user when users exist")
	}
	if !errors.Is(err, ports.UserNotFound) {
		t.Errorf("expected UserNotFound error, got: %v", err)
	}
}

func TestSQLite_IsValid(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password123", ports.RoleUser)

	token, _ := db.GetUserToken(ctx, "testuser", "password123")

	user, err := db.IsValid(ctx, token.Cookie.Value)
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
	db.Create(ctx, "testuser", "password123", ports.RoleUser)

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

func TestSQLite_Logout(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password123", ports.RoleUser)

	token, _ := db.GetUserToken(ctx, "testuser", "password123")

	err := db.Logout(ctx, "testuser")
	if err != nil {
		t.Fatalf("Logout() failed: %v", err)
	}

	_, err = db.IsValid(ctx, token.Cookie.Value)
	if err == nil {
		t.Error("token should be invalid after logout")
	}
}

func TestSQLite_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password123", ports.RoleUser)

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
	db.Create(ctx, "user1", "password1", ports.RoleUser)
	db.Create(ctx, "user2", "password2", ports.RoleUser)

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
	err = db.Create(ctx, "testuser", "password", ports.RoleUser)
	if err == nil {
		t.Error("Create() should fail when PASSWORD_PEPPER is not set")
	}

	_, err = db.GetUserToken(ctx, "testuser", "password")
	if err == nil {
		t.Error("GetUserToken() should fail when PASSWORD_PEPPER is not set")
	}

	_ = db.ChangePassword(ctx, "testuser", "newpass")
}

func TestSQLite_CreateAdmin(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	err := db.Create(ctx, "admin", "password", ports.RoleAdmin)
	if err != nil {
		t.Fatalf("Create() admin failed: %v", err)
	}

	admin, err := db.GetAdmin(ctx)
	if err != nil {
		t.Fatalf("GetAdmin() failed: %v", err)
	}
	if admin.Name != "admin" {
		t.Errorf("expected admin name 'admin', got '%s'", admin.Name)
	}
	if admin.Role != ports.RoleAdmin {
		t.Errorf("expected role 'admin', got '%s'", admin.Role)
	}
}

func TestSQLite_CreateSecondAdmin_Fails(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "admin1", "password", ports.RoleAdmin)

	err := db.Create(ctx, "admin2", "password", ports.RoleAdmin)
	if err == nil {
		t.Error("Create() should fail when admin already exists")
	}
	if err != ports.AdminExists {
		t.Errorf("expected AdminExists error, got: %v", err)
	}
}

func TestSQLite_GetAdmin_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := db.GetAdmin(ctx)
	if err == nil {
		t.Error("GetAdmin() should fail when no admin exists")
	}
}

func TestSQLite_CountAdmins(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	count, err := db.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins() failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 admins, got %d", count)
	}

	db.Create(ctx, "admin", "password", ports.RoleAdmin)

	count, err = db.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins() failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 admin, got %d", count)
	}
}

func TestSQLite_Create_InvalidRole(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	err := db.Create(ctx, "user", "password", "invalid")
	if err == nil {
		t.Error("Create() should fail for invalid role")
	}
	if err != ports.InvalidRole {
		t.Errorf("expected InvalidRole error, got: %v", err)
	}
}

func TestSQLite_CreateMultipleUsers(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "admin", "password", ports.RoleAdmin)
	db.Create(ctx, "user1", "password", ports.RoleUser)
	db.Create(ctx, "user2", "password", ports.RoleUser)

	users, err := db.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() failed: %v", err)
	}
	if len(users) != 3 {
		t.Errorf("expected 3 users, got %d", len(users))
	}
}

func TestSQLite_GetUserToken_FirstUserBecomesAdmin(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	result, err := db.GetUserToken(ctx, "firstuser", "password")
	if err != nil {
		t.Fatalf("GetUserToken() failed for first user: %v", err)
	}

	if !result.IsNewUser {
		t.Error("first user should be marked as new user")
	}

	if result.User.Role != ports.RoleAdmin {
		t.Errorf("first user should be admin, got: %s", result.User.Role)
	}

	admin, err := db.GetAdmin(ctx)
	if err != nil {
		t.Fatalf("GetAdmin() failed: %v", err)
	}
	if admin.Name != "firstuser" {
		t.Errorf("admin should be firstuser, got: %s", admin.Name)
	}
}

func TestSQLite_GetUserToken_SecondUserDoesNotBecomeAdmin(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	result1, _ := db.GetUserToken(ctx, "firstuser", "password")
	if result1.User.Role != ports.RoleAdmin {
		t.Errorf("first user should be admin, got: %s", result1.User.Role)
	}

	result2, err := db.GetUserToken(ctx, "seconduser", "password")
	if err == nil {
		t.Error("second user should fail to auto-register")
	}
	if !errors.Is(err, ports.UserNotFound) {
		t.Errorf("expected UserNotFound error, got: %v", err)
	}

	_ = result2
}

func TestSQLite_CountUsers(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	count, err := db.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers() failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 users, got %d", count)
	}

	db.Create(ctx, "user1", "password", ports.RoleUser)
	db.Create(ctx, "user2", "password", ports.RoleUser)

	count, err = db.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers() failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 users, got %d", count)
	}
}

var _ ports.UserRepo = (*database.SQLiteAdapter)(nil)
var _ ports.ProjectRepo = (*database.SQLiteAdapter)(nil)

func TestSQLite_Logout_Error(t *testing.T) {
	os.Setenv("PASSWORD_PEPPER", "test-pepper")
	db, err := database.NewSQLiteForTest("logout_error")
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	defer func() {
		db.Conn.Close()
		database.CleanupTestDB("logout_error")
	}()

	db.Conn.Close()

	ctx := context.Background()
	err = db.Logout(ctx, "testuser")
	if err == nil {
		t.Error("Logout() should fail when connection is closed")
	}
}

func TestSQLite_Delete_Error(t *testing.T) {
	os.Setenv("PASSWORD_PEPPER", "test-pepper")
	db, err := database.NewSQLiteForTest("delete_error")
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	defer func() {
		db.Conn.Close()
		database.CleanupTestDB("delete_error")
	}()

	db.Conn.Close()

	ctx := context.Background()
	err = db.Delete(ctx, "testuser")
	if err == nil {
		t.Error("Delete() should fail when connection is closed")
	}
}

func TestSQLite_ChangePassword_NoPepper(t *testing.T) {
	os.Unsetenv("PASSWORD_PEPPER")
	db, err := database.NewSQLiteForTest("change_pass_no_pepper")
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	defer func() {
		db.Conn.Close()
		database.CleanupTestDB("change_pass_no_pepper")
	}()

	ctx := context.Background()
	err = db.ChangePassword(ctx, "testuser", "newpass")
	if err == nil {
		t.Error("ChangePassword() should fail without pepper")
	}
}

func TestSQLite_GetProjectsByOwner_ClosedConnection(t *testing.T) {
	os.Setenv("PASSWORD_PEPPER", "test-pepper")
	db, err := database.NewSQLiteForTest("get_projects_closed")
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	defer func() {
		db.Conn.Close()
		database.CleanupTestDB("get_projects_closed")
	}()

	db.Conn.Close()

	ctx := context.Background()
	_, err = db.GetProjectsByOwner(ctx, "owner")
	if err == nil {
		t.Error("GetProjectsByOwner() should fail when connection is closed")
	}
}

func TestSQLite_DeleteProject_Error(t *testing.T) {
	os.Setenv("PASSWORD_PEPPER", "test-pepper")
	db, err := database.NewSQLiteForTest("delete_proj_error")
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	defer func() {
		db.Conn.Close()
		database.CleanupTestDB("delete_proj_error")
	}()

	db.Conn.Close()

	ctx := context.Background()
	err = db.DeleteProject(ctx, "some-id")
	if err == nil {
		t.Error("DeleteProject() should fail when connection is closed")
	}
}

func TestSQLite_IsValid_ClosedConnection(t *testing.T) {
	os.Setenv("PASSWORD_PEPPER", "test-pepper")
	db, err := database.NewSQLiteForTest("isvalid_closed")
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	defer func() {
		db.Conn.Close()
		database.CleanupTestDB("isvalid_closed")
	}()

	db.Conn.Close()

	ctx := context.Background()
	_, err = db.IsValid(ctx, "some-token")
	if err == nil {
		t.Error("IsValid() should fail when connection is closed")
	}
}

func TestSQLite_IsValid_ExpiredTokenManipulated(t *testing.T) {
	os.Setenv("PASSWORD_PEPPER", "test-pepper")
	db, err := database.NewSQLiteForTest("isvalid_expired_manip")
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	defer func() {
		db.Conn.Close()
		database.CleanupTestDB("isvalid_expired_manip")
	}()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password", ports.RoleUser)

	token, err := db.GetUserToken(ctx, "testuser", "password")
	if err != nil {
		t.Fatalf("required action failed %v", err)
	}

	_, err = db.Conn.Exec(`UPDATE users SET created_at = datetime('now', '-2 hours') WHERE username = ?`, "testuser")
	if err != nil {
		t.Fatalf("required action failed %v", err)
	}

	_, err = db.IsValid(ctx, token.Cookie.Value)
	if err == nil {
		t.Error("IsValid() should fail for expired token")
	}
	if err.Error() != "token expired" {
		t.Errorf("expected 'token expired' error, got: %v", err)
	}
}

func TestSQLite_GetAll_ClosedConnection(t *testing.T) {
	os.Setenv("PASSWORD_PEPPER", "test-pepper")
	db, err := database.NewSQLiteForTest("getall_closed")
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	defer func() {
		db.Conn.Close()
		database.CleanupTestDB("getall_closed")
	}()

	db.Conn.Close()

	ctx := context.Background()
	_, err = db.GetAll(ctx)
	if err == nil {
		t.Error("GetAll() should fail when connection is closed")
	}
}

func TestSQLite_CreateProject_ClosedConnection(t *testing.T) {
	os.Setenv("PASSWORD_PEPPER", "test-pepper")
	db, err := database.NewSQLiteForTest("create_proj_closed")
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	defer func() {
		db.Conn.Close()
		database.CleanupTestDB("create_proj_closed")
	}()

	db.Conn.Close()

	ctx := context.Background()
	_, err = db.CreateProject(ctx, "Project", "owner")
	if err == nil {
		t.Error("CreateProject() should fail when connection is closed")
	}
}

func TestSQLite_CountAdmins_ClosedConnection(t *testing.T) {
	os.Setenv("PASSWORD_PEPPER", "test-pepper")
	db, err := database.NewSQLiteForTest("count_admins_closed")
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	defer func() {
		db.Conn.Close()
		database.CleanupTestDB("count_admins_closed")
	}()

	db.Conn.Close()

	ctx := context.Background()
	_, err = db.CountAdmins(ctx)
	if err == nil {
		t.Error("CountAdmins() should fail when connection is closed")
	}
}

func TestSQLite_CountUsers_ClosedConnection(t *testing.T) {
	os.Setenv("PASSWORD_PEPPER", "test-pepper")
	db, err := database.NewSQLiteForTest("count_users_closed")
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	defer func() {
		db.Conn.Close()
		database.CleanupTestDB("count_users_closed")
	}()

	db.Conn.Close()

	ctx := context.Background()
	_, err = db.CountUsers(ctx)
	if err == nil {
		t.Error("CountUsers() should fail when connection is closed")
	}
}

func TestSQLite_GetUserToken_DBError(t *testing.T) {
	os.Setenv("PASSWORD_PEPPER", "test-pepper")
	db, err := database.NewSQLiteForTest("gettoken_dberror")
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	defer func() {
		db.Conn.Close()
		database.CleanupTestDB("gettoken_dberror")
	}()

	db.Conn.Close()

	ctx := context.Background()
	_, err = db.GetUserToken(ctx, "testuser", "password")
	if err == nil {
		t.Error("GetUserToken() should fail when connection is closed")
	}
}

func TestSQLite_GetAdmin_ClosedConnection(t *testing.T) {
	os.Setenv("PASSWORD_PEPPER", "test-pepper")
	db, err := database.NewSQLiteForTest("getadmin_closed")
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	defer func() {
		db.Conn.Close()
		database.CleanupTestDB("getadmin_closed")
	}()

	db.Conn.Close()

	ctx := context.Background()
	_, err = db.GetAdmin(ctx)
	if err == nil {
		t.Error("GetAdmin() should fail when connection is closed")
	}
}

func TestSQLite_RegisterFirstAdmin_DBError(t *testing.T) {
	os.Setenv("PASSWORD_PEPPER", "test-pepper")
	db, err := database.NewSQLiteForTest("register_first_db_error")
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	defer func() {
		db.Conn.Close()
		database.CleanupTestDB("register_first_db_error")
	}()

	db.Conn.Close()

	ctx := context.Background()
	_, err = db.GetUserToken(ctx, "firstadmin", "password")
	if err == nil {
		t.Error("should fail when connection is closed")
	}
}

func TestSQLite_ChangePassword_UserNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	err := db.ChangePassword(ctx, "nonexistent", "newpass")
	if err == nil {
		t.Error("ChangePassword() should fail for nonexistent user")
	}
	if err != ports.UserNotFound {
		t.Errorf("expected UserNotFound error, got: %v", err)
	}
}

func TestSQLite_GetProjectsByOwner_ScanError(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.CreateProject(ctx, "Project1", "owner")

	db.Conn.Close()

	_, err := db.GetProjectsByOwner(ctx, "owner")
	if err == nil {
		t.Error("GetProjectsByOwner() should fail when connection is closed")
	}
}

func TestSQLite_GetUserToken_QueryError(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password", ports.RoleUser)

	db.Conn.Close()

	_, err := db.GetUserToken(ctx, "testuser", "password")
	if err == nil {
		t.Error("GetUserToken() should fail when connection is closed")
	}
}

func TestSQLite_GetUserToken_UpdateError(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password", ports.RoleUser)

	rows, _ := db.Conn.Query(`SELECT id, password_hash, role FROM users WHERE username = ?`, "testuser")
	_ = rows
	db.Conn.Close()

	_, err := db.GetUserToken(ctx, "testuser", "password")
	if err == nil {
		t.Error("GetUserToken() should fail when connection is closed")
	}
}

func TestSQLite_Create_StmtError(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password", ports.RoleUser)

	db.Conn.Close()

	err := db.Create(ctx, "newuser", "password", ports.RoleUser)
	if err == nil {
		t.Error("Create() should fail when connection is closed")
	}
}

func TestSQLite_ChangePassword_ExecError(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password", ports.RoleUser)

	db.Conn.Close()

	err := db.ChangePassword(ctx, "testuser", "newpass")
	if err == nil {
		t.Error("ChangePassword() should fail when connection is closed")
	}
}

func TestSQLite_RegisterFirstAdmin_BcryptError(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.Conn.Close()

	ctx := context.Background()
	_, err := db.GetUserToken(ctx, "firstadmin", "password")
	if err == nil {
		t.Error("should fail when connection is closed")
	}
}

func TestSQLite_RegisterFirstAdmin_ExecError(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.Conn.Close()

	ctx := context.Background()
	_, err := db.GetUserToken(ctx, "firstadmin", "password")
	if err == nil {
		t.Error("should fail when connection is closed")
	}
}

func TestSQLite_GetAll_ScanError(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Create(ctx, "testuser", "password", ports.RoleUser)

	db.Conn.Close()

	_, err := db.GetAll(ctx)
	if err == nil {
		t.Error("GetAll() should fail when connection is closed")
	}
}

func TestSQLite_CreateProject_PrepareError(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.Conn.Close()

	ctx := context.Background()
	_, err := db.CreateProject(ctx, "Project", "owner")
	if err == nil {
		t.Error("CreateProject() should fail when connection is closed")
	}
}

func TestSQLite_CreateProject_ExecError(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	stmt, _ := db.Conn.Prepare(`INSERT INTO projects (id, name, owner, created_at) VALUES (?, ?, ?, ?)`)
	stmt.Close()
	db.Conn.Close()

	_, err := db.CreateProject(ctx, "Project", "owner")
	if err == nil {
		t.Error("CreateProject() should fail when connection is closed")
	}
}

func TestSQLite_newSQLiteWithFile_BadPath(t *testing.T) {
	_, err := database.NewSQLiteForTest("/invalid/path/test.db")
	if err == nil {
		t.Error("should fail for invalid path")
	}
}
