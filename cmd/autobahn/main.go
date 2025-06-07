package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const (
	serverMode = "server"
	clientMode = "client"
)

// a vibing log handler to write logs without prefixes
type noPrefixHandler struct {
	w     io.Writer
	level slog.Level
}

func (h noPrefixHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

//nolint:gocritic // cannot change param to pointer, need to respect interface object
func (h noPrefixHandler) Handle(_ context.Context, r slog.Record) error {
	if r.Level >= h.level {
		var err error
		attrs := make([]any, 0, r.NumAttrs())
		r.Attrs(func(a slog.Attr) bool {
			attrs = append(attrs, a.Value)
			return true
		})
		fmtMessage := fmt.Sprintf(r.Message, attrs...) + "\n"
		switch r.Level {
		case slog.LevelDebug:
			_, err = io.WriteString(h.w, "🗣   "+fmtMessage)
		case slog.LevelInfo:
			_, err = io.WriteString(h.w, "ℹ️   "+fmtMessage)
		case slog.LevelWarn:
			_, err = io.WriteString(h.w, "⚠️   "+fmtMessage)
		case slog.LevelError:
			_, err = io.WriteString(h.w, "❌   "+fmtMessage)
		default:
			_, err = io.WriteString(h.w, fmtMessage)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (h noPrefixHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h noPrefixHandler) WithGroup(_ string) slog.Handler {
	return h
}

type config struct {
	mode string
}

func validateConfig(c config) error {
	if c.mode != serverMode && c.mode != clientMode {
		return fmt.Errorf("expected mode %q to be %q or %q", c.mode, serverMode, clientMode)
	}
	var logLevel slog.Level
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	default:
		logLevel = slog.LevelInfo
	}
	slog.SetDefault(slog.New(noPrefixHandler{w: os.Stderr, level: logLevel}))
	return nil
}

func main() {
	fmt.Println("🚧 🏗️  ... under construction ... 🏗️  🚧")

	c := config{}
	flag.StringVar(&c.mode, "m", serverMode, fmt.Sprintf("test mode: %q or %q", serverMode, clientMode))
	flag.Parse()

	err := validateConfig(c)
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	switch c.mode {
	case serverMode:
		server(ctx)
	case clientMode:
		client(ctx)
	}
}
