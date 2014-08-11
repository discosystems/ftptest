package ftptest

import (
	"errors"
)

var (
	ErrQuitRequest = errors.New("Client requested to close the connection")

	ErrNotFound      = errors.New("File not found")
	ErrAlreadyExists = errors.New("File already exists")
	ErrNoParent      = errors.New("Parent not found")

	ErrProtocolNotSupported  = errors.New("Network protocol not supported, use (1,2)")
	ErrInvalidMessage        = errors.New("Invalid message")
	ErrDataSocketUnavailable = errors.New("Data socket unavailable")
)
