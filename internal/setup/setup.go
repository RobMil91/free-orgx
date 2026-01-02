package setup

import (
	"github.com/RobMil91/free-orgx/config"
	"github.com/RobMil91/free-orgx/internal/adapters/database"
	"github.com/RobMil91/free-orgx/internal/ports"
)

type Adapters struct {
	UserRep    ports.UserRepo
	ProjectRep ports.ProjectRepo
}

func Setup(cfg config.Config) (*Adapters, error) {

	db, err := database.NewSQLite()
	if err != nil {
		return nil, err
	}

	return &Adapters{
		UserRep:    db,
		ProjectRep: db,
	}, nil
}
