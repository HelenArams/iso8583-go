package main

import (
	"bytes"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/HelenArams/iso8583-go"
	"github.com/HelenArams/iso8583-go/netio"
	"github.com/HelenArams/iso8583-go/spec87"
)

// fakeHost accepts one connection, reads one request and answers it with
// reply, which receives the parsed request to build the response from.
// A nil reply closes the connection without answering.
func fakeHost(t *testing.T, reply func(req *iso8583.Message) *iso8583.Message) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		raw, err := netio.ReadMessage(conn)
		if err != nil {
			return
		}
		req, err := iso8583.Unpack(raw, spec87.Spec)
		if err != nil || reply == nil {
			return
		}
		out, err := iso8583.Pack(reply(req), spec87.Spec)
		if err != nil {
			return
		}
		_ = netio.WriteMessage(conn, out)
	}()
	return ln.Addr().String()
}

func approve(req *iso8583.Message) *iso8583.Message {
	resp := iso8583.NewMessage("0210")
	resp.Set(2, req.Get(2))
	resp.Set(4, req.Get(4))
	resp.Set(11, req.Get(11))
	resp.Set(38, "654321")
	resp.Set(39, "00")
	return resp
}

func decline(req *iso8583.Message) *iso8583.Message {
	resp := approve(req)
	resp.Unset(38)
	resp.Set(39, "51")
	return resp
}

func TestRunApproved(t *testing.T) {
	o := defaults()
	o.addr = fakeHost(t, approve)

	var stdout, stderr bytes.Buffer
	if code := run(o, now, &stdout, &stderr); code != exitApproved {
		t.Fatalf("run() = %d, want %d; stderr: %s", code, exitApproved, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"> request to " + o.addr,
		"MTI 0200",
		"002  411111******1111", // PAN masked in both directions
		"< response in ",
		"MTI 0210",
		"038  654321",
		"approved: response code 00, authorization code 654321",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "4111111111111111") {
		t.Error("output leaks the full PAN")
	}
	if stderr.Len() != 0 {
		t.Errorf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunDeclined(t *testing.T) {
	o := defaults()
	o.addr = fakeHost(t, decline)

	var stdout, stderr bytes.Buffer
	if code := run(o, now, &stdout, &stderr); code != exitDeclined {
		t.Fatalf("run() = %d, want %d", code, exitDeclined)
	}
	if !strings.Contains(stdout.String(), "declined: response code 51") {
		t.Errorf("output:\n%s", stdout.String())
	}
}

func TestRunWirePrintsRawMessages(t *testing.T) {
	o := defaults()
	o.wire = true
	o.addr = fakeHost(t, approve)

	var stdout, stderr bytes.Buffer
	run(o, now, &stdout, &stderr)
	out := stdout.String()
	if !strings.Contains(out, "> wire 0200") || !strings.Contains(out, "< wire 0210") {
		t.Errorf("output lacks wire dumps:\n%s", out)
	}
}

func TestRunDerivesStanWhenZero(t *testing.T) {
	var seen string
	o := defaults()
	o.stan = 0
	o.addr = fakeHost(t, func(req *iso8583.Message) *iso8583.Message {
		seen = req.Get(11)
		return approve(req)
	})

	var stdout, stderr bytes.Buffer
	if code := run(o, now, &stdout, &stderr); code != exitApproved {
		t.Fatalf("run() = %d: %s", code, stderr.String())
	}
	if len(seen) != 6 || seen == "000000" {
		t.Errorf("derived STAN = %q", seen)
	}
}

func TestRunErrors(t *testing.T) {
	t.Run("invalid options", func(t *testing.T) {
		o := defaults()
		o.pan = "bad"
		var stdout, stderr bytes.Buffer
		if code := run(o, now, &stdout, &stderr); code != exitError {
			t.Errorf("run() = %d, want %d", code, exitError)
		}
		if !strings.Contains(stderr.String(), "pan must be") {
			t.Errorf("stderr = %q", stderr.String())
		}
	})

	t.Run("connection refused", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		addr := ln.Addr().String()
		_ = ln.Close()

		o := defaults()
		o.addr = addr
		var stdout, stderr bytes.Buffer
		if code := run(o, now, &stdout, &stderr); code != exitError {
			t.Errorf("run() = %d, want %d", code, exitError)
		}
		if !strings.Contains(stderr.String(), "connect:") {
			t.Errorf("stderr = %q", stderr.String())
		}
	})

	t.Run("host closes without answering", func(t *testing.T) {
		o := defaults()
		o.addr = fakeHost(t, nil)
		var stdout, stderr bytes.Buffer
		if code := run(o, now, &stdout, &stderr); code != exitError {
			t.Errorf("run() = %d, want %d", code, exitError)
		}
		if !strings.Contains(stderr.String(), "closed the connection without answering") {
			t.Errorf("stderr = %q", stderr.String())
		}
	})

	t.Run("response times out", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		go func() {
			conn, err := ln.Accept()
			if err == nil {
				time.Sleep(500 * time.Millisecond) // never answer within the timeout
				_ = conn.Close()
			}
		}()

		o := defaults()
		o.addr = ln.Addr().String()
		o.timeout = 100 * time.Millisecond
		var stdout, stderr bytes.Buffer
		if code := run(o, now, &stdout, &stderr); code != exitError {
			t.Errorf("run() = %d, want %d", code, exitError)
		}
		if !strings.Contains(stderr.String(), "receive:") {
			t.Errorf("stderr = %q", stderr.String())
		}
	})

	t.Run("unparseable response", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		go func() {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			defer conn.Close()
			if _, err := netio.ReadMessage(conn); err == nil {
				_ = netio.WriteMessage(conn, []byte("garbage"))
			}
		}()

		o := defaults()
		o.addr = ln.Addr().String()
		var stdout, stderr bytes.Buffer
		if code := run(o, now, &stdout, &stderr); code != exitError {
			t.Errorf("run() = %d, want %d", code, exitError)
		}
		if !strings.Contains(stderr.String(), "cannot parse response") {
			t.Errorf("stderr = %q", stderr.String())
		}
	})
}
