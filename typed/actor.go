package typed

import (
	"context"
	"errors"
)

// Receiver defines the interface for actors with a generic envelope type
type Receiver[T any] interface {
	Receive(ctx context.Context, msg T) error
}

type Actor[T any] interface {
	Send(context.Context, T) error
	Wait()
}

type envelope[T any] struct {
	msg T
	ctx context.Context
}

type actor[T any] struct {
	mailbox chan envelope[T] // actor channel which must be closed on reader side
	done    <-chan struct{}  // we only need this as read channel
}

// Send sends an envelope to the actor's mailbox
func (r *actor[T]) Send(ctx context.Context, msg T) (err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			if err, ok = r.(error); ok {
				err = errors.Join(errors.New("failed to send envelope"), err)
			}
		}
	}()

	select {
	case <-ctx.Done():
		return errors.New("context canceled while sending envelope")
	case r.mailbox <- envelope[T]{
		msg: msg,
		ctx: ctx,
	}:
	}
	return nil
}

func (r *actor[T]) Wait() {
	<-r.done
}
