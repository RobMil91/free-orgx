package database

import (
	"context"

	"github.com/RobMil91/free-orgx/internal/models"
)

func (s *SQLiteAdapter) LoadSnapshot(ctx context.Context, projectID string) ([]models.Task, error) {
	rows, err := s.Conn.Query(`
		SELECT id, name, description
		FROM tasks
		WHERE project_id = ?
		ORDER BY created_at ASC`, projectID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.TaskID, &t.Title, &t.Description); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}

	return tasks, nil
}

func (s *SQLiteAdapter) CreateSnapshot(ctx context.Context, projectID string, tasks []models.Task) error {
	tx, err := s.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO tasks (id, name, description, project_id) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, t := range tasks {
		_, err = stmt.Exec(t.TaskID, t.Title, t.Description, projectID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
