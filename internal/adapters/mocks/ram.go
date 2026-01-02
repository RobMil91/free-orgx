package mocks

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/RobMil91/free-orgx/internal/models"
	"github.com/RobMil91/free-orgx/internal/ports"
)

var _ ports.UserRepo = (*RAM)(nil)
var _ ports.ProjectRepo = (*RAM)(nil)

const (
	// CookieLiveTime = time.Minute * 10
	CookieLiveTime = time.Minute * 1
)

type RAM struct {
	Users map[string]ports.User
}

// ChangePassword implements [ports.UserRepo].
func (r *RAM) ChangePassword(ctx context.Context, name string, newPassword string) error {
	panic("unimplemented")
}

// Create implements [ports.UserRepo].
func (r *RAM) Create(ctx context.Context, name string, password string) error {
	panic("unimplemented")
}

// Delete implements [ports.UserRepo].
func (r *RAM) Delete(ctx context.Context, name string) error {
	panic("unimplemented")
}

// Logout implements [ports.UserRepo].
func (r *RAM) Logout(ctx context.Context, name string) error {
	panic("unimplemented")
}

// CreateProject implements [ports.ProjectRepo].
func (r *RAM) CreateProject(ctx context.Context, name string) (models.Project, error) {
	panic("unimplemented")
}

// DeleteProject implements [ports.ProjectRepo].
func (r *RAM) DeleteProject(ctx context.Context, id string) error {
	panic("unimplemented")
}

func (r *RAM) GetUserToken(ctx context.Context, name, password string) (*ports.SessionCookie, error) {
	u, ok := r.Users[name]

	if !ok {
		return nil, fmt.Errorf("%s %w", name, ports.UserNotFound)
	}

	if u.Password != password {
		return nil, errors.New("bad password")
	}

	newCookie, err := CreateRandStr(32)
	if err != nil {
		return nil, fmt.Errorf("could not create cookie %w", err)
	}

	u.Cookie.CreateTime = time.Now()
	u.Cookie.Value = *newCookie
	r.Users[name] = u

	return &u.Cookie, nil
}

// this should not be adapter logic refacotor to use in controller
func (r *RAM) IsValid(ctx context.Context, cookie string) (*ports.User, error) {
	for _, u := range r.Users {

		expirationTime := u.Cookie.CreateTime.Add(CookieLiveTime)
		if u.Cookie.Value == cookie && time.Now().Before(expirationTime) {
			return &u, nil
		}
	}

	return nil, errors.New("no user with this active token")
}

// TODO: this probably bad random access, would need refactor for sufisticated solution. Do not use with sensitive data.
func CreateRandStr(length int) (*string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	randStr := base64.URLEncoding.EncodeToString(b)
	return &randStr, nil
}
