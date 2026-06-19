package domain

import (
	"errors"
	"fmt"
)

// ErrBatchNotFound is returned by the Store when a requested batch ID does not exist.
var ErrBatchNotFound = errors.New("batch not found")

// ValidationError represents an error during input validation.
// It implements the standard error interface.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface for ValidationError.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed on field '%s': %s", e.Field, e.Message)
}
