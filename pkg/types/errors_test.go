package types

import (
	"errors"
	"testing"
)

func TestErrorsUnwrap(t *testing.T) {
	inner := errors.New("inner")
	cases := []struct {
		name string
		err  error
	}{
		{"CryptoError", &CryptoError{Operation: "decrypt", Err: inner}},
		{"ArchiveError", &ArchiveError{Operation: "unpack", Path: "x", Err: inner}},
		{"ScanError", &ScanError{Path: "x", Err: inner}},
		{"nested", &ArchiveError{Operation: "list", Err: &CryptoError{Operation: "decrypt", Err: inner}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !errors.Is(tc.err, inner) {
				t.Errorf("errors.Is(%v, inner) = false", tc.err)
			}
		})
	}
}
