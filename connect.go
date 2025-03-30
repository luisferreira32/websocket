package websocket

import (
	"context"
	"crypto/rand"
	"crypto/sha1" // nolint:gosec // this is the spefied hash of the WebSocket protocol
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// Connection represents a WebSocket connection.
//
// It is an interface that abstracts the underlying connection
// implementation, allowing for different backends (e.g., native WebSocket,
// HTTP/2, etc.) to be used interchangeably.
type Connection interface {
	Close() error
}

type connection struct {
	conn net.Conn // the underlying tcp/udp connection

	stateConnecting chan struct{} // closed once the connection is established
}

func (c *connection) Close() error {
	return nil
}

var (
	connectionsMu sync.Mutex
	connections   = map[string]*connection{}
)

// Connect establishes a WebSocket connection to the given URI.
//
// The URI must be a valid WebSocket URI, as defined in RFC6455 Section 3.
// The function will block until the connection is established or an error occurs,
// any timeout should be given by the context.
//
// Example usage:
//
//	conn, err := websocket.Connect(context.TODO(), url.URL{
//		Scheme: websocket.SchemeWS.String(),
//		Host:   "localhost:8080",
//	}.String())
func Connect(ctx context.Context, uri string) (Connection, error) {
	validatedURI, err := validateWebsocketURI(uri)
	if err != nil {
		return nil, err
	}

	u, _ := url.Parse(validatedURI)
	host := u.Host

	// TODO: if we get websocket to support HTTP/3, we should use UDP instead of TCP
	// TODO: ctx dependent resolver, and maybe skip the resolve in the Dialer?
	// TODO: use-case if client cannot resolve host due to proxy server?
	addr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %w", err)
	}

	connectionsMu.Lock()
	// > There MUST be no more than one connection in the connecting state
	// > for any IP address and port pair (...) the client MUST serialize them (...)
	if conn, ok := connections[addr.String()]; ok {
		if conn.stateConnecting == nil {
			return nil, fmt.Errorf("%w: stateConnecting channel is missing", ErrOpenIssue)
		}
		select {
		case <-conn.stateConnecting:
			// the easiest way to serialize the connections is to wait for the
			// previous connection to be established
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	connectionsMu.Unlock()

	dialer := &net.Dialer{}
	// TODO: if we get websocket to support HTTP/3, we should use UDP instead of TCP
	// TODO: double check if we need extra steps for /secure/ mode
	conn, err := dialer.DialContext(ctx, "tcp", host)
	defer func() {
		// close the connection if errors occurred
		if conn != nil {
			_ = conn.Close()
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to host: %w", err)
	}

	nonce, err := generateNonce()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to generate nonce: %w", ErrOpenIssue, err)
	}

	// MAYBE: use std lib net/http for the request build?
	// TODO: |Sec-WebSocket-Protocol| header field support
	// TODO: |Sec-WebSocket-Extensions| header field support
	request := fmt.Sprintf(`GET %s HTTP/1.1
Host: %s
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: %s
Sec-WebSocket-Version: 13`,
		u.RequestURI(), u.Host, nonce,
	)

	_, err = conn.Write([]byte(request))
	if err != nil {
		return nil, fmt.Errorf("failed to send handshake request: %w", err)
	}

	response := make([]byte, 4096)
	n, err := conn.Read(response)
	if err != nil {
		return nil, fmt.Errorf("failed to read handshake response: %w", err)
	}

	if !strings.Contains(string(response[:n]), "101 Switching Protocols") {
		return nil, fmt.Errorf("invalid handshake response: %s", string(response[:n]))
	}

	expectedAccept := computeAcceptKey(nonce)
	if !strings.Contains(string(response[:n]), fmt.Sprintf("Sec-WebSocket-Accept: %s", expectedAccept)) {
		return nil, fmt.Errorf("invalid Sec-WebSocket-Accept in response")
	}

	c := &connection{
		conn:            conn,
		stateConnecting: make(chan struct{}),
	}
	conn = nil // do not close the connection in the defer if we succeed
	connectionsMu.Lock()
	connections[host] = c
	connectionsMu.Unlock()
	return c, nil
}

func generateNonce() (string, error) {
	nonce := make([]byte, 16)
	_, err := rand.Read(nonce) // Use crypto/rand to generate secure random bytes
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(nonce), nil
}

func computeAcceptKey(nonce string) string {
	const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	hash := sha1.New() // nolint:gosec // this is the spefied hash of the WebSocket protocol
	hash.Write([]byte(nonce + websocketGUID))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
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
func validateWebsocketURI(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidURI, err)
	}

	if u.Host == "" {
		return "", fmt.Errorf("%w: no host specified", ErrInvalidURI)
	}

	if u.Fragment != "" {
		return "", fmt.Errorf("%w: fragment identifiers are not allowed in WebSocket URIs", ErrInvalidURI)
	}

	if u.Port() != "" {
		port, err := strconv.Atoi(u.Port())
		if err != nil || port < 1 || port > 65535 {
			return "", fmt.Errorf("%w: invalid port number", ErrInvalidURI)
		}
	}

	if u.Scheme != string(SchemeWS) && u.Scheme != string(SchemeWSS) {
		return "", fmt.Errorf("%w: %s, must be '%s' or '%s'", ErrInvalidURI, u.Scheme, SchemeWSS, SchemeWS)
	}

	if u.Port() == "" {
		switch u.Scheme {
		case string(SchemeWS):
			u.Host += ":80"
		case string(SchemeWSS):
			u.Host += ":443"
		default:
			return "", fmt.Errorf("%w: %s, must be '%s' or '%s'", ErrInvalidURI, u.Scheme, SchemeWSS, SchemeWS)
		}
	}

	return u.String(), nil
}
