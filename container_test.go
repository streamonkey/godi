package di

import (
	"context"
	"errors"
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
