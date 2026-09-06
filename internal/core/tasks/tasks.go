package tasks

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"strconv"
	"text/template"

	"github.com/RobMil91/free-orgx/internal/models"
	"github.com/RobMil91/free-orgx/internal/ports"
)

var _ ports.EventTranslator = (*TaskBoard)(nil)

type TaskBoard struct {
	Topics ports.EventStore
	Files  fs.FS
}

func NewTaskBoard(e ports.EventStore, f fs.FS) *TaskBoard {
	return &TaskBoard{
		Files:  f,
		Topics: e,
	}
}

// CreateEventAndRow implements [ports.EventTranslator].
func (t *TaskBoard) CreateEventAndRow(ctx context.Context, e models.TaskEvent) ([]byte, error) {
	return t.rowAndCard(ctx, e)
}

// UpdateEvent implements [ports.EventTranslator].
func (t *TaskBoard) UpdateEvent(ctx context.Context, e models.TaskEvent) ([][]byte, error) {
	var (
		messages [][]byte
	)
	oldState, newestEvent, err := t.getLastTaskStatus(ctx, e)
	if err != nil {
		return nil, fmt.Errorf("could not get state and newest Event %w", err)
	}

	updated, updatedTask := models.Diff(oldState.NewTask, newestEvent.NewTask)

	if updated.Status {
		//delete old
		//determine position on board
		pos, err := t.rowPosition(ctx, e.TaskID, e.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("failed to get position %w", err)
		}

		posState := oldState.Status
		if posState == "In Progress" {
			posState = "InProgress"
		}

		oldPosition := fmt.Sprintf("%s.%s", *pos, posState)

		msg1, err := t.deleteDiv(ctx, oldPosition)
		if err != nil {
			return nil, fmt.Errorf("failed to delete div %w", err)
		}

		messages = append(messages, msg1)

		var rowPosition string = e.Status
		if e.NewTask.Status == "In Progress" {
			rowPosition = "InProgress"
		}

		newPosition := fmt.Sprintf("%s.%s", *pos, rowPosition)

		msg2, err := t.updateRowCard(ctx, models.CardInput{
			CardPosition: newPosition,
			Title:        updatedTask.Title,
			ID:           oldState.TaskID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update row card %w", err)
		}

		messages = append(messages, msg2)

	}

	return messages, nil

}

// GetPreviousEvents implements [ports.EventTranslator].
func (t *TaskBoard) GetPreviousEvents(ctx context.Context, pID string) ([]models.TaskEvent, error) {
	var (
		events []models.TaskEvent
	)
	//can be refactored into one html send.

	//calculate only the events that need to be send
	// create event plus update to newest state
	// per row

	previousEvents, err := t.Topics.GetEvents(ctx, pID)
	if err != nil {
		return nil, err
	}

	for _, e := range previousEvents {
		if e.Type == "create" {
			newEvent := models.TaskEvent{
				TaskEventRequest: models.TaskEventRequest{
					TaskID:    e.TaskID,
					Type:      "create-row",
					ProjectID: e.ProjectID,
				},
			}

			events = append(events, newEvent)

			state, event, err := t.getLastTaskStatus(context.Background(), e)
			if err != nil {
				return nil, fmt.Errorf("could not scrap history to state %w", err)
			}

			if event.Type == "create" && state.Status == "Todo" {
				event.Type = ports.EditEvent
			}

			events = append(events, *event)
		}
	}

	//TODO: is this correct in here?
	// for _, e := range previousEvents {
	// 	if err := newObserver.update(context.Background(), e); err != nil {
	// 		newObserver.Logger.ErrorContext(ctx, err.Error())
	// 		return nil, fmt.Errorf("failed to send event (%+v), because [%w]", e, err)
	// 	}
	// }

	return events, nil
}

// CreateRowEvent implements [ports.EventTranslator].
func (t *TaskBoard) CreateRowEvent(ctx context.Context, e models.TaskEvent) ([]byte, error) {
	return t.rowHTML(ctx, e)
}

func (t *TaskBoard) rowHTML(ctx context.Context, e models.TaskEvent) ([]byte, error) {
	tmpl, err := template.ParseFS(t.Files, "empty_row.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse file empty row %w", err)
	}

	var buffer bytes.Buffer

	rowNumber, err := t.rowPosition(ctx, e.TaskID, e.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("creating event row failed %w", err)
	}

	err = tmpl.Execute(&buffer, map[string]string{
		"RowNumber": *rowNumber,
		"ID":        e.TaskID,
	})

	if err != nil {
		return nil, fmt.Errorf("could not append data to templ [%w]", err)
	}

	return buffer.Bytes(), nil
}

func (t *TaskBoard) rowPosition(ctx context.Context, taskID, projectID string) (*string, error) {
	events, err := t.Topics.GetEvents(ctx, projectID)
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

func (t *TaskBoard) getLastTaskStatus(ctx context.Context, e models.TaskEvent) (*models.Task, *models.TaskEvent, error) {
	allEvents, err := t.Topics.GetEvents(ctx, e.ProjectID)
	if err != nil {
		return nil, nil, err
	}

	var rowEvents []models.TaskEvent

	for _, ele := range allEvents {
		if ele.TaskID == e.TaskID {
			rowEvents = append(rowEvents, ele)
		}
	}

	// we need to filter for task, but what if it is acutally just the first create?
	if len(rowEvents) < 1 {
		return nil, nil, fmt.Errorf("did not find  enough row events for taskID %s, in project %s, events %d", e.TaskID, e.ProjectID, len(rowEvents))
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

	oldState, err := models.EventsToState(rowEvents[:len(rowEvents)-1])
	if err != nil {
		return nil, nil, err
	}

	newEvent := rowEvents[len(rowEvents)-1]

	return oldState, &newEvent, nil
}

func (t *TaskBoard) updateRowCard(ctx context.Context, i models.CardInput) ([]byte, error) {
	tmpl, err := template.ParseFS(t.Files, "card.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse file empty row %w", err)
	}

	var buffer2 bytes.Buffer

	err = tmpl.Execute(&buffer2, map[string]string{
		"Title":        i.Title,
		"ID":           i.ID,
		"CardPosition": i.CardPosition,
	})

	if err != nil {
		return nil, fmt.Errorf("could not append data to templ [%w]", err)
	}

	msg2 := buffer2.Bytes()

	return msg2, nil
}

func (t *TaskBoard) deleteDiv(ctx context.Context, boardPosition string) ([]byte, error) {
	tmpl, err := template.ParseFS(t.Files, "task_delete.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse file empty row %w", err)
	}

	var buffer bytes.Buffer

	err = tmpl.Execute(&buffer, map[string]string{
		"BoardPosition": boardPosition,
	})

	if err != nil {
		return nil, fmt.Errorf("could not append data to templ [%w]", err)
	}

	return buffer.Bytes(), nil
}

func (t *TaskBoard) rowAndCard(ctx context.Context, e models.TaskEvent) ([]byte, error) {
	tmpl, err := template.ParseFS(t.Files, "task_card.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse file empty row %w", err)
	}

	var buffer bytes.Buffer

	rowNumber, err := t.rowPosition(ctx, e.TaskID, e.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("creating event row failed %w", err)
	}

	err = tmpl.Execute(&buffer, map[string]string{
		"Title":       e.Title,
		"Description": e.Description,
		"ID":          e.TaskID,
		"RowNumber":   *rowNumber,
	})

	if err != nil {
		return nil, fmt.Errorf("could not append data to templ [%w]", err)
	}

	return buffer.Bytes(), nil
}
