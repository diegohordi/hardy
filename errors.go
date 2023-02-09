package hardy

import "encoding/json"

// ErrorCode is the type of well-known error codes.
type ErrorCode string

const (

	// ErrInvalidClientConfiguration is the error returned when some client configuration is invalid.
	ErrInvalidClientConfiguration ErrorCode = "invalid_configuration_error"

	// ErrNoDebuggerFound is the error returned when the debug mode was enabled but debugger was given.
	ErrNoDebuggerFound ErrorCode = "no_debugger_found_error"

	// ErrNoHTTPClientFound is the error returned when no HTTP Client was given.
	ErrNoHTTPClientFound ErrorCode = "no_http_client_found_error"

	// ErrNoReaderFuncFound is the error returned when no ReaderFunc was given.
	ErrNoReaderFuncFound ErrorCode = "no_reader_func_found_error"

	// ErrMaxRetriesReached is the error returned when the max allowed retries were reached.
	ErrMaxRetriesReached ErrorCode = "max_retries_reached_error"

	// ErrUnexpected is the error returned when no one of the previous errors match.
	ErrUnexpected ErrorCode = "unexpected_error"
)

// Error returns the string representation of the given error.
func (e ErrorCode) Error() string {
	return string(e)
}

// Error represents the structured errors returned by the client.
type Error struct {

	// ErrorCode is a well-known error code.
	ErrorCode ErrorCode `json:"error_code"`

	// HTTPStatusCode is the equivalent HTTP status code.
	HTTPStatusCode int `json:"status_code"`

	// Message is the user-friendly error message.
	Message string `json:"message"`

	// cause is the error that cause this error.
	cause error
}

// Error returns the string representation of the given error as JSON.
func (e Error) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

// Is checks if the given target error equals to this error code
func (e Error) Is(tgt error) bool {
	return e.ErrorCode == tgt
}

// errorOption defines an error builder option
type errorOption func(err *Error)

// newError builds a new Error.
func newError(errorCode ErrorCode, opts ...errorOption) Error {
	err := &Error{
		ErrorCode: errorCode,
	}
	for i := range opts {
		opts[i](err)
	}
	return *err
}

// withCause sets the cause of the error.
func withCause(cause error) errorOption {
	return func(err *Error) {
		err.cause = cause
		if err.Message == "" && err.cause != nil {
			err.Message = err.cause.Error()
		}
	}
}

// withHTTPStatusCode sets the equivalent HTTP status code of the error.
func withHTTPStatusCode(statusCode int) errorOption {
	return func(err *Error) {
		err.HTTPStatusCode = statusCode
	}
}
