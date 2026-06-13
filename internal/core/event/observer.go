package event

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/RobMil91/free-orgx/internal/models"
	"github.com/RobMil91/free-orgx/internal/ports"
)

type observer interface {
	Update(t models.TaskEvent) error
	GetID() string
}

type subject interface {
	Add(o observer) error
	Remove(o observer) error
	Notify(ctx context.Context, t models.TaskEvent) error
}

var _ subject = (*TaskSubject)(nil)

type TaskSubject struct {
	observers map[string]observer
	logger    *slog.Logger
}

func NewTaskSubject(l *slog.Logger) *TaskSubject {
	return &TaskSubject{
		logger:    l,
		observers: map[string]observer{},
	}
}

// Add implements [subject].
func (w *TaskSubject) Add(o observer) error {
	_, ok := w.observers[o.GetID()]
	if ok {
		return fmt.Errorf("already got id in map %s", o.GetID())
	}

	w.observers[o.GetID()] = o

	return nil
}

// Notify implements [subject].
func (w *TaskSubject) Notify(ctx context.Context, t models.TaskEvent) error {
	for _, c := range w.observers {
		err := c.Update(t)
		if err != nil {
			return fmt.Errorf("could not update %s, because %w", c.GetID(), err)
		}
	}

	return nil
}

// Remove implements [subject].
func (w *TaskSubject) Remove(o observer) error {
	_, ok := w.observers[o.GetID()]
	if !ok {
		return fmt.Errorf("can not remove id  %s", o.GetID())
	}

	delete(w.observers, o.GetID())
	return nil
}

var _ observer = (*TaskObserver)(nil)

type TaskObserver struct {
	ID               string
	Logger           *slog.Logger
	ProjectID        string
	EventTranslators ports.EventTranslator
}

func NewTaskObserver(
	l *slog.Logger,
	ev ports.EventTranslator,
	pID string,
) (*TaskObserver, error) {

	newObserver := TaskObserver{
		Logger:           l,
		ProjectID:        pID,
		EventTranslators: ev,
	}

	ctx := context.Background()

	err := newObserver.EventTranslators.GetPreviousEvents(ctx, pID)
	if err != nil {
		newObserver.Logger.ErrorContext(ctx, err.Error())
		return nil, err
	}

	return &newObserver, nil
}

// GetID implements [observer].
func (t *TaskObserver) GetID() string {
	return t.ID
}

func (o *TaskObserver) update(ctx context.Context, t models.TaskEvent) error {
	switch t.TaskEventRequest.Type {
	default:
		return fmt.Errorf("unkown event type for observer to send %s", t.TaskEventRequest.Type)
	case "create-row":
		err := o.EventTranslators.CreateEvent(ctx, t)
		if err != nil {
			return err
		}

	case "create":

		err := o.EventTranslators.CreateEventAndRow(ctx, t)
		if err != nil {
			return err
		}

		// o.Logger.DebugContext(ctx, fmt.Sprintf("sending message update [ %s ]", string(msg)))
		// err = o.Conn.WriteMessage(websocket.TextMessage, msg)
		// if err != nil {
		// 	return err
		// }

	case ports.EditEvent:
		o.Logger.DebugContext(ctx, fmt.Sprintf("got edit event [ %+v ]", t))

		err := o.EventTranslators.UpdateEvent(ctx, t)
		if err != nil {
			return err
		}

	}

	return nil
}

// Update implements [observer].
func (o *TaskObserver) Update(t models.TaskEvent) error {
	return o.update(context.TODO(), t)
}
