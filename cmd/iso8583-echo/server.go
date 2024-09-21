package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HelenArams/iso8583-go"
	"github.com/HelenArams/iso8583-go/netio"
)

// server answers ISO 8583 requests on every accepted connection until the
// context is cancelled, then closes the connections and waits for their
// handlers to return.
type server struct {
	spec   iso8583.Spec
	logger *slog.Logger
	idle   time.Duration

	seq atomic.Uint32 // source of authorization codes

	mu    sync.Mutex
	conns map[net.Conn]struct{}
	wg    sync.WaitGroup
}

func newServer(spec iso8583.Spec, logger *slog.Logger, idle time.Duration) *server {
	return &server{
		spec:   spec,
		logger: logger,
		idle:   idle,
		conns:  make(map[net.Conn]struct{}),
	}
}

// serve accepts connections on ln until ctx is done or Accept fails. It
// always closes ln and returns only after every connection handler has
// finished.
func (s *server) serve(ctx context.Context, ln net.Listener) error {
	defer s.wg.Wait()

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			s.closeAll()
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept: %w", err)
		}

		s.mu.Lock()
		s.conns[conn] = struct{}{}
		s.mu.Unlock()

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handle(conn)
		}()
	}
}

func (s *server) closeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for conn := range s.conns {
		_ = conn.Close()
	}
}

func (s *server) release(conn net.Conn) {
	s.mu.Lock()
	delete(s.conns, conn)
	s.mu.Unlock()
	_ = conn.Close()
}

// handle serves one connection, one request at a time. Any framing or
// parsing error ends the connection: a peer that sends malformed data cannot
// be trusted to stay in sync.
func (s *server) handle(conn net.Conn) {
	defer s.release(conn)
	logger := s.logger.With("remote", conn.RemoteAddr().String())

	for {
		if s.idle > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(s.idle))
		}
		raw, err := netio.ReadMessage(conn)
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				logger.Warn("read failed", "err", err)
			}
			return
		}

		req, err := iso8583.Unpack(raw, s.spec)
		if err != nil {
			logger.Warn("rejecting malformed message", "err", err)
			return
		}

		resp, err := respond(req, s.authCode)
		if err != nil {
			logger.Warn("cannot answer message", "mti", req.MTI, "err", err)
			return
		}

		out, err := iso8583.Pack(resp, s.spec)
		if err != nil {
			logger.Error("pack response failed", "err", err)
			return
		}
		if err := netio.WriteMessage(conn, out); err != nil {
			logger.Warn("write failed", "err", err)
			return
		}
		logger.Info("answered", "mti", req.MTI, "stan", req.Get(11), "amount", req.Get(4), "rc", resp.Get(39))
	}
}

func (s *server) authCode() string {
	return fmt.Sprintf("%06d", s.seq.Add(1)%1_000_000)
}
