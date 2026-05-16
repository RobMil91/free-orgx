package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

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

	id := r.PathValue("id")
	p.Logger.DebugContext(r.Context(), fmt.Sprintf("attempt to subscribe to task events for project %s", id))

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

	defer conn.Close()

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

		if err = p.EventsRepo.NewEvent(r.Context(), id, *event); err != nil {
			p.Logger.ErrorContext(r.Context(), err.Error())
			continue
		}

		resp := map[string]any{
			"echo": string(msg),
		}

		if err = conn.WriteJSON(resp); err != nil {
			p.Logger.ErrorContext(r.Context(), err.Error())
			break
		}
	}
}

func parse(b []byte) (*ports.TaskEventRequest, error) {
	var event ports.TaskEventRequest

	if err := json.Unmarshal(b, &event); err != nil {
		return nil, err
	}

	return &event, nil
}
