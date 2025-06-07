package main

import (
	"context"

	"github.com/luisferreira32/websocket"
)

func client(ctx context.Context) {
	conn, err := websocket.Connect(ctx, "ws://localhost:9001")
	if err != nil {
		panic(err)
	}

	b := make([]byte, 1024)
	conn.Write(b)
	conn.Read(b)
}
