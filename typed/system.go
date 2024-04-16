package typed

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

var ErrActorPanicked = errors.New("actor panicked")
var ErrMailboxClosed = errors.New("mailbox closed")

type system struct {
	wg sync.WaitGroup
}

func (s *system) waitGroup() *sync.WaitGroup {
	return &s.wg
}

func (s *system) Wait() {
	s.wg.Wait()
}

func NewSystem() System {
	return &system{}
}

// Start starts the actor by running a goroutine that listens for messages on the channel
func Start[T any](ctx context.Context, system System, spawn Spawn[T]) Mailbox[T] {
	ref := mailbox[T]{channel: make(chan envelope[T], 1)}
	go func() {
		if err := run(ctx, system.waitGroup(), spawn, ref); err != nil {
			slog.ErrorContext(ctx, "forever loop errored", slog.Any("error", err))
		}
	}()

	return ref
}

func run[T any](ctx context.Context, wg *sync.WaitGroup, spawn Spawn[T], ref mailbox[T]) (err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "forever loop panicked", slog.Any("recover", r))
			err = errors.New("forever loop panicked")
		}
	}()

	defer func() {
		if err := ref.Close(); err != nil {
			slog.ErrorContext(ctx, "mailbox close errored", slog.Any("error", err))
		}
	}()

	wg.Add(1)
	defer wg.Done()

	for {
		receiver, err := spawn()
		if err != nil {
			return errors.Join(errors.New("spawn function failed"), err)
		}
		err = withoutTimeout[T](ctx, ref.channel, receiver)
		switch {
		case errors.Is(err, ErrActorPanicked):
			// restart an actor
			slog.ErrorContext(ctx, "actor panicked - restart", slog.Any("error", err))
			break
		case !errors.Is(err, nil):
			// stop an actor
			return errors.Join(errors.New("actor errored - stop"), err)
		}
	}
}

func withoutTimeout[T any](ctx context.Context, mailbox chan envelope[T], receive Receive[T]) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-mailbox:
			if !ok {
				return ErrMailboxClosed
			}

			if err := safeReceive(receive, msg); err != nil {
				return err
			}
		}
	}
}

func safeReceive[T any](receive Receive[T], msg envelope[T]) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ErrActorPanicked
		}
	}()

	return receive(msg.ctx, msg.msg)
}
