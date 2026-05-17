package event

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log/slog"

	"github.com/RobMil91/free-orgx/internal/ports"
	"github.com/gorilla/websocket"
)

type observer interface {
	Update(t ports.TaskEvent) error
	GetID() string
}

type subject interface {
	Add(o observer) error
	Remove(o observer) error
	Notify(ctx context.Context, t ports.TaskEvent) error
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
func (w *TaskSubject) Notify(ctx context.Context, t ports.TaskEvent) error {
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
	ID           string
	Conn         *websocket.Conn
	Logger       *slog.Logger
	TemplatePath string
}

func NewTaskObserver(id string,
	conn *websocket.Conn,
	l *slog.Logger,
	tp string,
	p []ports.TaskEvent,
) (*TaskObserver, error) {

	newObserver := TaskObserver{
		ID:           id,
		Conn:         conn,
		Logger:       l,
		TemplatePath: tp,
	}

	for _, e := range p {
		if err := newObserver.update(context.Background(), e); err != nil {
			return nil, fmt.Errorf("failed to send e (%+v), [%w]", e, err)
		}
	}

	return &newObserver, nil
}

// GetID implements [observer].
func (t *TaskObserver) GetID() string {
	return t.ID
}

func (o *TaskObserver) update(ctx context.Context, t ports.TaskEvent) error {
	msg, err := o.toHtml(t)
	if err != nil {
		return err
	}
	o.Logger.DebugContext(ctx, fmt.Sprintf("sending message update [ %s ]", string(msg)))
	err = o.Conn.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		return err
	}

	return nil
}

// Update implements [observer].
func (o *TaskObserver) Update(t ports.TaskEvent) error {
	return o.update(context.TODO(), t)
}

func (o *TaskObserver) toHtml(t ports.TaskEvent) ([]byte, error) {
	tmpl := template.Must(template.ParseFiles(o.TemplatePath + "task_card.html"))

	var buffer bytes.Buffer

	err := tmpl.Execute(&buffer, map[string]string{
		"Title":       t.Title,
		"Description": t.Description,
		"ID":          t.ID,
	})

	if err != nil {
		return nil, fmt.Errorf("could not append data to templ [%w]", err)
	}

	return buffer.Bytes(), nil
}
