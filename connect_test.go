package websocket

import (
	"errors"
	"testing"
)

func Test_validateWebsocketURI(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantErr error
	}{
		{
			name: "Valid ws URI",
			uri:  "ws://example.com",
		},
		{
			name: "Valid wss URI",
			uri:  "wss://example.com",
		},
		{
			name:    "Invalid scheme",
			uri:     "http://example.com",
			wantErr: ErrInvalidURI,
		},
		{
			name:    "Missing host",
			uri:     "ws://",
			wantErr: ErrInvalidURI,
		},
		{
			name:    "Fragment identifier present",
			uri:     "ws://example.com#fragment",
			wantErr: ErrInvalidURI,
		},
		{
			name:    "Empty URI",
			uri:     "",
			wantErr: ErrInvalidURI,
		},
		{
			name:    "Invalid port",
			uri:     "wss://example.com:900000/",
			wantErr: ErrInvalidURI,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			err := validateWebsocketURI(testcase.uri)
			if !errors.Is(err, testcase.wantErr) {
				t.Errorf("validateWebsocketURI() error = %v, wantErr %v", err, testcase.wantErr)
			}
		})
	}
}
