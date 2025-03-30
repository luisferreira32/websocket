package websocket

import (
	"errors"
)

var (
	// ErrInvalidURI indicates that the URI is not valid for WebSocket.
	//
	// Validation is done according to the normative Section 3. of RFC6455.
	// The URI must have a valid scheme (ws or wss), a host, and no fragment identifiers.
	ErrInvalidURI = errors.New("invalid URI")
)
