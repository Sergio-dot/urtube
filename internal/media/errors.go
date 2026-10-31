package media

import "errors"

var (
	// ErrInvalidInput matches every validation error (see inputError).
	ErrInvalidInput = errors.New("invalid input")
	// ErrNotFound is returned when a download ID is unknown or no longer active.
	ErrNotFound = errors.New("download not found")
	// ErrNoResults is returned when a search matches nothing.
	ErrNoResults = errors.New("no results found")
	// ErrDraining is returned by Start once the service is shutting down.
	ErrDraining = errors.New("server is shutting down")
)

// inputError is a validation error whose message is safe to show to clients.
// Every inputError matches ErrInvalidInput with errors.Is.
type inputError string

func (e inputError) Error() string { return string(e) }

func (e inputError) Is(target error) bool { return target == ErrInvalidInput }

var (
	ErrQueryRequired    error = inputError("search query is required")
	ErrURLRequired      error = inputError("url is required")
	ErrInvalidURL       error = inputError("url must be an absolute http or https URL")
	ErrInvalidMediaType error = inputError("options.type must be one of: video, audio")
	ErrInvalidFormat    error = inputError("unsupported format")
)
