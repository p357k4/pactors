package typed

import (
	"context"
	"errors"
	"log/slog"
)

type envelope[T any] struct {
	msg T
	ctx context.Context
}

type mailbox[T any] struct {
	channel chan envelope[T] // actor channel which must be closed on reader side
}

func (r mailbox[T]) Close() error {
	close(r.channel)
	return nil
}

// Send sends an envelope to the actor's channel
func (r mailbox[T]) Send(ctx context.Context, msg T) (err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "forever loop panicked", slog.Any("recover", r))
			err = errors.New("failed to send envelope")
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case r.channel <- envelope[T]{msg: msg, ctx: ctx}:
		return nil
	}
}
