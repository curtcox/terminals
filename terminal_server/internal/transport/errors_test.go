package transport

import "testing"

func TestErrorCodeFor(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{ErrInvalidClientMessage, ErrorCodeInvalidClientMessage},
		{ErrInvalidCommandAction, ErrorCodeInvalidCommandAction},
		{ErrInvalidCommandKind, ErrorCodeInvalidCommandKind},
		{ErrMissingCommandIntent, ErrorCodeMissingIntent},
		{ErrMissingCommandText, ErrorCodeMissingText},
		{ErrMissingCommandDeviceID, ErrorCodeMissingDeviceID},
	}
	for _, tc := range cases {
		if got := errorCodeFor(tc.err); got != tc.want {
			t.Fatalf("errorCodeFor(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}
