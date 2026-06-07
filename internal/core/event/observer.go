package event

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"strconv"

	"github.com/RobMil91/free-orgx/internal/models"
	"github.com/RobMil91/free-orgx/internal/ports"
	"github.com/gorilla/websocket"
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
	ID           string
	Conn         *websocket.Conn
	Logger       *slog.Logger
	TemplatePath string
	Events       ports.EventStore
	ProjectID    string
}

func NewTaskObserver(id string,
	conn *websocket.Conn,
	l *slog.Logger,
	tp string,
	t ports.EventStore,
	pID string,
) (*TaskObserver, error) {

	newObserver := TaskObserver{
		ID:           id,
		Conn:         conn,
		Logger:       l,
		TemplatePath: tp,
		Events:       t,
		ProjectID:    pID,
	}

	previousEvents, err := newObserver.Events.GetEvents(context.TODO(), pID)
	if err != nil {
		newObserver.Logger.ErrorContext(context.TODO(), err.Error())
		return nil, err
	}

	//can be refactored into one html send.

	//calculate only the events that need to be send
	// create event plus update to newest state
	// per row

	for _, e := range previousEvents {
		if e.Type == "create" {
			if err := newObserver.update(context.Background(), models.TaskEvent{
				TaskEventRequest: models.TaskEventRequest{
					TaskID:    e.TaskID,
					Type:      "create-row",
					ProjectID: e.ProjectID,
				},
			}); err != nil {
				return nil, fmt.Errorf("failed to send event (%+v), because [%w]", e, err)
			}

			state, event, err := newObserver.getLastTaskStatus(context.Background(), e)
			if err != nil {
				return nil, fmt.Errorf("could not scrap history to state %w", err)
			}

			if event.Type == "create" && state.Status == "Todo" {
				newObserver.Logger.DebugContext(context.TODO(), "changed event type")
				event.Type = ports.EditEvent
			}
			//if last event is only an create this will create an extra row...
			if err := newObserver.update(context.Background(), *event); err != nil {
				return nil, fmt.Errorf("failed to send event (%+v), because [%w]", e, err)
			}

		}
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
		msg, err := o.rowHTML(t)
		if err != nil {
			return err
		}

		o.Logger.DebugContext(ctx, fmt.Sprintf("sending message update [ %s ]", string(msg)))
		err = o.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			return err
		}

	case "create":
		msg, err := o.rowAndCard(t)
		if err != nil {
			return err
		}

		o.Logger.DebugContext(ctx, fmt.Sprintf("sending message update [ %s ]", string(msg)))
		err = o.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			return err
		}

	case ports.EditEvent:
		o.Logger.DebugContext(ctx, fmt.Sprintf("got edit event [ %+v ]", t))
		// old, err := o.Events.GetEvent(ctx, t.EventID)
		// if err != nil {
		// 	return fmt.Errorf("edit event could not find taskID %s events, because %w", t.Status, err)
		// }

		oldState, newestEvent, err := o.getLastTaskStatus(ctx, t)
		if err != nil {
			return fmt.Errorf("could not get state and newest Event %w", err)
		}

		updated, updatedTask := models.Diff(oldState.NewTask, newestEvent.NewTask)

		if updated.Status {
			//delete old
			//determine position on board
			pos, err := o.rowPosition(ctx, t.TaskID, t.ProjectID)
			if err != nil {
				return fmt.Errorf("failed to get position %w", err)
			}

			posState := oldState.Status
			if posState == "In Progress" {
				posState = "InProgress"
			}

			oldPosition := fmt.Sprintf("%s.%s", *pos, posState)

			err = o.deleteDiv(ctx, oldPosition)
			if err != nil {
				return fmt.Errorf("failed to delete div %w", err)
			}

			var rowPosition string = t.Status
			if t.NewTask.Status == "In Progress" {
				rowPosition = "InProgress"
			}

			newPosition := fmt.Sprintf("%s.%s", *pos, rowPosition)

			err = o.updateRowCard(ctx, models.CardInput{
				CardPosition: newPosition,
				Title:        updatedTask.Title,
				ID:           oldState.TaskID,
			})
			if err != nil {
				return fmt.Errorf("failed to update row card %w", err)
			}

		}
	}

	return nil
}

