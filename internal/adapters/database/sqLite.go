package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/RobMil91/free-orgx/internal/models"
	"github.com/RobMil91/free-orgx/internal/ports"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var _ ports.UserRepo = (*SQLiteAdapter)(nil)
var _ ports.ProjectRepo = (*SQLiteAdapter)(nil)
var _ ports.TasksRepo = (*SQLiteAdapter)(nil)
var _ ports.EventStore = (*SQLiteAdapter)(nil)

type SQLiteAdapter struct {
	Conn *sql.DB
}

// GetEvents implements [ports.EventStore].
func (s *SQLiteAdapter) GetEvents(ctx context.Context, project_id string) ([]ports.TaskEvent, error) {
	panic("unimplemented")
}

// NewEvent implements [ports.EventStore].
func (s *SQLiteAdapter) NewEvent(
	ctx context.Context,
	project_id string,
	t ports.TaskEventRequest) error {

	id, err := createRandStr(16)
	if err != nil {
		slog.Error(fmt.Sprintf("could not generate project id: %v", err))
		return ports.DatabaseError
	}

	created := time.Now().Format(time.RFC3339)

	stmt, err := s.Conn.Prepare(
		`INSERT INTO task_events (id,event_type,created_at,deadline,user,description,status,project_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		slog.Error(fmt.Sprintf("could not prepare project insert: %v", err))
		return ports.DatabaseError
	}
	defer stmt.Close()

	_, err = stmt.Exec(*id, t.Type, created, t.Deadline, t.User, t.Description, t.Status, project_id)
	if err != nil {
		slog.Error(fmt.Sprintf("could not create project: %v", err))
		return ports.DatabaseError
	}

	return nil
}

// GetAll implements [ports.UserRepo].
func (s *SQLiteAdapter) GetAll(ctx context.Context) ([]ports.User, error) {
	rows, err := s.Conn.Query(`SELECT id, username, role FROM users;`)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var users []ports.User

	for rows.Next() {
		var u ports.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Role); err != nil {
			return nil, err
		}

		users = append(users, u)
	}

	return users, nil
}

func NewSQLite() (*SQLiteAdapter, error) {
	return newSQLiteWithFile("orgxdb.db")
}

func NewSQLiteForTest(name string) (*SQLiteAdapter, error) {
	return newSQLiteWithFile(fmt.Sprintf("test_%s.db", name))
}

func newSQLiteWithFile(filename string) (*SQLiteAdapter, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}

	adapter := &SQLiteAdapter{
		Conn: db,
	}

	err = adapter.CreateTables()
	if err != nil {
		return nil, err
	}

	return adapter, nil
}

func (s *SQLiteAdapter) CreateTables() error {
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		token TEXT,
		role TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := s.Conn.Exec(createUsersTable)
	if err != nil {
		return err
	}

	createProjectsTable := `
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		owner TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = s.Conn.Exec(createProjectsTable)
	if err != nil {
		return err
	}

	createTasksTable := `
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL,
		project_id TEXT NOT NULL,

		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		deadline DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);`

	_, err = s.Conn.Exec(createTasksTable)
	if err != nil {
		return err
	}

	eventTable := `
	CREATE TABLE IF NOT EXISTS task_events (
		id TEXT PRIMARY KEY,
		event_type TEXT NOT NULL,

		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		deadline DATETIME NOT NULL,

		user TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL,
		project_id TEXT NOT NULL,

		FOREIGN KEY (project_id) REFERENCES projects(id)
	);`

	_, err = s.Conn.Exec(eventTable)
	if err != nil {
		return err
	}

	return nil
}

// ChangePassword implements [ports.UserRepo].
func (s *SQLiteAdapter) ChangePassword(ctx context.Context, name string, newPassword string) error {
	pepper := os.Getenv("PASSWORD_PEPPER")
	if pepper == "" {
		slog.Error("PASSWORD_PEPPER not set in environment")
		return ports.DatabaseError
	}

	combined := newPassword + pepper
	hash, err := bcrypt.GenerateFromPassword([]byte(combined), bcrypt.DefaultCost)
	if err != nil {
		slog.Error(fmt.Sprintf("could not hash new password for user %s: %v", name, err))
		return ports.DatabaseError
	}

	result, err := s.Conn.Exec(`UPDATE users SET password_hash = ? WHERE username = ?`, hash, name)
	if err != nil {
		slog.Error(fmt.Sprintf("could not update password for user %s: %v", name, err))
		return ports.DatabaseError
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ports.UserNotFound
	}

	return nil
}

// Create implements [ports.UserRepo].
func (s *SQLiteAdapter) Create(ctx context.Context, name string, password string, role string) error {
	if role != ports.RoleAdmin && role != ports.RoleUser {
		return ports.InvalidRole
	}

	if role == ports.RoleAdmin {
		count, err := s.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if count > 0 {
			return ports.AdminExists
		}
	}

	pepper := os.Getenv("PASSWORD_PEPPER")
	if pepper == "" {
		slog.Error(fmt.Errorf("could not create new user %s, because their is no pepper set it env", name).Error())
		return ports.DatabaseError
	}

	combined := password + pepper

	hash, err := bcrypt.GenerateFromPassword([]byte(combined), bcrypt.DefaultCost)
	if err != nil {
		slog.Error(fmt.Errorf("could not create user %s because of hash fail %w", name, err).Error())
		return ports.DatabaseError
	}

	stmt, err := s.Conn.Prepare(`
    INSERT INTO users (username, password_hash, token, role)
    VALUES (?, ?, ?, ?)
`)
	if err != nil {
		slog.Error(err.Error())
		return ports.DatabaseError
	}

	defer stmt.Close()

	_, err = stmt.Exec(name, hash, "", role)
	if err != nil {
		slog.Error(err.Error())
		return ports.DatabaseError
	}

	return nil
}

// GetAdmin implements [ports.UserRepo].
func (s *SQLiteAdapter) GetAdmin(ctx context.Context) (*ports.User, error) {
	var user ports.User
	err := s.Conn.QueryRow(`SELECT id, username, role FROM users WHERE role = ?`, ports.RoleAdmin).Scan(&user.ID, &user.Name, &user.Role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ports.UserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// CountAdmins implements [ports.UserRepo].
func (s *SQLiteAdapter) CountAdmins(ctx context.Context) (int, error) {
	var count int
	err := s.Conn.QueryRow(`SELECT COUNT(*) FROM users WHERE role = ?`, ports.RoleAdmin).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Delete implements [ports.UserRepo].
func (s *SQLiteAdapter) Delete(ctx context.Context, name string) error {
	_, err := s.Conn.Exec(`DELETE FROM users WHERE username = ?`, name)
	if err != nil {
		slog.Error("fmt.Sprintf(delete failed for %s, because of %s", name, err.Error())
		return ports.DatabaseError
	}

	return nil
}

// Logout implements [ports.UserRepo].
func (s *SQLiteAdapter) Logout(ctx context.Context, name string) error {
	_, err := s.Conn.Exec(`UPDATE users SET token = NULL WHERE username = ?`, name)
	if err != nil {
		slog.Error(fmt.Sprintf("could not logout user %s: %v", name, err))
		return ports.DatabaseError
	}
	return nil
}

// GetUserToken implements [ports.UserRepo].
func (s *SQLiteAdapter) GetUserToken(ctx context.Context, name string, password string) (*ports.LoginResult, error) {
	pepper := os.Getenv("PASSWORD_PEPPER")
	if pepper == "" {
		slog.Error("PASSWORD_PEPPER not set in environment")
		return nil, ports.DatabaseError
	}

	var passwordHash string
	var userID int
	var userRole string
	query := `SELECT id, password_hash, role FROM users WHERE username = ?`
	err := s.Conn.QueryRow(query, name).Scan(&userID, &passwordHash, &userRole)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			userCount, countErr := s.CountUsers(ctx)
			if countErr != nil {
				return nil, countErr
			}
			if userCount == 0 {
				return s.registerFirstAdmin(ctx, name, password, pepper)
			}
			return nil, fmt.Errorf("%w", ports.UserNotFound)
		}
		return nil, fmt.Errorf("can not get user with name %s, [%w]", name, err)
	}

	combined := password + pepper
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(combined)); err != nil {
		return nil, errors.New("bad password")
	}

	token, err := createRandStr(32)
	if err != nil {
		slog.Error(fmt.Sprintf("could not create cookie for user %s: %v", name, err))
		return nil, ports.DatabaseError
	}

	_, err = s.Conn.Exec(`UPDATE users SET token = ? WHERE username = ?`, *token, name)
	if err != nil {
		slog.Error(fmt.Sprintf("could not update token for user %s: %v", name, err))
		return nil, ports.DatabaseError
	}

	now := time.Now()

	_, err = s.Conn.Exec(`UPDATE users SET created_at= ? WHERE username = ?`, now, name)
	if err != nil {
		slog.Error(fmt.Sprintf("could not update create time for user %s: %v", name, err))
		return nil, ports.DatabaseError
	}
	slog.Debug(fmt.Sprintf("user: %s, token created at: %s", name, now.String()))

	return &ports.LoginResult{
		Cookie: &ports.SessionCookie{
			Value:      *token,
			CreateTime: now,
		},
		User: &ports.User{
			ID:   userID,
			Name: name,
			Role: userRole,
		},
		IsNewUser: false,
	}, nil
}

func (s *SQLiteAdapter) registerFirstAdmin(ctx context.Context, name, password, pepper string) (*ports.LoginResult, error) {
	combined := password + pepper
	hash, err := bcrypt.GenerateFromPassword([]byte(combined), bcrypt.DefaultCost)
	if err != nil {
		slog.Error(fmt.Sprintf("could not hash password for first admin %s: %v", name, err))
		return nil, ports.DatabaseError
	}

	token, err := createRandStr(32)
	if err != nil {
		slog.Error(fmt.Sprintf("could not create token for first admin %s: %v", name, err))
		return nil, ports.DatabaseError
	}

	result, err := s.Conn.Exec(
		`INSERT INTO users (username, password_hash, token, role) VALUES (?, ?, ?, ?)`,
		name, hash, *token, ports.RoleAdmin,
	)
	if err != nil {
		slog.Error(fmt.Sprintf("could not create first admin %s: %v", name, err))
		return nil, ports.DatabaseError
	}

	id, _ := result.LastInsertId()

	slog.Info(fmt.Sprintf("first admin registered: %s", name))

	return &ports.LoginResult{
		Cookie: &ports.SessionCookie{
			Value:      *token,
			CreateTime: time.Now(),
		},
		User: &ports.User{
			ID:   int(id),
			Name: name,
			Role: ports.RoleAdmin,
		},
		IsNewUser: true,
	}, nil
}

// CountUsers implements [ports.UserRepo].
func (s *SQLiteAdapter) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := s.Conn.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// IsValid implements [ports.UserRepo].
func (s *SQLiteAdapter) IsValid(ctx context.Context, c string) (*ports.User, error) {
	var user ports.User
	var createTime time.Time
	query := `SELECT id, username, role, created_at FROM users WHERE token = ?`
	err := s.Conn.QueryRow(query, c).Scan(&user.ID, &user.Name, &user.Role, &createTime)
	if err != nil {
		slog.Error(fmt.Sprintf("could not validate token: %s", err.Error()))
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("no user with this active token")
		}

		return nil, ports.DatabaseError
	}

	now := time.Now().UTC()

	expirationTime := createTime.Add(time.Minute * 60)
	if now.After(expirationTime) {
		slog.Error(fmt.Sprintf(
			"token expired experiationTime: %s, time: %s",
			expirationTime.String(),
			now.String()))
		return nil, errors.New("token expired")
	}

	return &user, nil
}

