package database

import (
	"context"

	"github.com/RobMil91/free-orgx/internal/ports"
)

// LoadSnapshot implements [ports.TasksRepo].
func (s *SQLiteAdapter) LoadSnapshot(ctx context.Context, project_id string) ([]ports.Task, error) {
	panic("unimplemented")
}
