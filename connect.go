package websocket

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// Connection represents a WebSocket connection.
//
// It is an interface that abstracts the underlying connection
// implementation, allowing for different backends (e.g., native WebSocket,
// HTTP/2, etc.) to be used interchangeably.
type Connection interface {
	Close() error
}

// Connect establishes a WebSocket connection to the given URI.
//
// The URI must be a valid WebSocket URI, as defined in RFC6455 Section 3.
//
// Example usage:
//
//	conn, err := websocket.Connect(context.TODO(), url.URL{
//		Scheme: websocket.SchemeWS.String(),
//		Host:   "localhost:8080",
//	}.String())
func Connect(ctx context.Context, uri string) (Connection, error) {
	err := validateWebsocketURI(uri)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// Scheme represents the scheme of a WebSocket URI.
type Scheme string

const (
	SchemeWS  Scheme = "ws"
	SchemeWSS Scheme = "wss"
)

func (s Scheme) String() string {
	return string(s)
}

// validation according to RFC6455 Section 3.
func validateWebsocketURI(uri string) error {
	u, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidURI, err)
	}

	if u.Port() != "" {
		port, err := strconv.Atoi(u.Port())
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("%w: invalid port number", ErrInvalidURI)
		}
	}

	if u.Scheme != string(SchemeWS) && u.Scheme != string(SchemeWSS) {
		return fmt.Errorf("%w: %s, must be '%s' or '%s'", ErrInvalidURI, u.Scheme, SchemeWSS, SchemeWS)
	}

	if u.Host == "" {
		return fmt.Errorf("%w: no host specified", ErrInvalidURI)
	}

	if u.Fragment != "" {
		return fmt.Errorf("%w: fragment identifiers are not allowed in WebSocket URIs", ErrInvalidURI)
	}

	return nil
}
