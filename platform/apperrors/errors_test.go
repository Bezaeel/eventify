package apperrors_test

import (
	"errors"
	"fmt"
	"testing"

	"eventify/platform/apperrors"
)

func TestKindOf(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want apperrors.Kind
	}{
		{"nil-safe default", errors.New("plain"), apperrors.Internal},
		{"direct", apperrors.New(apperrors.NotFound, "missing"), apperrors.NotFound},
		{"wrapped by apperrors", apperrors.Wrap(apperrors.Conflict, "dup", errors.New("23505")), apperrors.Conflict},
		{
			"wrapped by fmt.Errorf preserves the Kind",
			fmt.Errorf("layer: %w", apperrors.New(apperrors.Forbidden, "nope")),
			apperrors.Forbidden,
		},
		{
			"an error from another package is Internal",
			fmt.Errorf("layer: %w", errors.New("driver exploded")),
			apperrors.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apperrors.KindOf(tt.err); got != tt.want {
				t.Fatalf("KindOf() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestError_UnwrapsToItsCause(t *testing.T) {
	cause := errors.New("root cause")
	err := apperrors.Wrap(apperrors.Internal, "context", cause)

	if !errors.Is(err, cause) {
		t.Fatal("errors.Is must find the wrapped cause")
	}
	if got, want := err.Error(), "context: root cause"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestNew_HasNoCause(t *testing.T) {
	err := apperrors.New(apperrors.Invalid, "bad input")
	if got, want := err.Error(), "bad input"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if errors.Unwrap(err) != nil {
		t.Fatal("New must not wrap anything")
	}
}
