package models

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type NewTask struct {
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Deadline    string              `json:"deadline"`
	Status      string              `json:"status"`
	Assigned    FlexibleStringArray `json:"assignees"`
}

type FlexibleStringArray []string

func (f *FlexibleStringArray) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = []string{s}
		return nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*f = arr
		return nil
	}
	return errors.New("expected string or array of strings")
}

func CreateRandStr(length int) (*string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	randStr := base64.URLEncoding.EncodeToString(b)
	return &randStr, nil
}

type EditTask struct {
	NewTask
	User   string
	TaskID string `json:"taskID"`
}

type Task struct {
	NewTask
	TaskID string `json:"id"`

	// Messages []string
}

type TaskEventRequest struct {
	Type      string `json:"event-type"`
	User      string
	ProjectID string
	NewTask

	TaskID string //is also the row ID for UI
}

type TaskEvent struct {
	EventID string //TODO: has to be EventID
	TaskEventRequest
	EventTime time.Time
}

func EventsToState(events []TaskEvent) (*Task, error) {
	var createEvent TaskEvent
	var otherEvents []TaskEvent

	found := false
	for _, e := range events {
		if e.Type == "create" {
			createEvent = e
			found = true
			continue
		}

		otherEvents = append(otherEvents, e)
	}

	if !found {
		return nil, fmt.Errorf("no create event found %+v", events)
	}

	current := createEvent

	// if len(otherEvents) == 1 {
	// 	return &Task{
	// 		NewTask: current.NewTask,
	// 		TaskID:  current.TaskID,
	// 	}, nil
	// }

	for _, o := range otherEvents {
		updated, newTask := Diff(current.NewTask, o.NewTask)

		if updated.Changed() {
			current.NewTask = newTask
		}
	}

	return &Task{
		NewTask: current.NewTask,
		TaskID:  current.TaskID,
	}, nil
}

type CardInput struct {
	CardPosition string
	Title        string
	ID           string
}

type Updated struct {
	Status      bool
	Deadline    bool
	Title       bool
	Description bool
	User        bool
}

func (u Updated) Changed() bool {
	if u.Status {
		return true
	}

	if u.User {
		return true
	}

	// if u.deadline {}
	if u.Title {
		return true
	}

	if u.Description {
		return true
	}

	return false
}

func Diff(o, n NewTask) (Updated, NewTask) {
	u := Updated{}
	result := o
	if n.Status != "" {
		result.Status = n.Status
		u.Status = true
	}

	if n.Title != "" {
		result.Title = n.Title
		u.Title = true
	}

	if n.Deadline != "" {
		result.Deadline = n.Deadline
		u.Deadline = true
	}

	if n.Description != "" {
		result.Description = n.Description
		u.Description = true
	}

	if len(n.Assigned) != 0 {
		result.Assigned = n.Assigned
		u.User = true
	}

	return u, result
}

func FilterTaskEvents(events []TaskEvent, taskID string) []TaskEvent {
	var rowEvents []TaskEvent

	for _, e := range events {
		if e.TaskID == taskID {
			rowEvents = append(rowEvents, e)
		}
	}

	return rowEvents
}
