//go:build go1.20
// +build go1.20

package xerrors

import "errors"

func Join(errs ...error) error {
	return errors.Join(errs...)
}
