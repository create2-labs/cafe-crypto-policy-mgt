package persistence

import "errors"

var (
	// ErrPersistenceUnavailable is returned when cafe-persistence is unreachable or returns 503.
	ErrPersistenceUnavailable = errors.New("persistence unavailable")
	// ErrUnsupportedStoreOperation is returned when the active store cannot perform a legacy operation.
	ErrUnsupportedStoreOperation = errors.New("operation not supported by persistence store")
)
