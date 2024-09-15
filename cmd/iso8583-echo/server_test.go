package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/HelenArams/iso8583-go"
	"github.com/HelenArams/iso8583-go/netio"
	"github.com/HelenArams/iso8583-go/spec87"
)

func startServer(t *testing.T, idle time.Duration) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	srv := newServer(spec87.Spec, slog.New(slog.NewTextHandler(io.Discard, nil)), idle)

	done := make(chan error, 1)
	go func() { done <- srv.serve(ctx, ln) }()

	return ln.Addr().String(), func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("serve() = %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("serve() did not return after cancel")
		}
	}
}

func dial(t *testing.T, addr string) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func send(t *testing.T, conn net.Conn, msg *iso8583.Message) *iso8583.Message {
	t.Helper()
	data, err := iso8583.Pack(msg, spec87.Spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := netio.WriteMessage(conn, data); err != nil {
		t.Fatal(err)
	}
	raw, err := netio.ReadMessage(conn)
	if err != nil {
		t.Fatalf("reading response: %v", err)
	}
	resp, err := iso8583.Unpack(raw, spec87.Spec)
	if err != nil {
		t.Fatalf("parsing response %q: %v", raw, err)
	}
	return resp
}

func TestServerAnswersOnSameConnection(t *testing.T) {
	addr, stop := startServer(t, 0)
	defer stop()
	conn := dial(t, addr)

	resp := send(t, conn, newRequest("0200", validAuth))
	if resp.MTI != "0210" || resp.Get(39) != "00" {
		t.Fatalf("first response: %s", resp)
	}
	if resp.Get(38) != "000001" {
		t.Errorf("field 38 = %q, want first auth code", resp.Get(38))
	}
	if resp.Get(11) != validAuth[11] || resp.Get(37) != validAuth[37] {
		t.Errorf("STAN/RRN not echoed:\n%s", resp)
	}
	if resp.Has(35) || resp.Has(52) {
		t.Error("track 2 or PIN block echoed back")
	}

	resp = send(t, conn, newRequest("0200", with(validAuth, 4, "000000002505")))
	if resp.Get(39) != "05" || resp.Has(38) {
		t.Errorf("decline response: %s", resp)
	}

	resp = send(t, conn, newRequest("0800", map[int]string{7: "0915153500", 11: "000777", 70: "301"}))
	if resp.MTI != "0810" || resp.Get(39) != "00" || resp.Get(70) == "301" {
		t.Errorf("echo test response: %s", resp)
	}

	// a second connection gets the next auth code
	resp = send(t, dial(t, addr), newRequest("0100", validAuth))
	if resp.MTI != "0110" || resp.Get(38) != "000002" {
		t.Errorf("second connection: %s", resp)
	}
}

func TestServerClosesOnMalformedInput(t *testing.T) {
	addr, stop := startServer(t, 0)
	defer stop()

	t.Run("bad frame content", func(t *testing.T) {
		conn := dial(t, addr)
		if err := netio.WriteMessage(conn, []byte("this is not iso8583")); err != nil {
			t.Fatal(err)
		}
		if _, err := netio.ReadMessage(conn); !errors.Is(err, io.EOF) {
			t.Errorf("err = %v, want EOF from server closing", err)
		}
	})

	t.Run("response mti", func(t *testing.T) {
		conn := dial(t, addr)
		data, _ := iso8583.Pack(newRequest("0210", validAuth), spec87.Spec)
		if err := netio.WriteMessage(conn, data); err != nil {
			t.Fatal(err)
		}
		if _, err := netio.ReadMessage(conn); !errors.Is(err, io.EOF) {
			t.Errorf("err = %v, want EOF", err)
		}
	})

	t.Run("zero length frame", func(t *testing.T) {
		conn := dial(t, addr)
		if _, err := conn.Write([]byte{0, 0}); err != nil {
			t.Fatal(err)
		}
		if _, err := netio.ReadMessage(conn); !errors.Is(err, io.EOF) {
			t.Errorf("err = %v, want EOF", err)
		}
	})

	// server must still be healthy afterwards
	if resp := send(t, dial(t, addr), newRequest("0200", validAuth)); resp.Get(39) != "00" {
		t.Errorf("server unhealthy after bad input: %s", resp)
	}
}

func TestServerConcurrentClients(t *testing.T) {
	addr, stop := startServer(t, 0)
	defer stop()

	const clients, perClient = 8, 20
	var wg sync.WaitGroup
	codes := make(chan string, clients*perClient)
	for c := 0; c < clients; c++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
			if err != nil {
				t.Error(err)
				return
			}
			defer conn.Close()
			_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
			for i := 0; i < perClient; i++ {
				data, _ := iso8583.Pack(newRequest("0200", validAuth), spec87.Spec)
				if err := netio.WriteMessage(conn, data); err != nil {
					t.Error(err)
					return
				}
				raw, err := netio.ReadMessage(conn)
				if err != nil {
					t.Error(err)
					return
				}
				resp, err := iso8583.Unpack(raw, spec87.Spec)
				if err != nil {
					t.Error(err)
					return
				}
				codes <- resp.Get(38)
			}
		}()
	}
	wg.Wait()
	close(codes)

	seen := make(map[string]bool)
	for code := range codes {
		if seen[code] {
			t.Errorf("auth code %s issued twice", code)
		}
		seen[code] = true
	}
	if len(seen) != clients*perClient {
		t.Errorf("got %d auth codes, want %d", len(seen), clients*perClient)
	}
}

func TestServerIdleTimeout(t *testing.T) {
	addr, stop := startServer(t, 150*time.Millisecond)
	defer stop()

	conn := dial(t, addr)
	start := time.Now()
	_, err := netio.ReadMessage(conn)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("err = %v, want EOF after idle timeout", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Error("idle connection was not closed promptly")
	}
}

func TestServerShutdownClosesConnections(t *testing.T) {
	addr, stop := startServer(t, 0)
	conn := dial(t, addr)

	// make sure the server has accepted before stopping
	if resp := send(t, conn, newRequest("0800", map[int]string{11: "000001"})); resp.MTI != "0810" {
		t.Fatalf("unexpected response %s", resp)
	}

	stop()
	if _, err := netio.ReadMessage(conn); err == nil {
		t.Error("connection still open after shutdown")
	}
	if _, err := net.DialTimeout("tcp", addr, 500*time.Millisecond); err == nil {
		t.Error("listener still accepting after shutdown")
	}
}

func TestRunListensAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, "127.0.0.1:0", 0, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run() = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run() did not stop")
	}

	// a bad address fails fast
	if err := run(context.Background(), "127.0.0.1:-1", 0, slog.New(slog.NewTextHandler(io.Discard, nil))); err == nil {
		t.Error("run() with invalid address succeeded")
	}
}
