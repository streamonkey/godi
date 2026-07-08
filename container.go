package di

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

type (
	Container[C any] struct {
		config C
		// immutable once container is built. No lock needed. Each factory is a singleflight and thus
		// holds its own lock for service construction.
		factories map[string]func(context.Context, *Container[C]) (any, error)
	}

	ContainerBuilder[C any] struct {
		container *Container[C]
		mu        sync.Mutex
		frozen    atomic.Bool
	}

	Factory[T, C any] func(context.Context, *Container[C]) (T, error)

	ServiceID[T any] string
)

// New initializes a new ContainerBuilder
func New[C any](config C) *ContainerBuilder[C] {

	c := &Container[C]{
		config:    config,
		factories: make(map[string]func(context.Context, *Container[C]) (any, error)),
	}

	return &ContainerBuilder[C]{
		container: c,
	}
}

func (c *ContainerBuilder[C]) isFrozen() bool {
	return c.frozen.Load()
}

// Build wil build the Dependency Container
// If the ContainerBuilder is already build, it will return an error, otherwhise it'll return the container
func (c *ContainerBuilder[C]) Build() (*Container[C], error) {
	if c.isFrozen() {
		return nil, ErrContainerSealed
	}
	c.frozen.Store(true)

	return c.container, nil
}

func (c *Container[C]) Config() C {
	return c.config
}

// Inject injects a final type into the container.
// This is a convenient function as it just wraps the final type into a factory function
func Inject[T, C any](cc *ContainerBuilder[C], id ServiceID[T], obj T) error {
	return Register(cc, id, func(ctx context.Context, c *Container[C]) (T, error) {
		return obj, nil
	})
}

func Register[T, C any](cc *ContainerBuilder[C], id ServiceID[T], fac Factory[T, C]) error {
	if cc.isFrozen() {
		var v T
		return fmt.Errorf("cannot register %s[%T]: %w", id, v, ErrContainerSealed)
	}

	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.container.factories[string(id)] = singleFlight(func(ctx context.Context, c *Container[C]) (any, error) {
		return fac(ctx, c)
	})
	return nil
}

// singleFlight wraps a factory so it is invoked at most once successfully
//
// it can fail multiple times, so service construction recovery is possible.
//
// Concurrent calls to the factory will block until the first call returns.
func singleFlight[C any](fac func(context.Context, *Container[C]) (any, error)) func(context.Context, *Container[C]) (any, error) {
	var (
		mu   sync.Mutex
		done bool
		val  any
	)
	return func(ctx context.Context, c *Container[C]) (any, error) {
		mu.Lock()
		defer mu.Unlock()
		if done {
			return val, nil
		}
		v, err := fac(ctx, c)
		if err != nil {
			return nil, err
		}
		val, done = v, true
		return val, nil
	}
}

func Get[T, C any](ctx context.Context, cc *Container[C], service ServiceID[T]) (T, error) {
	return get(ctx, cc, service)
}

func get[T, C any](ctx context.Context, c *Container[C], service ServiceID[T]) (T, error) {
	var tt T
	s, err := fromContainer(ctx, c, service)
	if err != nil {
		return tt, err
	}

	srv, ok := s.(T)
	if !ok {
		return tt, fmt.Errorf("type %T is not %T", s, service)
	}

	return srv, nil
}

func fromContainer[T, C any](ctx context.Context, c *Container[C], id ServiceID[T]) (any, error) {
	f, ok := c.factories[string(id)]
	if !ok {
		return nil,
			fmt.Errorf("no factory is registered for %+v", id)
	}

	return f(ctx, c)
}
