package di

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

type (
	Config struct {
		A string
		B string
	}

	A struct {
		AName string
	}

	B struct {
		BName string
	}
)

func TestContainerRegister(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	cb := New(&Config{A: "a", B: "b"})

	sidA := ServiceID[*A]("service.a")
	sidB := ServiceID[*B]("service.b")

	var err error
	err = Register(cb, sidA, func(ctx context.Context, cc *Container[*Config]) (*A, error) {
		return &A{AName: cc.Config().A}, nil
	})
	assert.NoError(t, err)

	err = Register(cb, sidB, func(ctx context.Context, cc *Container[*Config]) (*B, error) {
		return &B{BName: cc.Config().B}, nil
	})
	assert.NoError(t, err)

	c, err := cb.Build()
	assert.NoError(t, err)

	a, err := Get(ctx, c, sidA)
	assert.NoError(t, err)
	b, err := Get(ctx, c, sidB)
	assert.NoError(t, err)

	assert.Equal(t, c.Config().A, a.AName)
	assert.Equal(t, c.Config().B, b.BName)
}

func TestConcurrentResolve(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	cb := New(&Config{A: "a", B: "b"})

	sidA := ServiceID[*A]("service.a")
	var err error
	err = Register(cb, sidA, func(ctx context.Context, cc *Container[*Config]) (*A, error) {
		return &A{AName: cc.Config().A}, nil
	})
	assert.NoError(t, err)

	assert.NoError(t, err)

	c, err := cb.Build()
	assert.NoError(t, err)

	wg := &sync.WaitGroup{}

	resultc := make(chan *A, 10)
	errc := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := Get(ctx, c, sidA)
			if err != nil {
				errc <- err
				return
			}

			resultc <- res
		}()
	}

	go func() {
		wg.Wait()
		close(resultc)
		close(errc)
	}()

	res, err := Get(ctx, c, sidA)
	assert.NoError(t, err)

	for a := range resultc {
		assert.Same(t, res, a)
	}

	for err := range errc {
		t.Error(err)
	}
}

// TestConcurrentNestedResolve exercises the case the plain TestConcurrentResolve
// misses: a factory that resolves a dependency via Get while being resolved
// concurrently. It guards against the self-deadlock that occurs when the
// container lock is held across factory execution, and asserts each service is
// constructed exactly once (single-flight) and shared.
func TestConcurrentNestedResolve(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	cb := New(&Config{A: "a", B: "b"})

	var aBuilds, bBuilds atomic.Int64
	sidA := ServiceID[*A]("service.a")
	sidB := ServiceID[*B]("service.b")

	assert.NoError(t, Register(cb, sidB, func(ctx context.Context, cc *Container[*Config]) (*B, error) {
		bBuilds.Add(1)
		return &B{BName: cc.Config().B}, nil
	}))
	// A depends on B: nested Get inside A's factory.
	assert.NoError(t, Register(cb, sidA, func(ctx context.Context, cc *Container[*Config]) (*A, error) {
		aBuilds.Add(1)
		b, err := Get(ctx, cc, sidB)
		if err != nil {
			return nil, err
		}
		return &A{AName: cc.Config().A + b.BName}, nil
	}))

	c, err := cb.Build()
	assert.NoError(t, err)

	const n = 50
	wg := &sync.WaitGroup{}
	results := make([]*A, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = Get(ctx, c, sidA)
		}(i)
	}
	wg.Wait()

	first := results[0]
	for i := 0; i < n; i++ {
		assert.NoError(t, errs[i])
		assert.Same(t, first, results[i]) // all share the one singleton
	}
	assert.Equal(t, int64(1), aBuilds.Load(), "A built exactly once")
	assert.Equal(t, int64(1), bBuilds.Load(), "B built exactly once")
}

// TestResolveErrorNotCached asserts a factory error is not memoized: a later Get
// retries the factory. xstream's startup Retry relies on this behavior.
func TestResolveErrorNotCached(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	cb := New(&Config{A: "a", B: "b"})

	var calls atomic.Int64
	sidA := ServiceID[*A]("service.a")
	assert.NoError(t, Register(cb, sidA, func(ctx context.Context, cc *Container[*Config]) (*A, error) {
		if calls.Add(1) == 1 {
			return nil, errors.New("transient failure")
		}
		return &A{AName: "ok"}, nil
	}))

	c, err := cb.Build()
	assert.NoError(t, err)

	_, err = Get(ctx, c, sidA)
	assert.Error(t, err) // first attempt fails and is NOT cached

	a, err := Get(ctx, c, sidA)
	assert.NoError(t, err) // retry succeeds
	assert.Equal(t, "ok", a.AName)
	assert.Equal(t, int64(2), calls.Load())
}

func TestContainerSeal(t *testing.T) {
	cb := New(&Config{A: "a", B: "b"})

	sidA := ServiceID[*A]("service.a")
	sidB := ServiceID[*B]("service.b")

	var err error
	err = Register(cb, sidA, func(ctx context.Context, cc *Container[*Config]) (*A, error) {
		return &A{AName: cc.Config().A}, nil
	})
	assert.NoError(t, err)

	cb.Build()

	err = Register(cb, sidB, func(ctx context.Context, cc *Container[*Config]) (*B, error) {
		return &B{BName: cc.Config().B}, nil
	})

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrContainerSealed))
}
