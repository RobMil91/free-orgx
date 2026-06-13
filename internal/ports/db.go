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
	GetAllProjects(ctx context.Context) ([]models.Project, error)
	GetProject(ctx context.Context, id string) (*models.Project, error)
}

const (
	CreateTask = "create"
	UpdateTask = "update"

	EditEvent = "edit-event"

	Todo       = "todo"
	InProgress = "in progress"
	Done       = "done"
	Achieved   = "achieved"
)

type EventType struct {
	T string `json:"event-type"`
}

type TaskID struct {
	ID string `json:"taskID"`
}

type TasksRepo interface {
	LoadSnapshot(ctx context.Context, project_id string) ([]models.Task, error)
	CreateSnapshot(ctx context.Context, projectID string, tasks []models.Task) error
}

type EventStore interface {
	NewEvent(ctx context.Context, project_id string, t models.TaskEventRequest) (*models.TaskEvent, error)
	GetEvents(ctx context.Context, project_id string) ([]models.TaskEvent, error)
}

var DatabaseError = errors.New("database adapter failed to execute action")
