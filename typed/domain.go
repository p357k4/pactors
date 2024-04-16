package typed

import (
	"context"
	"io"
	"sync"
)

type Receive[T any] func(context.Context, T) error

// Spawn creates new instance of a receiver or an error if it is not possible
type Spawn[T any] func() (Receive[T], error)

type System interface {
	Wait()
	waitGroup() *sync.WaitGroup
}

type Mailbox[T any] interface {
	io.Closer
	Send(context.Context, T) error
}
