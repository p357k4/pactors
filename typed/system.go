package typed

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"
	"sync"
)

var ErrActorPanicked = errors.New("actor panicked")

type System interface {
	Wait()
	waitGroup() *sync.WaitGroup
}

type system struct {
	wg *sync.WaitGroup
}

func (s *system) waitGroup() *sync.WaitGroup {
	return s.wg
}

func (s *system) Wait() {
	s.wg.Wait()
}

var _ System = (*system)(nil)

func NewSystem() System {
	return &system{wg: &sync.WaitGroup{}}
}

// Start starts the actor by running a goroutine that listens for messages on the mailbox
func Start[T any](system System, ctx context.Context, spawn func() Receiver[T]) Actor[T] {
	mailbox := make(chan envelope[T])
	done := make(chan struct{})

	go forever(system.waitGroup(), ctx, spawn, mailbox, done)

	return &actor[T]{mailbox: mailbox, done: done}
}

func forever[T any](wg *sync.WaitGroup, ctx context.Context, spawn func() Receiver[T], mailbox chan envelope[T], done chan<- struct{}) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "forever loop panicked", slog.String("stack", string(debug.Stack())))
		}
	}()

	defer close(mailbox)
	defer close(done)

	wg.Add(1)
	defer wg.Done()

	var receiver Receiver[T]
	for {
		select {
		case <-ctx.Done():
			slog.ErrorContext(ctx, "context canceled")
			return
		case msg, ok := <-mailbox:
			if !ok {
				slog.ErrorContext(ctx, "mailbox was closed")
				return
			}
			if receiver == nil {
				receiver = spawn()
			}

			err := func() (err error) {
				defer func() {
					if r := recover(); r != nil {
						err = ErrActorPanicked
					}
				}()
				return receiver.Receive(msg.ctx, msg.msg)
			}()

			switch {
			case errors.Is(err, nil):
				// do nothing - no error
			case errors.Is(err, ErrActorPanicked):
				// receiver panicked - restart it
				slog.ErrorContext(ctx, "receiver panicked - restarting")
				receiver = nil
			default:
				slog.ErrorContext(ctx, "receiver returned error")
				return
			}
		}
	}
}
