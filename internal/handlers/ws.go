package handlers

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

func (p *Project) TasksTopicHandler(w http.ResponseWriter, r *http.Request) {
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
	p.Logger.DebugContext(r.Context(), "successfull websocket connection")

	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			p.Logger.ErrorContext(r.Context(), err.Error())
			break
		}

		p.Logger.DebugContext(r.Context(), fmt.Sprintf("retrieved via websocket message: %s", string(msg)))

		resp := map[string]any{
			"echo": string(msg),
		}

		conn.WriteJSON(resp)
	}
}
