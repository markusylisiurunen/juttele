package util

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"
)

func SafeGo[T any](
	ctx context.Context,
	fn func(ctx context.Context, vs chan<- T, errs chan<- error),
) (<-chan Result[T], context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	out := make(chan Result[T], 1)
	go func() {
		defer close(out)
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}
				select {
				case out <- Result[T]{Err: fmt.Errorf("panic: %w; stack: %s", err, stack)}:
				case <-time.After(5 * time.Second):
				}
			}
		}()
		vs, errs := make(chan T), make(chan error)
		go func() {
			defer close(vs)
			defer close(errs)
			defer func() {
				if r := recover(); r != nil {
					stack := debug.Stack()
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}
					select {
					case errs <- fmt.Errorf("panic: %w; stack: %s", err, stack):
					case <-time.After(5 * time.Second):
					}
				}
			}()
			fn(ctx, vs, errs)
		}()
		var vClosed, errClosed bool
		for !vClosed || !errClosed {
			select {
			case v, ok := <-vs:
				if !ok {
					vClosed = true
					continue
				}
				out <- Result[T]{Val: v}
			case err, ok := <-errs:
				if !ok {
					errClosed = true
					continue
				}
				out <- Result[T]{Err: err}
			}
		}
	}()
	return out, cancel
}
