package ports

import (
	"context"

	"github.com/RobMil91/free-orgx/internal/models"
)

type EventTranslator interface {
	CreateRowEvent(ctx context.Context, e models.TaskEvent) ([]byte, error)
	CreateEventAndRow(ctx context.Context, e models.TaskEvent) ([]byte, error)
	// GetEvents responds the messages needed to send, on empty board
	GetPreviousEvents(ctx context.Context, pID string) ([]models.TaskEvent, error)

	UpdateEvent(ctx context.Context, e models.TaskEvent) ([][]byte, error)
}