// CreateProject implements [ports.ProjectRepo].
func (s *SQLiteAdapter) CreateProject(ctx context.Context, name, owner string) (models.Project, error) {
	id, err := createRandStr(16)
	if err != nil {
		slog.Error(fmt.Sprintf("could not generate project id: %v", err))
		return models.Project{}, ports.DatabaseError
	}

	created := time.Now().Format(time.RFC3339)

	stmt, err := s.Conn.Prepare(`INSERT INTO projects (id, name, owner, created_at) VALUES (?, ?, ?, ?)`)
	if err != nil {
		slog.Error(fmt.Sprintf("could not prepare project insert: %v", err))
		return models.Project{}, ports.DatabaseError
	}
	defer stmt.Close()

	_, err = stmt.Exec(*id, name, owner, created)
	if err != nil {
		slog.Error(fmt.Sprintf("could not create project: %v", err))
		return models.Project{}, ports.DatabaseError
	}

	return models.Project{
		ID:      *id,
		Name:    name,
		Owner:   owner,
		Created: created,
	}, nil
}

// GetProjectsByOwner implements [ports.ProjectRepo].
func (s *SQLiteAdapter) GetProjectsByOwner(ctx context.Context, owner string) ([]models.Project, error) {
	rows, err := s.Conn.Query(`SELECT id, name, owner, created_at FROM projects WHERE owner = ?`, owner)
	if err != nil {
		slog.Error(fmt.Sprintf("could not query projects: %v", err))
		return nil, ports.DatabaseError
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Owner, &p.Created); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}

	if projects == nil {
		projects = []models.Project{}
	}

	return projects, nil
}

// DeleteProject implements [ports.ProjectRepo].
func (s *SQLiteAdapter) DeleteProject(ctx context.Context, id string) error {
	result, err := s.Conn.Exec(`DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		slog.Error(fmt.Sprintf("could not delete project %s: %v", id, err))
		return ports.DatabaseError
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("project not found")
	}

	return nil
}

func createRandStr(length int) (*string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	randStr := base64.URLEncoding.EncodeToString(b)
	return &randStr, nil
}

func CleanupTestDB(name string) {
	os.Remove(fmt.Sprintf("test_%s.db", name))
}
