package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/RobMil91/free-orgx/internal/core/event"
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

	previousEvents, err := p.EventsRepo.GetEvents(r.Context(), projectID)
	if err != nil {
		p.Logger.ErrorContext(r.Context(), err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//TODO: load the new events, and put them on the board.
	//Need the code from the observer that can translate the events
	//they all need to be send.
	p.Logger.DebugContext(r.Context(), fmt.Sprintf("attempt to subscribe to task events for project %s", projectID))

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

	wsID, err := createRandStr(15)
	if err != nil {
		p.Logger.ErrorContext(r.Context(), err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	subjectObserver, err := event.NewTaskObserver(*wsID, conn, p.Logger, p.TemplatePath, previousEvents)
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

		event, err := parse(msg)
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
			p.Logger.ErrorContext(r.Context(), fmt.Sprintf("no subject project id found, within websocket connection %s", projectID))
			http.Error(w, fmt.Sprintf("no subject project id found, within websocket connection %s", projectID), http.StatusInternalServerError)
			return
		}

		err = subject.Notify(r.Context(), *newEvent)
		if err != nil {
			p.Logger.ErrorContext(r.Context(), err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// code snipet test remove after TODO
		// resp := map[string]any{
		// 	"echo": string(msg),
		// }

		// if err = conn.WriteJSON(resp); err != nil {
		// 	p.Logger.ErrorContext(r.Context(), err.Error())
		// 	break
		// }
	}
}

func parse(b []byte) (*ports.TaskEventRequest, error) {
	var event ports.TaskEventRequest

	if err := json.Unmarshal(b, &event); err != nil {
		return nil, err
	}

	return &event, nil
}

func createRandStr(length int) (*string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	randStr := base64.URLEncoding.EncodeToString(b)
	return &randStr, nil
}
