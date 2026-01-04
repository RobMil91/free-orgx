package database

import (
	"context"
	"database/sql"

	"github.com/RobMil91/free-orgx/internal/models"
	"github.com/RobMil91/free-orgx/internal/ports"
	_ "github.com/mattn/go-sqlite3"
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

	return &SQLiteAdapter{
		Conn: db,
	}, nil
}

func (s SQLiteAdapter) CreateUserTable() error {

	createTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
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
	panic("unimplemented")
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
