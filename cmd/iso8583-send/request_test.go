package main

import (
	"strings"
	"testing"
	"time"

	"github.com/HelenArams/iso8583-go"
	"github.com/HelenArams/iso8583-go/spec87"
)

var now = time.Date(2024, time.September, 5, 21, 43, 20, 0, time.FixedZone("CEST", 2*3600))

func defaults() options {
	return options{
		addr:     "127.0.0.1:8583",
		mti:      "0200",
		pan:      "4111111111111111",
		amount:   "1000",
		currency: "978",
		stan:     123,
		terminal: "TERM0001",
		merchant: "MERCHANT0000001",
		timeout:  time.Second,
	}
}

func TestBuildRequest(t *testing.T) {
	o := defaults()
	o.expiry = "2712"
	m, err := buildRequest(o, now)
	if err != nil {
		t.Fatal(err)
	}

	want := map[int]string{
		2:  "4111111111111111",
		3:  "000000",
		4:  "1000",
		7:  "0905194320", // transmission time is UTC
		11: "000123",
		12: "214320", // local time
		13: "0905",
		14: "2712",
		22: "012",
		25: "00",
		37: "424921000123",
		41: "TERM0001",
		42: "MERCHANT0000001",
		43: "DEMO MERCHANT            YEREVAN      AM",
		49: "978",
	}
	if m.MTI != "0200" {
		t.Errorf("MTI = %s", m.MTI)
	}
	if len(m.Fields()) != len(want) {
		t.Errorf("fields %v, want %d of them", m.Fields(), len(want))
	}
	for f, v := range want {
		if got := m.Get(f); got != v {
			t.Errorf("field %d = %q, want %q", f, got, v)
		}
	}

	// The request must pack with the standard spec.
	if _, err := iso8583.Pack(m, spec87.Spec); err != nil {
		t.Errorf("Pack() error = %v", err)
	}
}

func TestBuildRequestOmitsExpiryByDefault(t *testing.T) {
	m, err := buildRequest(defaults(), now)
	if err != nil {
		t.Fatal(err)
	}
	if m.Has(14) {
		t.Error("field 14 should be absent when no expiry is given")
	}
}

func TestOptionsValidate(t *testing.T) {
	tests := []struct {
		name   string
		change func(*options)
		want   string
	}{
		{"short mti", func(o *options) { o.mti = "200" }, "must be four digits"},
		{"alpha mti", func(o *options) { o.mti = "02A0" }, "must be four digits"},
		{"response mti", func(o *options) { o.mti = "0210" }, "not a request"},
		{"short pan", func(o *options) { o.pan = "41111111111" }, "pan must be 12 to 19 digits"},
		{"long pan", func(o *options) { o.pan = strings.Repeat("4", 20) }, "pan must be"},
		{"alpha pan", func(o *options) { o.pan = "4111111111111abc" }, "pan must be"},
		{"bad expiry", func(o *options) { o.expiry = "12/27" }, "must be YYMM"},
		{"empty amount", func(o *options) { o.amount = "" }, "amount must be"},
		{"decimal amount", func(o *options) { o.amount = "10.00" }, "amount must be"},
		{"long amount", func(o *options) { o.amount = "1234567890123" }, "amount must be"},
		{"alpha currency", func(o *options) { o.currency = "EUR" }, "three-digit numeric code"},
		{"zero stan", func(o *options) { o.stan = 0 }, "stan 0 must be between"},
		{"big stan", func(o *options) { o.stan = 1_000_000 }, "stan 1000000 must be between"},
		{"empty terminal", func(o *options) { o.terminal = "" }, "terminal must be"},
		{"long terminal", func(o *options) { o.terminal = "TERMINAL1" }, "terminal must be"},
		{"long merchant", func(o *options) { o.merchant = strings.Repeat("M", 16) }, "merchant must be"},
		{"zero timeout", func(o *options) { o.timeout = 0 }, "timeout must be positive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaults()
			tt.change(&o)
			_, err := buildRequest(o, now)
			if err == nil {
				t.Fatal("buildRequest() succeeded, want error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not contain %q", err, tt.want)
			}
		})
	}
}

func TestRetrievalReference(t *testing.T) {
	if got := retrievalReference(now, 7); got != "424921000007" {
		t.Errorf("retrievalReference() = %q", got)
	}
	if got := retrievalReference(time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC), 999_999); got != "000100999999" {
		t.Errorf("retrievalReference() = %q", got)
	}
}
