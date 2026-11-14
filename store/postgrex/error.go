package postgrex

import (
	"errors"
)

var (
	// ErrQueryPreparation indicates that a statement could not be prepared or found.
	ErrQueryPreparation = errors.New("query preparation error")
	// ErrConcurrentModification indicates a conflicting concurrent modification.
	ErrConcurrentModification = errors.New("concurrent modification detected")
)
