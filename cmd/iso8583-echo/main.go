// Command iso8583-echo is a tiny authorization host for testing clients.
//
// It listens for length-prefixed ISO 8583 messages, answers authorization
// (01xx) and financial (02xx) requests with a response whose field 39 is "00"
// (approved), or "05" (do not honor) when the transaction amount ends in 05,
// and acknowledges reversals and network management messages. Connections
// are kept open for further requests until the client closes them or stays
// idle too long.
//
// Usage:
//
//	iso8583-echo [-addr :8583] [-idle-timeout 5m]
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HelenArams/iso8583-go/spec87"
)

func main() {
	addr := flag.String("addr", ":8583", "TCP address to listen on")
	idle := flag.Duration("idle-timeout", 5*time.Minute, "close connections idle for longer than this (0 disables)")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, *addr, *idle, logger)
	stop()
	if err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, addr string, idle time.Duration, logger *slog.Logger) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	logger.Info("listening", "addr", ln.Addr().String())

	srv := newServer(spec87.Spec, logger, idle)
	return srv.serve(ctx, ln)
}
