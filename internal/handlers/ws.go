package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"slices"

	"github.com/RobMil91/free-orgx/internal/adapters/clients"
	"github.com/RobMil91/free-orgx/internal/core/event"
	"github.com/RobMil91/free-orgx/internal/models"
	"github.com/RobMil91/free-orgx/internal/ports"
	"github.com/gorilla/websocket"
)

func (p *Project) WebsocketHandler(w http.ResponseWriter, r *http.Request) {
	user, err := p.authUser(r)
	if err != nil {
		http.Error(w, err.Error(), 401)
		return
	}

	p.Logger.DebugContext(r.Context(), fmt.Sprintf("user %s attempt ws connect", user.Name))

	projectID := r.PathValue("id")

	//TODO: load the new events, and put them on the board.
	//Need the code from the observer that can translate the events
	//they all need to be send.
	p.Logger.DebugContext(r.Context(),
		fmt.Sprintf("attempt to subscribe to task events for project %s", projectID))

	//TODO: ws connection is dual -> need to pass consumer and producer
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		p.Logger.ErrorContext(r.Context(), err.Error())
		return
	}
	p.Logger.DebugContext(r.Context(), "successful websocket connection")

	wsID, err := models.CreateRandStr(15)
	if err != nil {
		p.Logger.ErrorContext(r.Context(), err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	subjectObserver, err := event.NewTaskObserver(p.Logger, p.EventTranslator, projectID, clients.NewWS(conn), *wsID)
	if err != nil {
		p.Logger.ErrorContext(r.Context(), err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, ok := p.TasksSubjects[projectID]
	if !ok {
		p.TasksSubjects[projectID] = *event.NewTaskSubject(p.Logger)
	}

	subject, ok := p.TasksSubjects[projectID]
	if !ok {
		p.Logger.ErrorContext(r.Context(), "internal problem subject is not initalized race cond?")
		return
	}

	err = subject.Add(subjectObserver)

	if err != nil {
		p.Logger.ErrorContext(r.Context(), err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func(o event.TaskObserver, projectID string) {
		subject := p.TasksSubjects[projectID]
		if err := subject.Remove(&o); err != nil {
			p.Logger.ErrorContext(r.Context(), err.Error())
		}

		conn.Close()
	}(*subjectObserver, projectID)

	for {
		typ, msg, err := conn.ReadMessage()
		if err != nil {
			p.Logger.ErrorContext(r.Context(), err.Error())
			break
		}

		p.Logger.DebugContext(r.Context(),
			fmt.Sprintf("retrieved via websocket message: %s, of type: %d",
				string(msg),
				typ,
			))

		var eTyp ports.EventType

		err = json.Unmarshal(msg, &eTyp)
		if err != nil {

			p.Logger.ErrorContext(r.Context(), err.Error())
			continue
		}

		switch eTyp.T {
		default:
			p.Logger.WarnContext(r.Context(), fmt.Sprintf("unkown event type %s", eTyp.T))
			continue

		case "deleteTask":
			taskCardID, err := parseTaskID(msg)
			if err != nil {
				p.Logger.ErrorContext(r.Context(), err.Error())
				continue
			}

			err = p.handlEvent(r.Context(), models.TaskEvent{
				EventID: *taskCardID,
				TaskEventRequest: models.TaskEventRequest{
					Type: "deleteTask",
				},
			}, projectID)
			if err != nil {
				p.Logger.ErrorContext(r.Context(), err.Error())
				continue
			}

			continue

		case "create":
			event, err := parseCreate(msg)
			if err != nil {
				p.Logger.ErrorContext(r.Context(), err.Error())
				continue
			}

			event.User = user.Name

			p.Logger.DebugContext(r.Context(),
				fmt.Sprintf("serialized to %+v",
					event,
				))

			newEvent, err := p.EventsRepo.NewEvent(r.Context(), projectID, *event)
			if err != nil {
				p.Logger.ErrorContext(r.Context(), err.Error())
				continue
			}

			subject, ok := p.TasksSubjects[projectID]
			if !ok {
				p.Logger.ErrorContext(r.Context(),
					fmt.Sprintf("no subject project id found, within websocket connection %s", projectID))
				http.Error(w,
					fmt.Sprintf("no subject project id found, within websocket connection %s", projectID),
					http.StatusInternalServerError)
				continue
			}

			err = subject.Notify(r.Context(), *newEvent)
			if err != nil {
				p.Logger.ErrorContext(r.Context(), err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				continue
			}
			continue

			// it just direct back command, sort them later, it is currently not added to db
		case "edit":
			taskCardID, err := parseTaskID(msg)
			if err != nil {
				p.Logger.ErrorContext(r.Context(), err.Error())
				continue
			}

			if err := p.handleEdit(r.Context(), projectID, *taskCardID, conn); err != nil {
				p.Logger.ErrorContext(r.Context(), err.Error())
				continue
			}

		case "edit-event":
			event, err := parseEdit(msg)
			if err != nil {
				p.Logger.ErrorContext(r.Context(), err.Error())
				continue
			}

			event.Type = ports.EditEvent

			event.User = user.Name

			event.ProjectID = projectID

			p.Logger.DebugContext(r.Context(),
				fmt.Sprintf("serialized to %+v",
					event,
				))

			p.Logger.DebugContext(r.Context(), fmt.Sprintf("storing new  edit event %+v", *event))
			newEvent, err := p.EventsRepo.NewEvent(r.Context(), projectID, *event)
			if err != nil {
				p.Logger.ErrorContext(r.Context(), err.Error())
				continue
			}

			subject, ok := p.TasksSubjects[projectID]
			if !ok {
				p.Logger.ErrorContext(r.Context(),
					fmt.Sprintf("no subject project id found, within websocket connection %s", projectID))
				http.Error(w,
					fmt.Sprintf("no subject project id found, within websocket connection %s", projectID),
					http.StatusInternalServerError)
				continue
			}

			err = subject.Notify(r.Context(), *newEvent)
			if err != nil {
				p.Logger.ErrorContext(r.Context(), err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				continue
			}

			if err := p.handleEdit(r.Context(), projectID, newEvent.TaskID, conn); err != nil {
				p.Logger.ErrorContext(r.Context(), err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				continue
			}

			continue

		case "get-create-card":
			users, err := p.UsersHTML(r.Context())
			if err != nil {
				p.Logger.ErrorContext(r.Context(), err.Error())
				continue
			}

			if err := sendTemplate(p.TemplatePath+"create_card.html", map[string]any{
				"Users": users,
			}, conn); err != nil {
				p.Logger.ErrorContext(r.Context(), err.Error())
				continue
			}

			continue

		}
	}
}

func (p *Project) handleEdit(ctx context.Context, projectID, taskCardID string, c *websocket.Conn) error {
	p.Logger.DebugContext(ctx, fmt.Sprintf("edit event card request %s", taskCardID))

	events, err := p.EventsRepo.GetEvents(ctx, projectID)
	if err != nil {
		p.Logger.ErrorContext(ctx, err.Error())
		return err
	}
	p.Logger.DebugContext(ctx, fmt.Sprintf("events %+v", events))

	events = models.FilterTaskEvents(events, taskCardID)

	taskState, err := models.EventsToState(events)
	if err != nil {
		p.Logger.ErrorContext(ctx, err.Error())
		return err
	}

	p.Logger.DebugContext(ctx, fmt.Sprintf("endState %+v, assignes %d", taskState, len(taskState.Assigned)))

	if len(taskState.Assigned) == 1 && taskState.Assigned[0] == "" {
		taskState.Assigned = models.FlexibleStringArray([]string{
			"Unassigned",
		})
	}

	p.Logger.DebugContext(ctx, fmt.Sprintf("endState %+v", taskState))

	tmpl := template.Must(template.ParseFiles(p.TemplatePath + "edit_card.html"))

	users, err := p.UsersHTML(ctx)
	if err != nil {
		p.Logger.ErrorContext(ctx, err.Error())
		return err
	}

	users = slices.DeleteFunc(users, func(u UsersHTML) bool {
		return slices.Contains(taskState.Assigned, u.Name)
	})

	if !slices.ContainsFunc(taskState.Assigned, func(u string) bool {
		return u == "Unassigned"
	}) {
		users = append(users, UsersHTML{
			Name: "Unassigned",
		})
	}

	var buffer bytes.Buffer

	err = tmpl.Execute(&buffer, map[string]any{
		"Title":        taskState.Title,
		"Description":  taskState.Description,
		"TaskID":       taskState.TaskID,
		"Users":        users,
		"SelectedUser": taskState.Assigned,
	})

	if err != nil {
		p.Logger.ErrorContext(ctx, err.Error())
		return err
	}

	err = c.WriteMessage(websocket.TextMessage, buffer.Bytes())
	if err != nil {
		p.Logger.ErrorContext(ctx, err.Error())
		return err
	}

	return nil
}

func (p *Project) handlEvent(ctx context.Context, te models.TaskEvent, projectID string) error {
	subject, ok := p.TasksSubjects[projectID]
	if !ok {
		return fmt.Errorf("no subject project id found, within websocket connection %s", projectID)
	}

	err := subject.Notify(ctx, te)
	if err != nil {
		return fmt.Errorf("notify error [%w]", err)
	}

	return nil
}

func parseCreate(b []byte) (*models.TaskEventRequest, error) {
	var event models.TaskEventRequest

	if err := json.Unmarshal(b, &event); err != nil {
		return nil, err
	}

	return &event, nil
}

func parseEdit(b []byte) (*models.TaskEventRequest, error) {
	var event models.EditTask

	if err := json.Unmarshal(b, &event); err != nil {
		return nil, err
	}

	if slices.ContainsFunc(event.Assigned, func(u string) bool {
		return u == "Unassigned"
	}) && len(event.Assigned) > 1 {
		return nil, fmt.Errorf("can not unassign and put another person also %+v", event.Assigned)
	}

	return &models.TaskEventRequest{
		NewTask: event.NewTask,
		TaskID:  event.TaskID,
	}, nil
}

func parseTaskID(b []byte) (*string, error) {
	var event ports.TaskID

	if err := json.Unmarshal(b, &event); err != nil {
		return nil, err
	}

	return &event.ID, nil
}

func sendTemplate(templatePath string, values map[string]any, c *websocket.Conn) error {
	tmpl := template.Must(template.ParseFiles(templatePath))
	var buffer bytes.Buffer

	err := tmpl.Execute(&buffer, values)
	if err != nil {
		return err
	}

	err = c.WriteMessage(websocket.TextMessage, buffer.Bytes())
	if err != nil {
		return err
	}

	return nil
}
