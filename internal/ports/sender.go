package ports

import "context"

type Sender interface {
	Send(ctx context.Context, msg []byte) error
}
