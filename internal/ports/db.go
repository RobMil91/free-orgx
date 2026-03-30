package ports

import (
	"context"
	"errors"
	"time"

	"github.com/RobMil91/free-orgx/internal/models"
)

var (
	UserNotFound = errors.New("[no such user]")
)

type User struct {
	ID       int
	Name     string
	Role     string
	Password string
	Cookie   SessionCookie
}

type SessionCookie struct {
	Value      string
	CreateTime time.Time
}

type UserRepo interface {
	// GetUserToken requests the user for being in the database and returns a token, if he exists
	// on error UserNotFound means no such user in the db GetUserToken(ctx context.Context, name, password string) (string, error)
	GetUserToken(ctx context.Context, name, password string) (*SessionCookie, error)
	IsValid(ctx context.Context, c string) (*User, error)
	Create(ctx context.Context, name, password string) error

	//Use Cookie to authorize for delete etc action
	Delete(ctx context.Context, name string) error
	GetAll(ctx context.Context) ([]User, error)
	ChangePassword(ctx context.Context, name, newPassword string) error
	Logout(ctx context.Context, name string) error
}

type ProjectRepo interface {
	CreateProject(ctx context.Context, name, owner string) (models.Project, error)
	GetProjectsByOwner(ctx context.Context, owner string) ([]models.Project, error)
	DeleteProject(ctx context.Context, id string) error
}

var DatabaseError = errors.New("database adapter failed to execute action")
