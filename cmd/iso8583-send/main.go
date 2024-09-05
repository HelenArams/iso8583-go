// Command iso8583-send builds an authorization request from flags, sends it
// to a host over TCP with two-byte length framing and prints the parsed
// response with card data masked.
//
// Usage:
//
//	iso8583-send [-addr 127.0.0.1:8583] [-mti 0200] [-pan 4111111111111111]
//	             [-amount 1000] [-currency 978] [-stan 123] [-expiry 2712]
//	             [-terminal TERM0001] [-merchant MERCHANT0000001]
//	             [-timeout 5s] [-wire]
//
// The exit status is 0 when the host approved the request (field 39 = "00"),
// 2 when it declined, and 1 on any transport or format error, so the command
// can be used directly in shell scripts.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/HelenArams/iso8583-go"
	"github.com/HelenArams/iso8583-go/netio"
	"github.com/HelenArams/iso8583-go/spec87"
)

const (
	exitApproved = 0
	exitError    = 1
	exitDeclined = 2
)

func main() {
	var o options
	flag.StringVar(&o.addr, "addr", "127.0.0.1:8583", "host address")
	flag.StringVar(&o.mti, "mti", "0200", "message type: 0100 (authorization) or 0200 (financial)")
	flag.StringVar(&o.pan, "pan", "4111111111111111", "primary account number, field 2")
	flag.StringVar(&o.expiry, "expiry", "", "card expiry as YYMM, field 14 (omitted when empty)")
	flag.StringVar(&o.amount, "amount", "1000", "amount in minor units, field 4 (iso8583-echo declines amounts ending in 05)")
	flag.StringVar(&o.currency, "currency", "978", "numeric currency code, field 49")
	flag.IntVar(&o.stan, "stan", 0, "system trace audit number, field 11 (0 derives one from the clock)")
	flag.StringVar(&o.terminal, "terminal", "TERM0001", "terminal identification, field 41")
	flag.StringVar(&o.merchant, "merchant", "MERCHANT0000001", "card acceptor identification, field 42")
	flag.DurationVar(&o.timeout, "timeout", 5*time.Second, "connect and response timeout")
	flag.BoolVar(&o.wire, "wire", false, "also print the raw packed messages (contains the unmasked PAN)")
	flag.Parse()

	os.Exit(run(o, time.Now(), os.Stdout, os.Stderr))
}

// run performs one request/response exchange and returns the exit status.
func run(o options, now time.Time, stdout, stderr io.Writer) int {
	if o.stan == 0 {
		o.stan = int(now.UnixMilli()%999_999) + 1
	}
	req, err := buildRequest(o, now)
	if err != nil {
		fmt.Fprintln(stderr, "iso8583-send:", err)
		return exitError
	}
	data, err := iso8583.Pack(req, spec87.Spec)
	if err != nil {
		fmt.Fprintln(stderr, "iso8583-send:", err)
		return exitError
	}

	fmt.Fprintf(stdout, "> request to %s\n%s\n", o.addr, req)
	if o.wire {
		fmt.Fprintf(stdout, "> wire %s\n", data)
	}

	raw, elapsed, err := exchange(o.addr, data, o.timeout)
	if err != nil {
		fmt.Fprintln(stderr, "iso8583-send:", err)
		return exitError
	}
	resp, err := iso8583.Unpack(raw, spec87.Spec)
	if err != nil {
		fmt.Fprintf(stderr, "iso8583-send: cannot parse response: %v\n", err)
		if o.wire {
			fmt.Fprintf(stderr, "< wire %s\n", raw)
		}
		return exitError
	}

	fmt.Fprintf(stdout, "\n< response in %s\n%s\n", elapsed.Round(time.Microsecond), resp)
	if o.wire {
		fmt.Fprintf(stdout, "< wire %s\n", raw)
	}

	if rc := resp.Get(39); rc != "00" {
		fmt.Fprintf(stdout, "\ndeclined: response code %s\n", rc)
		return exitDeclined
	}
	fmt.Fprintf(stdout, "\napproved: response code 00, authorization code %s\n", resp.Get(38))
	return exitApproved
}

// exchange connects to addr, sends one framed message and reads one framed
// reply, all within timeout. It reports how long the host took to answer.
func exchange(addr string, data []byte, timeout time.Duration) (reply []byte, elapsed time.Duration, err error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, 0, fmt.Errorf("connect: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, 0, err
	}
	start := time.Now()
	if err := netio.WriteMessage(conn, data); err != nil {
		return nil, 0, fmt.Errorf("send: %w", err)
	}
	reply, err = netio.ReadMessage(conn)
	elapsed = time.Since(start)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, elapsed, errors.New("host closed the connection without answering")
		}
		return nil, elapsed, fmt.Errorf("receive: %w", err)
	}
	return reply, elapsed, nil
}
