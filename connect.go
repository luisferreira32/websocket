package websocket

import (
	"context"
	"net/http"
)

type ConnectionOption func(*http.Request)

func Connect(ctx context.Context, uri string, opts ...ConnectionOption) (Connection, error) {
	return Connection{}, nil
}
