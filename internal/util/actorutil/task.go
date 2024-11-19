package actorutil

import (
	"errors"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/primetalk/goio/io"
)

type SafeBackgroundTask[T any] struct {
	ctx       actor.Context
	fn        func() (*T, error)
	timeout   *time.Duration
	onError   func(error)
	recover   func(error) T
	onSuccess func(T)
}

func NewBackgroundTask[T any](ctx actor.Context, fn func() (*T, error)) *SafeBackgroundTask[T] {
	return &SafeBackgroundTask[T]{
		ctx: ctx,
		fn:  fn,
	}
}

func NewBackgroundTaskNoError[T any](ctx actor.Context, fn func() *T) *SafeBackgroundTask[T] {
	return &SafeBackgroundTask[T]{
		ctx: ctx,
		fn: func() (*T, error) {
			return fn(), nil
		},
	}
}

func NewBackgroundTaskErr(ctx actor.Context, fn func() error) *SafeBackgroundTask[any] {
	return &SafeBackgroundTask[any]{
		ctx: ctx,
		fn: func() (*any, error) {
			return nil, fn()
		},
	}
}

func (t *SafeBackgroundTask[T]) WithTimeout(timeout time.Duration) *SafeBackgroundTask[T] {
	t.timeout = &timeout
	return t
}

func (t *SafeBackgroundTask[T]) OnError(fn func(error)) *SafeBackgroundTask[T] {
	t.onError = fn
	return t
}

func (t *SafeBackgroundTask[T]) Recover(fn func(error) T) *SafeBackgroundTask[T] {
	t.recover = fn
	return t
}

func (t *SafeBackgroundTask[T]) OnSuccess(fn func(T)) *SafeBackgroundTask[T] {
	t.onSuccess = fn
	return t
}

func (t *SafeBackgroundTask[T]) PipeTo(actor *actor.PID) {
	t.onSuccess = func(value T) {
		t.ctx.Send(actor, value)
	}
	t.Run()
}

func (t *SafeBackgroundTask[T]) Run() {
	bgFn := io.Eval(t.fn)
	bg := io.Map(bgFn, func(a *T) T {
		if a != nil {
			return *a
		}
		panic(errors.New("result is nil"))
	})
	if t.timeout != nil {
		bg = io.WithTimeout[T](*t.timeout)(bg)
	}
	result := io.RunSync(bg)
	var finalValue *T
	if result.Error != nil {
		if t.recover != nil {
			a := t.recover(result.Error)
			finalValue = &a
		} else if t.onError != nil {
			t.onError(result.Error)
			return
		}
	}
	if finalValue == nil {
		finalValue = &result.Value
	}

	if t.onSuccess != nil {
		t.onSuccess(result.Value)
	}
}

func MapBackgroundTask[T, T2 any](bgt *SafeBackgroundTask[T], mapFn func(*T) *T2) *SafeBackgroundTask[T2] {
	newFn := func() (*T2, error) {
		r, err := bgt.fn()
		if err != nil {
			return nil, err
		}
		return mapFn(r), nil
	}
	return &SafeBackgroundTask[T2]{
		ctx: bgt.ctx,
		fn:  newFn,
	}
}
