//go:build go1.20
// +build go1.20

package resolve

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	di "github.com/streamonkey/godi"
)

var (
	ErrTa = errors.New("err 1")
	ErrTb = errors.New("err 2")
)

func TestResolveErrorWorksWithoutPreciedingError(t *testing.T) {
	var service di.ServiceID[struct{}] = "swx"
	assert.Equal(t, Error(service).Error(), "error creating services [swx]")
}

func TestResolveErrorWrapsErrorArgs(t *testing.T) {
	var service di.ServiceID[struct{}] = "swx"
	err := Error(service, errors.New("B"), errors.New("A"))
	assert.Equal(t, "error creating services [swx]:\nB\nA", err.Error())
}

func TestResolveErrorWrapsErrorArgs2(t *testing.T) {
	var service di.ServiceID[struct{}] = "swx"
	err := Error(service, ErrTa, ErrTb)

	assert.True(t, IsServiceError(err))
	assert.ErrorIs(t, err, ErrTa)
	assert.ErrorIs(t, err, ErrTb)
}
