package websocket

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1" // nolint:gosec // this is the spefied hash of the WebSocket protocol
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
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

// ConnectionOption is a function that configures the WebSocket connection.
//
// It can be used to set optional fields in the initial request, such as cookies,
// additional headers, etc.
type ConnectionOption func(*http.Request)

// WithCookies adds a cookie to the WebSocket connection request.
func WithCookies(cookie http.Cookie) func(r *http.Request) {
	return func(r *http.Request) {
		r.AddCookie(&cookie)
	}
}

var (
	connectionsMu sync.Mutex
	connections   = map[string]*connection{}
)

// EstablishConnection establishes a WebSocket connection to the given URI.
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
func EstablishConnection(ctx context.Context, uri string, opts ...ConnectionOption) (Connection, error) {
	u, err := validateWebsocketURI(uri)
	if err != nil {
		return nil, err
	}
	host := u.Host

	// TODO: if we get websocket to support HTTP/3, we should use UDP instead of TCP
	// TODO: ctx dependent resolver, and maybe skip the resolve in the Dialer?
	// TODO: use-case if client cannot resolve host due to proxy server?
	addr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %w", err)
	}

	c := &connection{
		stateConnecting: make(chan struct{}),
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
	connections[host] = c
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

	// TODO: apply the options to the request
	// TODO: validate the request before executing it
	// TODO: |Sec-WebSocket-Protocol| header field support
	// TODO: |Sec-WebSocket-Extensions| header field support
	req := &http.Request{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Method:     http.MethodGet,
		URL:        u,
		Host:       u.Host,
		Header: http.Header{
			"Upgrade":               {"websocket"},
			"Connection":            {"Upgrade"},
			"Sec-WebSocket-Key":     {nonce},
			"Sec-WebSocket-Version": {"13"},
		},
	}

	err = req.Write(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to send handshake request: %w", err)
	}

	response, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return nil, fmt.Errorf("failed to read handshake response: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusSwitchingProtocols {
		return nil, fmt.Errorf("invalid handshake response: %s", response.Status)
	}

	if conditions := response.Header["Connection"]; len(conditions) == 0 || conditions[0] != "Upgrade" {
		return nil, fmt.Errorf("invalid handshake response: invalid Connection header: %v", conditions)
	}

	if upgrade := response.Header["Upgrade"]; len(upgrade) == 0 || upgrade[0] != "websocket" {
		return nil, fmt.Errorf("invalid handshake response: invalid Upgrade header: %v", upgrade)
	}

	expectedAccept := computeAcceptKey(nonce)
	if accept := response.Header["Sec-WebSocket-Accept"]; len(accept) == 0 || strings.TrimSpace(accept[0]) != expectedAccept {
		return nil, fmt.Errorf("invalid handshake response: invalid Sec-WebSocket-Accept header: %v", accept)
	}

	// TODO: check for |Sec-WebSocket-Extensions|
	// TODO: check for |Sec-WebSocket-Protocol|

	c.conn = conn
	close(c.stateConnecting)
	conn = nil // do not close the connection in the defer if we succeed
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
func validateWebsocketURI(uri string) (*url.URL, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidURI, err)
	}

	if u.Host == "" {
		return nil, fmt.Errorf("%w: no host specified", ErrInvalidURI)
	}

	if u.Fragment != "" {
		return nil, fmt.Errorf("%w: fragment identifiers are not allowed in WebSocket URIs", ErrInvalidURI)
	}

	if u.Port() != "" {
		port, err := strconv.Atoi(u.Port())
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("%w: invalid port number", ErrInvalidURI)
		}
	}

	if u.Scheme != string(SchemeWS) && u.Scheme != string(SchemeWSS) {
		return nil, fmt.Errorf("%w: %s, must be '%s' or '%s'", ErrInvalidURI, u.Scheme, SchemeWSS, SchemeWS)
	}

	if u.Port() == "" {
		switch u.Scheme {
		case string(SchemeWS):
			u.Host += ":80"
		case string(SchemeWSS):
			u.Host += ":443"
		default:
			return nil, fmt.Errorf("%w: %s, must be '%s' or '%s'", ErrInvalidURI, u.Scheme, SchemeWSS, SchemeWS)
		}
	}

	return u, nil
}
