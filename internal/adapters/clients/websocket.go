package clients

import (
	"context"

	"github.com/RobMil91/free-orgx/internal/ports"
	"github.com/gorilla/websocket"
)

var _ ports.Sender = (*WS)(nil)

type WS struct {
	Conn *websocket.Conn
}

// Send implements [ports.Sender].
func (w *WS) Send(ctx context.Context, msg []byte) error {
	err := w.Conn.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		return err
	}

	return nil
}
