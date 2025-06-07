package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/luisferreira32/websocket"
)

func server(ctx context.Context) {
	m := http.Server{Addr: ":9001", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r)
		if err != nil {
			slog.Error("Failed to accept websocket connection", "error", err)
			return
		}

		go func() {
			defer conn.Close()
			buf := make([]byte, 1024)
			for {
				n, err := conn.Read(buf)
				if err != nil {
					slog.Error("Error reading from websocket", "error", err)
					return
				}
				slog.Debug("Received message", "message", string(buf[:n]))

				_, err = conn.Write(buf[:n])
				if err != nil {
					slog.Error("Error writing to websocket", "error", err)
					return
				}
			}
		}()
	})}

	go func() {
		err := m.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Error starting server", "error", err)
		}
	}()
	<-ctx.Done()

	xtc, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := m.Shutdown(xtc)
	if err != nil {
		slog.Error("Error shutting down server", "error", err)
	}
}
