package database

import (
	"context"

	"github.com/RobMil91/free-orgx/internal/ports"
)

func (s *SQLiteAdapter) LoadSnapshot(ctx context.Context, projectID string) ([]ports.Task, error) {
	rows, err := s.Conn.Query(`
		SELECT id, name, description
		FROM tasks
		WHERE project_id = ?
		ORDER BY created_at ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []ports.Task
	for rows.Next() {
		var t ports.Task
		if err := rows.Scan(&t.ID, &t.Name, &t.Description); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}

	if tasks == nil {
		tasks = []ports.Task{}
	}

	return tasks, nil
}
