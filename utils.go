package mxlgolang

import "errors"

func joinErrors(err error, msg string) error {
	return errors.Join(err, errors.New(msg))
}
