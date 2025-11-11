package ports

import (
	"context"
	"errors"
	"time"
)

var (
	UserNotFound = errors.New("[no such user]")
)

type User struct {
	Name     string
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
}
