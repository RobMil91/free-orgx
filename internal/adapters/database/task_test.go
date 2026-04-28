package database_test

import (
	"context"
	"testing"

	"github.com/RobMil91/free-orgx/internal/adapters/database"
	"github.com/RobMil91/free-orgx/internal/ports"
)

var _ ports.TasksRepo = (*database.SQLiteAdapter)(nil)

func TestSQLite_LoadSnapshot(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	project, _ := db.CreateProject(ctx, "Test Project", "testuser")

	_, err := db.Conn.Exec(`INSERT INTO tasks (id, name, description, project_id) VALUES (?, ?, ?, ?)`,
		"task1", "Task One", "Description One", project.ID)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}

	_, err = db.Conn.Exec(`INSERT INTO tasks (id, name, description, project_id) VALUES (?, ?, ?, ?)`,
		"task2", "Task Two", "Description Two", project.ID)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}

	tasks, err := db.LoadSnapshot(ctx, project.ID)
	if err != nil {
		t.Fatalf("LoadSnapshot() failed: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestSQLite_LoadSnapshot_Empty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	project, _ := db.CreateProject(ctx, "Empty Project", "testuser")

	tasks, err := db.LoadSnapshot(ctx, project.ID)
	if err != nil {
		t.Fatalf("LoadSnapshot() failed: %v", err)
	}

	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestSQLite_LoadSnapshot_WrongProjectID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	tasks, err := db.LoadSnapshot(ctx, "nonexistent-project-id")
	if err != nil {
		t.Fatalf("LoadSnapshot() failed: %v", err)
	}

	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks for nonexistent project, got %d", len(tasks))
	}
}

func TestSQLite_LoadSnapshot_ClosedConnection(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.Conn.Close()

	ctx := context.Background()
	_, err := db.LoadSnapshot(ctx, "some-project-id")
	if err == nil {
		t.Error("LoadSnapshot() should fail when connection is closed")
	}
}