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

type SQLiteAdapter struct {
	Conn *sql.DB
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
	db, err := sql.Open("sqlite3", "orgxdb.db")
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
func (s *SQLiteAdapter) Create(ctx context.Context, name string, password string) error {
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

	_, err = stmt.Exec(name, hash, "", "user")
	if err != nil {
		slog.Error(err.Error())
		return ports.DatabaseError
	}

	return nil
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
func (s *SQLiteAdapter) GetUserToken(ctx context.Context, name string, password string) (*ports.SessionCookie, error) {
	pepper := os.Getenv("PASSWORD_PEPPER")
	if pepper == "" {
		slog.Error("PASSWORD_PEPPER not set in environment")
		return nil, ports.DatabaseError
	}

	var passwordHash string
	query := `SELECT password_hash FROM users WHERE username = ?`
	if err := s.Conn.QueryRow(query, name).Scan(&passwordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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

	return &ports.SessionCookie{
		Value:      *token,
		CreateTime: time.Now(),
	}, nil
}

// IsValid implements [ports.UserRepo].
func (s *SQLiteAdapter) IsValid(ctx context.Context, c string) (*ports.User, error) {
	var user ports.User
	var createTime time.Time
	query := `SELECT id, username, role, created_at FROM users WHERE token = ?`
	err := s.Conn.QueryRow(query, c).Scan(&user.ID, &user.Name, &user.Role, &createTime)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("no user with this active token")
		}
		slog.Error(fmt.Sprintf("could not validate token: %v", err))
		return nil, ports.DatabaseError
	}

	expirationTime := createTime.Add(time.Minute * 60)
	if time.Now().After(expirationTime) {
		return nil, errors.New("token expired")
	}

	return &user, nil
}

// CreateProject implements [ports.ProjectRepo].
func (s *SQLiteAdapter) CreateProject(ctx context.Context, name string) (models.Project, error) {
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

	_, err = stmt.Exec(*id, name, "", created)
	if err != nil {
		slog.Error(fmt.Sprintf("could not create project: %v", err))
		return models.Project{}, ports.DatabaseError
	}

	return models.Project{
		ID:      *id,
		Name:    name,
		Owner:   "",
		Created: created,
	}, nil
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
