package domain

import (
	"errors"
	"fmt"
)

var (
	// ErrInternalServerError will throw if any the Internal Server Error happen
	ErrInternalServerError = errors.New("internal Server Error")
	// ErrNotFound will throw if the requested item is not exists
	ErrNotFound = errors.New("your requested Item is not found")
	// ErrConflict will throw if the current action already exists
	ErrConflict = errors.New("your Item already exist")
	// ErrBadParamInput will throw if the given request-body or params is not valid
	ErrBadParamInput = errors.New("given Param is not valid")
	// ErrForbidden will throw if status code is 403
	ErrForbidden = errors.New("permission denied")
	// ErrEmptyBotToken will throw if user doesn't have Bot Access Token
	ErrEmptyBotToken = errors.New("bot Access Token is empty")
	// ErrUnknownReportType will throw if they try to get something different from the known report types (such as report or dashboard)
	ErrUnknownReportType = errors.New("unknown report type")
	// ErrInvalidType is returned when an unsupported concrete type is encountered during a type assertion.
	ErrInvalidType = errors.New("invalid type")
	// ErrNotUpdated is returned due to an unsuccessful update operation.
	ErrNotUpdated = errors.New("couldn't update")
	// ErrUnexpectedContentType will throw if content type is unexpected
	ErrUnexpectedContentType = func(contentType interface{}) error { return fmt.Errorf("unexpected content type: %v", contentType) }
	// ErrUnexpectedStatusCode will throw if status code is unexpected
	ErrUnexpectedStatusCode = func(status int) error { return fmt.Errorf("unexpected status code: %v", status) }
)
