package clients

import (
	"context"

	"github.com/RobMil91/free-orgx/internal/ports"
	"github.com/gorilla/websocket"
)

var _ ports.Sender = (*WS)(nil)

type WS struct {
	conn *websocket.Conn
}

func NewWS(c *websocket.Conn) *WS {
	return &WS{
		conn: c,
	}
}

// Send implements [ports.Sender].
func (w *WS) Send(ctx context.Context, msg []byte) error {
	err := w.conn.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		return err
	}

	return nil
}
