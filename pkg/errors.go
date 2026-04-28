package fctl

import "fmt"

type ErrUnauthorized struct {
}

func (e ErrUnauthorized) Is(target error) bool {
	_, ok := target.(*ErrUnauthorized)
	return ok
}

func (e *ErrUnauthorized) Error() string {
	return "unauthorized access"
}

func newErrUnauthorized() error {
	return &ErrUnauthorized{}
}

type ErrForbidden struct {
}

func (e ErrForbidden) Is(target error) bool {
	_, ok := target.(*ErrForbidden)
	return ok
}

func (e *ErrForbidden) Error() string {
	return "forbidden access"
}

func newErrForbidden() error {
	return &ErrForbidden{}
}

type UnexpectedStatusCodeError struct {
	StatusCode int
}

func (e UnexpectedStatusCodeError) Is(target error) bool {
	_, ok := target.(*UnexpectedStatusCodeError)
	return ok
}

func (e *UnexpectedStatusCodeError) Error() string {
	return fmt.Sprintf("unexpected status code: %d", e.StatusCode)
}

func newUnexpectedStatusCodeError(statusCode int) error {
	return &UnexpectedStatusCodeError{StatusCode: statusCode}
}
