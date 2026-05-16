package ports

import (
	"context"
	"errors"
	"time"

	"github.com/RobMil91/free-orgx/internal/models"
)

var (
	UserNotFound = errors.New("[no such user]")
	AdminExists  = errors.New("[admin already exists]")
	InvalidRole  = errors.New("[invalid role]")
)

type LoginResult struct {
	Cookie    *SessionCookie
	User      *User
	IsNewUser bool
}

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
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
	GetUserToken(ctx context.Context, name, password string) (*LoginResult, error)
	IsValid(ctx context.Context, c string) (*User, error)
	Create(ctx context.Context, name, password, role string) error
	Delete(ctx context.Context, name string) error
	GetAll(ctx context.Context) ([]User, error)
	ChangePassword(ctx context.Context, name, newPassword string) error
	Logout(ctx context.Context, name string) error
	GetAdmin(ctx context.Context) (*User, error)
	CountAdmins(ctx context.Context) (int, error)
	CountUsers(ctx context.Context) (int, error)
}

type ProjectRepo interface {
	CreateProject(ctx context.Context, name, owner string) (models.Project, error)
	GetProjectsByOwner(ctx context.Context, owner string) ([]models.Project, error)
	DeleteProject(ctx context.Context, id string) error
}

const (
	CreateTask = "create"
	UpdateTask = "update"

	Todo       = "todo"
	InProgress = "in progress"
	Done       = "done"
	Achieved   = "achieved"
)

type TaskEventRequest struct {
	Type string `json:"event-type"`
	User string
	NewTask
}

type TaskEvent struct {
	TaskEventRequest
	EventTime time.Time
}

type TasksRepo interface {
	LoadSnapshot(ctx context.Context, project_id string) ([]Task, error)
	CreateSnapshot(ctx context.Context, projectID string, tasks []Task) error
}

type EventStore interface {
	NewEvent(ctx context.Context, project_id string, t TaskEventRequest) error
	GetEvents(ctx context.Context, project_id string) ([]TaskEvent, error)
}

type NewTask struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Deadline    string `json:"deadline"`
	Status      string `json:"status"`
}

type Task struct {
	NewTask
	ID string `json:"id"`

	// Messages []string
}

var DatabaseError = errors.New("database adapter failed to execute action")
