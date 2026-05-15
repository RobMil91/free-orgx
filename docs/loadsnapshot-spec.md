# LoadSnapshot - SQLite Adapter

## Overview
`LoadSnapshot` retrieves all tasks associated with a given project ID from the SQLite database.

## Function Signature
```go
func (s *SQLiteAdapter) LoadSnapshot(ctx context.Context, projectID string) ([]ports.Task, error)
```

## Parameters
- `ctx`: Context for cancellation and timeout
- `projectID`: The unique identifier of the project

## Returns
- `[]ports.Task`: A slice of Task structs ordered by creation time (ascending)
- if no tasks are in db return empty slice
- `error`: Any database error encountered

## Database Schema
Tasks are stored in the `tasks` table:
```sql
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id)
);
```

## Implementation Notes
- Returns an empty slice if no tasks exist for the project
- Orders tasks by `created_at` in ascending order
- Implements `ports.TasksRepo` interface
