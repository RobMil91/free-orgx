package setup

import (
	"log/slog"

	"github.com/RobMil91/free-orgx/config"
	"github.com/RobMil91/free-orgx/internal/adapters/database"
	"github.com/RobMil91/free-orgx/internal/adapters/mocks"
	"github.com/RobMil91/free-orgx/internal/core/tasks"
	"github.com/RobMil91/free-orgx/internal/ports"
)

type Adapters struct {
	UserRep          ports.UserRepo
	ProjectRep       ports.ProjectRepo
	TaskRepo         ports.TasksRepo
	EventRepo        ports.EventStore
	EventsTranslator ports.EventTranslator
}

func Setup(cfg config.Config) (*Adapters, error) {
	var (
		users    ports.UserRepo
		projects ports.ProjectRepo

		taskRepo     ports.TasksRepo
		eventRepo    ports.EventStore
		evTranslator ports.EventTranslator
	)

	if cfg.RAMDB {
		ramDB := mocks.RAM{
			Users: map[string]ports.User{
				"testadmin": {
					Name:     "testadmin",
					Password: "testpass",
					Role:     ports.RoleAdmin,
				},
			},
		}

		users = &ramDB
		projects = &ramDB

		slog.Info("started ram db")
	} else {
		db, err := database.NewSQLite()
		if err != nil {
			return nil, err
		}

		users = db
		projects = db
		taskRepo = db
		eventRepo = db

		evTranslator = tasks.NewTaskBoard(db, cfg.Files)
		// evTranslator = tasks.NewTaskBoard(db, "../../static/")

		slog.Info("started sqlite db (first login becomes admin)")
	}

	//new translator

	return &Adapters{
		UserRep:          users,
		ProjectRep:       projects,
		TaskRepo:         taskRepo,
		EventRepo:        eventRepo,
		EventsTranslator: evTranslator,
	}, nil
}
