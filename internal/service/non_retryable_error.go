package service

import "errors"

// NonRetryableError identifies a poison event that retrying cannot repair.
// Code is safe to log; Cause may contain producer data and must not be logged
// by the transport handler.
type NonRetryableError struct {
	Code  string
	Cause error
}

func (e *NonRetryableError) Error() string {
	return e.Code + ": " + e.Cause.Error()
}

func (e *NonRetryableError) Unwrap() error {
	return e.Cause
}

func nonRetryable(code string, cause error) error {
	return &NonRetryableError{Code: code, Cause: cause}
}

func NonRetryableCode(err error) (string, bool) {
	var target *NonRetryableError
	if !errors.As(err, &target) {
		return "", false
	}
	return target.Code, true
}
