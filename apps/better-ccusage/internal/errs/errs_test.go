package errs

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	for _, e := range []error{ErrNoData, ErrUnknownModel, ErrInvalidJSON, ErrConfigNotFound, ErrIncompatibleMode, ErrInvalidMode} {
		wrapped := fmt.Errorf("context: %w", e)
		if !errors.Is(wrapped, e) {
			t.Errorf("errors.Is(%v, %v) = false, want true", wrapped, e)
		}
	}
}
