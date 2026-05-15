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

func TestSQLite_CreateSnapshot(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	project, _ := db.CreateProject(ctx, "Test Project", "testuser")

	tasks := []ports.Task{
		{ID: "task1", NewTask: ports.NewTask{Name: "Task One", Description: "Description One"}},
		{ID: "task2", NewTask: ports.NewTask{Name: "Task Two", Description: "Description Two"}},
	}

	err := db.CreateSnapshot(ctx, project.ID, tasks)
	if err != nil {
		t.Fatalf("CreateSnapshot() failed: %v", err)
	}

	loaded, err := db.LoadSnapshot(ctx, project.ID)
	if err != nil {
		t.Fatalf("LoadSnapshot() failed: %v", err)
	}

	if len(loaded) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(loaded))
	}

	if loaded[0].Name != "Task One" {
		t.Errorf("expected first task name 'Task One', got '%s'", loaded[0].Name)
	}
	if loaded[1].Name != "Task Two" {
		t.Errorf("expected second task name 'Task Two', got '%s'", loaded[1].Name)
	}
}

func TestSQLite_CreateSnapshot_Empty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	project, _ := db.CreateProject(ctx, "Test Project", "testuser")

	err := db.CreateSnapshot(ctx, project.ID, []ports.Task{})
	if err != nil {
		t.Fatalf("CreateSnapshot() failed with empty tasks: %v", err)
	}

	loaded, err := db.LoadSnapshot(ctx, project.ID)
	if err != nil {
		t.Fatalf("LoadSnapshot() failed: %v", err)
	}

	if len(loaded) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(loaded))
	}
}

func TestSQLite_CreateSnapshot_OverwritesExisting(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	project, _ := db.CreateProject(ctx, "Test Project", "testuser")

	_, err := db.Conn.Exec(`INSERT INTO tasks (id, name, description, project_id) VALUES (?, ?, ?, ?)`,
		"old-task", "Old Task", "Old Description", project.ID)
	if err != nil {
		t.Fatalf("failed to insert existing task: %v", err)
	}

	newTasks := []ports.Task{
		{ID: "new-task", NewTask: ports.NewTask{Name: "New Task", Description: "New Description"}},
	}

	err = db.CreateSnapshot(ctx, project.ID, newTasks)
	if err != nil {
		t.Fatalf("CreateSnapshot() failed: %v", err)
	}

	loaded, err := db.LoadSnapshot(ctx, project.ID)
	if err != nil {
		t.Fatalf("LoadSnapshot() failed: %v", err)
	}

	if len(loaded) != 1 {
		t.Errorf("expected 1 task after overwrite, got %d", len(loaded))
	}

	if loaded[0].Name != "New Task" {
		t.Errorf("expected 'New Task', got '%s'", loaded[0].Name)
	}
}

func TestSQLite_CreateSnapshot_ClosedConnection(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.Conn.Close()

	ctx := context.Background()
	tasks := []ports.Task{
		{ID: "task1", NewTask: ports.NewTask{Name: "Task One", Description: "Desc"}},
	}

	err := db.CreateSnapshot(ctx, "some-project-id", tasks)
	if err == nil {
		t.Error("CreateSnapshot() should fail when connection is closed")
	}
}