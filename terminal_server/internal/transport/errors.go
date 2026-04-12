package transport

import "errors"

// Stable machine-readable error codes for control stream responses.
const (
	ErrorCodeInvalidClientMessage = "invalid_client_message"
	ErrorCodeInvalidCommandAction = "invalid_command_action"
	ErrorCodeInvalidCommandKind   = "invalid_command_kind"
	ErrorCodeMissingIntent        = "missing_command_intent"
	ErrorCodeMissingText          = "missing_command_text"
	ErrorCodeMissingDeviceID      = "missing_command_device_id"
	ErrorCodeProtocolViolation    = "protocol_violation"
	ErrorCodeUnknown              = "unknown_error"
)

func errorCodeFor(err error) string {
	switch {
	case errors.Is(err, ErrInvalidClientMessage):
		return ErrorCodeInvalidClientMessage
	case errors.Is(err, ErrInvalidCommandAction):
		return ErrorCodeInvalidCommandAction
	case errors.Is(err, ErrInvalidCommandKind):
		return ErrorCodeInvalidCommandKind
	case errors.Is(err, ErrMissingCommandIntent):
		return ErrorCodeMissingIntent
	case errors.Is(err, ErrMissingCommandText):
		return ErrorCodeMissingText
	case errors.Is(err, ErrMissingCommandDeviceID):
		return ErrorCodeMissingDeviceID
	default:
		return ErrorCodeUnknown
	}
}
