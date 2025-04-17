package websocket

import (
	"errors"
	"testing"
)

func Test_validateWebsocketURI(t *testing.T) {
	tests := []struct {
		name       string
		uri        string
		wantErr    error
		wantResult string
	}{
		{
			name:       "Valid ws URI without port",
			uri:        "ws://example.com",
			wantResult: "ws://example.com:80",
		},
		{
			name:       "Valid wss URI without port",
			uri:        "wss://example.com",
			wantResult: "wss://example.com:443",
		},
		{
			name:       "Valid ws URI with port",
			uri:        "ws://example.com:8080",
			wantResult: "ws://example.com:8080",
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
			result, err := validateWebsocketURI(testcase.uri)
			if !errors.Is(err, testcase.wantErr) {
				t.Errorf("validateWebsocketURI() error = %v, wantErr %v", err, testcase.wantErr)
			}
			if result != nil && result.String() != testcase.wantResult {
				t.Errorf("validateWebsocketURI() result = %v, wantResult %v", result.String(), testcase.wantResult)
			}
		})
	}
}