func (o *TaskObserver) getLastTaskStatus(ctx context.Context, t models.TaskEvent) (*models.Task, *models.TaskEvent, error) {
	o.Logger.DebugContext(ctx, fmt.Sprintf("getting events for project %s", t.ProjectID))
	allEvents, err := o.Events.GetEvents(ctx, t.ProjectID)
	if err != nil {
		return nil, nil, err
	}

	o.Logger.DebugContext(ctx, fmt.Sprintf("found events for project %d", len(allEvents)))

	var rowEvents []models.TaskEvent

	o.Logger.DebugContext(ctx, fmt.Sprintf("collecting taskID %s", t.TaskID))
	for _, e := range allEvents {
		// o.Logger.DebugContext(ctx, fmt.Sprintf("comparing %s %s", e.TaskID, t.TaskID))
		if e.TaskID == t.TaskID {
			rowEvents = append(rowEvents, e)
		}
	}

	o.Logger.DebugContext(ctx, fmt.Sprintf("found events %d", len(rowEvents)))
	o.Logger.DebugContext(ctx, fmt.Sprintf("found events %+v", rowEvents))

	// we need to filter for task, but what if it is acutally just the first create?
	if len(rowEvents) < 1 {
		return nil, nil, fmt.Errorf("did not find  enough row events for taskID %s, in project %s, events %d", t.TaskID, t.ProjectID, len(rowEvents))
	}

	// if len(rowEvents) == 1 && rowEvents[0].Type =="create" {
	if len(rowEvents) == 1 {
		return &models.Task{
			TaskID:  rowEvents[0].TaskID,
			NewTask: rowEvents[0].NewTask,
		}, &rowEvents[0], nil
	}

	// all the events should already be in database
	// meaning we need to update fromm n-1 to n
	// the send event should contain the new data of n, which is bad... probably should ignore.
	//
	// o.Logger.DebugContext(ctx, fmt.Sprintf("events to merge %+v", rowEvents[len(rowEvents)-1:]))

	//skip last element
	oldState, err := models.EventsToState(rowEvents)
	if err != nil {
		return nil, nil, err
	}

	newEvent := rowEvents[len(rowEvents)-1]

	return oldState, &newEvent, nil
}

func (o *TaskObserver) updateRowCard(ctx context.Context, i models.CardInput) error {
	tmpl := template.Must(template.ParseFiles(o.TemplatePath + "card.html"))

	var buffer2 bytes.Buffer

	err := tmpl.Execute(&buffer2, map[string]string{
		"Title":        i.Title,
		"ID":           i.ID,
		"CardPosition": i.CardPosition,
	})

	if err != nil {
		return fmt.Errorf("could not append data to templ [%w]", err)
	}

	msg2 := buffer2.Bytes()

	o.Logger.DebugContext(ctx, fmt.Sprintf("sending create new card [ %s ]", string(msg2)))
	err = o.Conn.WriteMessage(websocket.TextMessage, msg2)
	if err != nil {
		return err
	}

	return nil
}

func (o *TaskObserver) rowPosition(ctx context.Context, taskID, projectID string) (*string, error) {
	events, err := o.Events.GetEvents(ctx, projectID)
	if err != nil {
		return nil, err
	}

	rowNum := 2 //header + first row
	for _, e := range events {

		if e.Type == "edit-event" {
			continue
		}

		if e.TaskID == taskID {
			rN := strconv.Itoa(rowNum)
			return &rN, nil
		}

		rowNum++

		//TODO: the table hold all projectIDs, there can be multiple rows, how to know i added one?
		//when i have another create there is one more row..
		// if e.Type == "create" {
		// 	rowNum++
		// }
	}

	return nil, fmt.Errorf("did not find taskID %s in projectID %s", taskID, projectID)
}

func (o *TaskObserver) deleteDiv(ctx context.Context, boardPosition string) error {
	tmpl := template.Must(template.ParseFiles(o.TemplatePath + "task_delete.html"))

	var buffer bytes.Buffer

	err := tmpl.Execute(&buffer, map[string]string{
		"BoardPosition": boardPosition,
	})

	if err != nil {
		return fmt.Errorf("could not append data to templ [%w]", err)
	}

	msg := buffer.Bytes()

	o.Logger.DebugContext(context.TODO(), fmt.Sprintf("sending message delete [ %s ]", string(msg)))
	err = o.Conn.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		return err
	}

	return nil
}

// Update implements [observer].
func (o *TaskObserver) Update(t models.TaskEvent) error {
	return o.update(context.TODO(), t)
}

func (o *TaskObserver) rowHTML(t models.TaskEvent) ([]byte, error) {
	tmpl := template.Must(template.ParseFiles(o.TemplatePath + "empty_row.html"))

	var buffer bytes.Buffer

	rowNumber, err := o.rowPosition(context.TODO(), t.TaskID, t.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("creating event row failed %w", err)
	}

	err = tmpl.Execute(&buffer, map[string]string{
		"RowNumber": *rowNumber,
	})

	if err != nil {
		return nil, fmt.Errorf("could not append data to templ [%w]", err)
	}

	return buffer.Bytes(), nil
}

func (o *TaskObserver) rowAndCard(t models.TaskEvent) ([]byte, error) {
	tmpl := template.Must(template.ParseFiles(o.TemplatePath + "task_card.html"))

	var buffer bytes.Buffer

	rowNumber, err := o.rowPosition(context.TODO(), t.TaskID, t.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("creating event row failed %w", err)
	}

	err = tmpl.Execute(&buffer, map[string]string{
		"Title":       t.Title,
		"Description": t.Description,
		"ID":          t.TaskID,
		"RowNumber":   *rowNumber,
	})

	if err != nil {
		return nil, fmt.Errorf("could not append data to templ [%w]", err)
	}

	return buffer.Bytes(), nil
}
