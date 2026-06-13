package ports

import (
	"context"

	"github.com/RobMil91/free-orgx/internal/models"
)

type EventTranslator interface {
	CreateEvent(ctx context.Context, e models.TaskEvent) error
	CreateEventAndRow(ctx context.Context, e models.TaskEvent) error
	// GetEvents responds the messages needed to send, on empty board
	GetPreviousEvents(ctx context.Context, pID string) error

	UpdateEvent(ctx context.Context, e models.TaskEvent) error
}
