package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"

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
	createTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		token TEXT,
		salt TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := s.Conn.Exec(createTable)
	if err != nil {
		return err
	}

	return nil
}

// ChangePassword implements [ports.UserRepo].
func (s *SQLiteAdapter) ChangePassword(ctx context.Context, name string, newPassword string) error {
	panic("unimplemented")
}

// Create implements [ports.UserRepo].
func (s *SQLiteAdapter) Create(ctx context.Context, name string, password string) error {
	pepper := os.Getenv("PASSWORD_PEPPER")
	if pepper == "" {
		slog.Error(fmt.Errorf("could not create new user %s, because their is no pepper set it env", name).Error())
		return ports.CreateError
	}

	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		slog.Error(fmt.Errorf("create user %s failed to generate random salt, [%w]", name, err).Error())
		return ports.CreateError
	}
	saltStr := base64.RawStdEncoding.EncodeToString(salt)

	combined := password + saltStr + pepper

	hash, err := bcrypt.GenerateFromPassword([]byte(combined), bcrypt.DefaultCost)
	if err != nil {
		slog.Error(fmt.Errorf("could not create user %s because of hash fail %w", name, err).Error())
		return ports.CreateError
	}

	stmt, err := s.Conn.Prepare(`
    INSERT INTO users (username, password_hash, token, salt)
    VALUES (?, ?, ?, ?)
`)
	if err != nil {
		slog.Error(err.Error())
		return ports.CreateError
	}

	defer stmt.Close()

	_, err = stmt.Exec(name, hash, "", saltStr)
	if err != nil {
		slog.Error(err.Error())
		return ports.CreateError
	}

	return nil
}

// Delete implements [ports.UserRepo].
func (s *SQLiteAdapter) Delete(ctx context.Context, name string) error {
	panic("unimplemented")
}

// Logout implements [ports.UserRepo].
func (s *SQLiteAdapter) Logout(ctx context.Context, name string) error {
	panic("unimplemented")
}

// GetUserToken implements [ports.UserRepo].
func (s *SQLiteAdapter) GetUserToken(ctx context.Context, name string, password string) (*ports.SessionCookie, error) {
	panic("unimplemented")
}

// IsValid implements [ports.UserRepo].
func (s *SQLiteAdapter) IsValid(ctx context.Context, c string) (*ports.User, error) {
	panic("unimplemented")
}

// CreateProject implements [ports.ProjectRepo].
func (s *SQLiteAdapter) CreateProject(ctx context.Context, name string) (models.Project, error) {
	panic("unimplemented")
}

// DeleteProject implements [ports.ProjectRepo].
func (s *SQLiteAdapter) DeleteProject(ctx context.Context, id string) error {
	panic("unimplemented")
}
