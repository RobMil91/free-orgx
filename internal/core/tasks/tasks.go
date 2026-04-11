package tasks

import "context"

type NewTaskInput struct {
	ProjectID string
	Name      string
}

type Task struct {
	ID string
	NewTaskInput
}

func NewTask(ctx context.Context, n NewTaskInput) (*Task, error) {
	panic("ni")
}

type TaskProducer interface {
	NewEvent(ctx context.Context)
}

// Organizer is used to organize tasks
type Organizer struct {
	Producer TaskProducer
}

func NewOrganizer(p TaskProducer) (*Organizer, error) {
	panic("ni")
}
