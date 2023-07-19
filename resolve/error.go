package resolve

import (
	"errors"
	"fmt"

	di "github.com/streamonkey/godi"
	"github.com/streamonkey/godi/internal/xerrors"
)

type (
	serviceErr struct {
		msg  string
		errs error
	}
)

func (s *serviceErr) Error() string {
	if s.errs == nil {
		return fmt.Sprintf("error creating services [%s]", s.msg)
	}

	return fmt.Sprintf("error creating services [%s]:\n%s", s.msg, s.errs)
}

func (s *serviceErr) Unwrap() error {
	return s.errs
}

func Error[T any](srv di.ServiceID[T], errors ...error) error {
	r := &serviceErr{msg: string(srv)}
	if len(errors) == 0 {
		return r
	}

	r.errs = xerrors.Join(errors...)
	return r
}

func IsServiceError(err error) bool {
	switch err.(type) {
	case nil:
		return false
	case *serviceErr:
		return true
	default:
		return IsServiceError(errors.Unwrap(err))
	}
}
