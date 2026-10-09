// Package errs contains shared errors that cross layers. Conversion to HTTP status is done by the server layer.
package errs

import "errors"

var (
	// ErrUnauthorized is used when not logged in or the token is invalid.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrNotFound is used when the target does not exist or is not owned by the caller. Others' resources are not even revealed to exist.
	ErrNotFound = errors.New("not found")
)
