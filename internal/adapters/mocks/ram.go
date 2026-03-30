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
	Users    map[string]ports.User
	Projects map[string][]models.Project
}

// GetAll implements [ports.UserRepo].
func (r *RAM) GetAll(ctx context.Context) ([]ports.User, error) {
	panic("unimplemented")
}

// ChangePassword implements [ports.UserRepo].
func (r *RAM) ChangePassword(ctx context.Context, name string, newPassword string) error {
	panic("unimplemented")
}

// Create implements [ports.UserRepo].
func (r *RAM) Create(ctx context.Context, name, password, role string) error {
	if r.Users == nil {
		r.Users = make(map[string]ports.User)
	}
	r.Users[name] = ports.User{
		Name:     name,
		Password: password,
		Role:     role,
	}
	return nil
}

// Delete implements [ports.UserRepo].
func (r *RAM) Delete(ctx context.Context, name string) error {
	delete(r.Users, name)
	return nil
}

// Logout implements [ports.UserRepo].
func (r *RAM) Logout(ctx context.Context, name string) error {
	if u, ok := r.Users[name]; ok {
		u.Cookie.Value = ""
		r.Users[name] = u
	}
	return nil
}

// GetAdmin implements [ports.UserRepo].
func (r *RAM) GetAdmin(ctx context.Context) (*ports.User, error) {
	for _, u := range r.Users {
		if u.Role == ports.RoleAdmin {
			return &u, nil
		}
	}
	return nil, ports.UserNotFound
}

// CountAdmins implements [ports.UserRepo].
func (r *RAM) CountAdmins(ctx context.Context) (int, error) {
	count := 0
	for _, u := range r.Users {
		if u.Role == ports.RoleAdmin {
			count++
		}
	}
	return count, nil
}

// CreateProject implements [ports.ProjectRepo].
func (r *RAM) CreateProject(ctx context.Context, name, owner string) (models.Project, error) {
	id, _ := CreateRandStr(16)
	project := models.Project{
		ID:      *id,
		Name:    name,
		Owner:   owner,
		Created: time.Now().Format(time.RFC3339),
	}
	if r.Projects == nil {
		r.Projects = make(map[string][]models.Project)
	}
	r.Projects[owner] = append(r.Projects[owner], project)
	return project, nil
}

// GetProjectsByOwner implements [ports.ProjectRepo].
func (r *RAM) GetProjectsByOwner(ctx context.Context, owner string) ([]models.Project, error) {
	if r.Projects == nil {
		return []models.Project{}, nil
	}
	projects, ok := r.Projects[owner]
	if !ok {
		return []models.Project{}, nil
	}
	return projects, nil
}

// DeleteProject implements [ports.ProjectRepo].
func (r *RAM) DeleteProject(ctx context.Context, id string) error {
	panic("unimplemented")
}

func (r *RAM) GetUserToken(ctx context.Context, name, password string) (*ports.LoginResult, error) {
	if r.Users == nil {
		r.Users = make(map[string]ports.User)
	}

	u, ok := r.Users[name]

	if !ok {
		count, _ := r.CountUsers(ctx)
		if count == 0 {
			return r.registerFirstAdmin(ctx, name, password)
		}
		return nil, fmt.Errorf("%w", ports.UserNotFound)
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

	return &ports.LoginResult{
		Cookie:    &u.Cookie,
		User:      &u,
		IsNewUser: false,
	}, nil
}

func (r *RAM) registerFirstAdmin(ctx context.Context, name, password string) (*ports.LoginResult, error) {
	newCookie, _ := CreateRandStr(32)
	user := ports.User{
		ID:       len(r.Users) + 1,
		Name:     name,
		Password: password,
		Role:     ports.RoleAdmin,
		Cookie: ports.SessionCookie{
			Value:      *newCookie,
			CreateTime: time.Now(),
		},
	}
	r.Users[name] = user

	return &ports.LoginResult{
		Cookie:    &user.Cookie,
		User:      &user,
		IsNewUser: true,
	}, nil
}

// CountUsers implements [ports.UserRepo].
func (r *RAM) CountUsers(ctx context.Context) (int, error) {
	if r.Users == nil {
		return 0, nil
	}
	return len(r.Users), nil
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
