package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

const (
	serverMode = "server"
	clientMode = "client"
)

type config struct {
	mode string
}

func validateConfig(c config) error {
	if c.mode != serverMode && c.mode != clientMode {
		return fmt.Errorf("expected mode %q to be %q or %q", c.mode, serverMode, clientMode)
	}
	return nil
}

func main() {
	fmt.Println("🚧 🏗️  ... under construction ... 🏗️  🚧")

	c := config{}
	flag.StringVar(&c.mode, "m", serverMode, fmt.Sprintf("test mode: %q or %q", serverMode, clientMode))
	flag.Parse()

	err := validateConfig(c)
	if err != nil {
		fmt.Printf("config validation failed:\n%v", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	switch c.mode {
	case serverMode:
		testServer(ctx)
	case clientMode:
		testClient(ctx)
	}
}
