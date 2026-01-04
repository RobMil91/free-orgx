package setup

import (
	"log/slog"

	"github.com/RobMil91/free-orgx/config"
	"github.com/RobMil91/free-orgx/internal/adapters/database"
	"github.com/RobMil91/free-orgx/internal/adapters/mocks"
	"github.com/RobMil91/free-orgx/internal/ports"
)

type Adapters struct {
	UserRep    ports.UserRepo
	ProjectRep ports.ProjectRepo
}

func Setup(cfg config.Config) (*Adapters, error) {
	var (
		users    ports.UserRepo
		projects ports.ProjectRepo
	)

	if cfg.RAMDB {
		ramDB := mocks.RAM{
			Users: map[string]ports.User{
				"tom": {
					Name:     "tom",
					Password: "test",
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
		slog.Info("started sqlite db")
	}

	return &Adapters{
		UserRep:    users,
		ProjectRep: projects,
	}, nil
}
