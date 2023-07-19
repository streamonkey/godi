package xerror

import (
	"errors"
	"strings"
	"sync"
)

type (
	Stack struct {
		errors *stacked
		mu     sync.Mutex
	}
	stacked struct {
		err  error
		prev error
	}
)

var nillStacked *stacked

func (e *stacked) Error() string {
	if e.prev == nil {
		return e.err.Error()

	}

	return strings.Join([]string{e.err.Error(), e.prev.Error()}, "\n")
}

// Join creates a reversed stack maintaining the original order of errors
func Join(errs ...error) error {
	s := &Stack{}
	l := len(errs)
	for i := l; i > 0; i-- {
		s.add(errs[i-1])
	}

	return s
}

func (e *stacked) Err() error {
	return e.err
}

func (e *stacked) Unwrap() error {
	if e.prev == nil || e.prev == nillStacked {
		return nil
	}
	return e.prev
}

func (s *Stack) Is(target error) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	err = s.errors

	for {
		if err == nil {
			return false
		}
		v, ok := err.(*stacked)
		if !ok {
			return false
		}

		if v == nil || v == nillStacked {
			return false
		}
		switch true {
		case errors.Is(v.err, target):
			return true

		case errors.Is(v.Unwrap(), target):
			return true
		default:
			err = v.Unwrap()
		}
	}
}

func (s *Stack) Next() *stacked {
	return s.errors
}

func (s *Stack) Add(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.add(err)
}

func (s *Stack) add(err error) {
	if err == nil {
		return
	}
	var prev error
	if s.errors != nil && s.errors != nillStacked {
		prev = s.errors
	}
	st := &stacked{err: err, prev: prev}
	s.errors = st
}

func (s *Stack) Error() (msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.errors == nil {
		return
	}

	return s.errors.Error()
}

func (s *Stack) Trace() (msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.errors == nil {
		return
	}

	var nilerr *stacked
	err := s.errors
	for {
		if err == nilerr || err == nil {
			break
		}

		msg += err.Error()
		unwrapped := err.Unwrap()
		if unwrapped == nil {
			break
		}
		uerr, ok := unwrapped.(*stacked)
		if !ok {
			break
		}
		err = uerr
		if err == nilerr || err == nil {
			break
		}
		msg += "\n"

	}

	return msg
}
