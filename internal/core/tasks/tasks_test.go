package tasks

import (
	"context"
	"errors"
	"testing"

	"github.com/RobMil91/free-orgx/internal/models"
	"github.com/RobMil91/free-orgx/internal/ports"
)

type mockEventStore struct {
	events map[string][]models.TaskEvent
	err    error
}

func (m *mockEventStore) NewEvent(ctx context.Context, projectID string, t models.TaskEventRequest) (*models.TaskEvent, error) {
	panic("ni")
}

func (m *mockEventStore) GetEvents(ctx context.Context, projectID string) ([]models.TaskEvent, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.events[projectID], nil
}

func newTestBoard(e ports.EventStore) *TaskBoard {
	return NewTaskBoard(e, nil)
}

func TestGetPreviousEvents_Empty(t *testing.T) {
	store := &mockEventStore{events: map[string][]models.TaskEvent{}}
	b := newTestBoard(store)

	events, err := b.GetPreviousEvents(context.Background(), "proj1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
	//TODO: this has to error
}

func TestGetPreviousEvents_CreateWithTodoStatus(t *testing.T) {
	store := &mockEventStore{
		events: map[string][]models.TaskEvent{
			"proj1": {
				{
					TaskEventRequest: models.TaskEventRequest{
						TaskID:    "task1",
						Type:      "create",
						ProjectID: "proj1",
						NewTask:   models.NewTask{Title: "todo task", Status: "Todo"},
					},
				},
			},
		},
	}
	b := newTestBoard(store)

	events, err := b.GetPreviousEvents(context.Background(), "proj1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 { // this seems wrong there is only one task in mock why would that be 2?
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[0].Type != "create-row" {
		t.Errorf("expected first event type create-row, got %s", events[0].Type)
	}
	if events[0].TaskID != "task1" {
		t.Errorf("expected first event TaskID task1, got %s", events[0].TaskID)
	}

	if events[1].Type != ports.EditEvent {
		t.Errorf("expected second event type %s, got %s", ports.EditEvent, events[1].Type)
	}
}
