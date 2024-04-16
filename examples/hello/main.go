package main

import (
	"context"
	"github.com/p357k4/pactors/typed"
	"log/slog"
	"time"
)

// myActor is a concrete implementation of the Receiver interface
type myActor[T any] struct{}

// NewMyReceive creates a new instance of myActor
func NewMyReceive[T any]() typed.Receive[T] {
	return myActor[T]{}.Receive
}

// Receive implements the Receiver interface and receives messages
func (a myActor[T]) Receive(ctx context.Context, msg T) error {
	slog.InfoContext(ctx, "Custom logic", slog.Any("msg", msg))
	return nil
}

type functionActor[T any] struct {
	f func(ctx context.Context, msg T) error
}

func (f functionActor[T]) Receive(ctx context.Context, msg T) error {
	return f.f(ctx, msg)
}

func NewActorFromFunction[T any](f func(ctx context.Context, msg T) error) typed.Receive[T] {
	return functionActor[T]{f: f}.Receive
}

func main() {
	// create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// create system
	system := typed.NewSystem()

	// create actors
	actorStringRef := typed.Start(ctx, system, func() (typed.Receive[string], error) {
		return NewMyReceive[string](), nil
	})
	intActorRef := typed.Start(ctx, system, func() (typed.Receive[int], error) {
		return NewMyReceive[int](), nil
	})

	panicActor := NewActorFromFunction[string](func(ctx context.Context, msg string) error {
		panic("total panic")
	})
	panicActorRef := typed.Start(ctx, system, func() (typed.Receive[string], error) {
		return panicActor, nil
	})

	// Send string messages with the custom logic
	actorStringRef.Send(ctx, "Hello")
	actorStringRef.Send(ctx, "World")

	// Send integer messages without custom logic (uses default logic)
	intActorRef.Send(ctx, 42)
	intActorRef.Send(ctx, 100)
	panicActorRef.Send(ctx, "nothing else matters")
	time.Sleep(100 * time.Millisecond)
	panicActorRef.Send(ctx, "nothing else matters")
	time.Sleep(2 * time.Second)

	cancel()

	system.Wait()
}
