package xerror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStackError(t *testing.T) {
	s := &Stack{}
	s.Add(errors.New("A"))
	s.Add(errors.New("B"))
	s.Add(errors.New("C"))

	assert.Equal(t, "C\nB\nA", s.Error())
}
