package di

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

type (
	Container[C any] struct {
		config    C
		factories map[string]func(context.Context, *Container[C]) (any, error)
		services  map[string]any
		sync.Mutex
		frozen atomic.Bool
	}

	ContainerBuilder[C any] struct {
		container *Container[C]
		frozen    atomic.Bool
	}

	Factory[T, C any] func(context.Context, *Container[C]) (T, error)

	ServiceID[T any] string
)

func New[C any](config C) *ContainerBuilder[C] {

	c := &Container[C]{
		config:    config,
		factories: make(map[string]func(context.Context, *Container[C]) (any, error)),
		services:  make(map[string]any),
	}

	return &ContainerBuilder[C]{
		container: c,
	}
}

func (c *ContainerBuilder[C]) isFrozen() bool {
	return c.frozen.Load()
}

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

func Register[T, C any](cc *ContainerBuilder[C], id ServiceID[T], fac Factory[T, C]) error {
	if cc.isFrozen() {
		var v T
		return fmt.Errorf("cannot register %s[%T]: %w", id, v, ErrContainerSealed)
	}

	cc.container.Lock()
	defer cc.container.Unlock()

	cc.container.factories[string(id)] = func(ctx context.Context, c *Container[C]) (any, error) {
		return fac(ctx, c)
	}
	return nil
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
	s, ok := c.services[string(id)]
	if ok {
		return s, nil
	}

	f, ok := c.factories[string(id)]
	if !ok {
		return nil,
			fmt.Errorf("no factory is registered for %+v", id)
	}

	srv, err := f(ctx, c)
	if err != nil {
		return nil, err
	}

	c.services[string(id)] = srv
	return srv, nil
}
