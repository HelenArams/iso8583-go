package main

import (
	"testing"

	"github.com/HelenArams/iso8583-go"
)

func fixedAuthCode() string { return "123456" }

func newRequest(mti string, fields map[int]string) *iso8583.Message {
	m := iso8583.NewMessage(mti)
	for f, v := range fields {
		m.Set(f, v)
	}
	return m
}

var validAuth = map[int]string{
	2: "4111111111111111", 3: "000000", 4: "000000001000", 7: "0905211600",
	11: "000123", 12: "231600", 13: "0905", 14: "2712", 22: "012", 25: "00",
	32: "12345678901", 35: "4111111111111111=2712101", 37: "424921000123",
	41: "TERM0001", 42: "MERCHANT0000001", 49: "978", 52: "A1B2C3D4E5F60718",
}

func TestRespond(t *testing.T) {
	tests := []struct {
		name     string
		mti      string
		fields   map[int]string
		wantMTI  string
		wantRC   string
		wantAuth bool
	}{
		{"financial approved", "0200", validAuth, "0210", "00", true},
		{"authorization approved", "0100", validAuth, "0110", "00", true},
		{"financial advice", "0220", validAuth, "0230", "00", true},
		{"amount ending 05 declined", "0200", with(validAuth, 4, "000000001005"), "0210", "05", false},
		{"amount 05 exactly", "0200", with(validAuth, 4, "5"), "0210", "05", false},
		{"amount ending 50 approved", "0200", with(validAuth, 4, "000000000050"), "0210", "00", true},
		{"missing PAN", "0200", without(validAuth, 2), "0210", "30", false},
		{"missing amount", "0100", without(validAuth, 4), "0110", "30", false},
		{"missing STAN", "0200", without(validAuth, 11), "0210", "30", false},
		{"reversal", "0400", map[int]string{11: "000001", 90: "0200000001090521160000000000000000000000000"}, "0410", "00", false},
		{"reversal advice", "0420", map[int]string{11: "000001"}, "0430", "00", false},
		{"network management", "0800", map[int]string{7: "0905211600", 11: "000001", 70: "301"}, "0810", "00", false},
		{"unknown class", "0600", map[int]string{11: "000001"}, "0610", "12", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newRequest(tt.mti, tt.fields)
			resp, err := respond(req, fixedAuthCode)
			if err != nil {
				t.Fatalf("respond() error = %v", err)
			}
			if resp.MTI != tt.wantMTI {
				t.Errorf("MTI = %s, want %s", resp.MTI, tt.wantMTI)
			}
			if got := resp.Get(39); got != tt.wantRC {
				t.Errorf("field 39 = %q, want %q", got, tt.wantRC)
			}
			if resp.Has(38) != tt.wantAuth {
				t.Errorf("field 38 present = %v, want %v", resp.Has(38), tt.wantAuth)
			}
			if tt.wantAuth && resp.Get(38) != "123456" {
				t.Errorf("field 38 = %q, want 123456", resp.Get(38))
			}
		})
	}
}

func TestRespondEchoesRequestFields(t *testing.T) {
	resp, err := respond(newRequest("0200", validAuth), fixedAuthCode)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range echoedFields {
		if resp.Get(f) != validAuth[f] {
			t.Errorf("field %d = %q, want %q", f, resp.Get(f), validAuth[f])
		}
	}
	for _, f := range []int{22, 25, 35, 52} {
		if resp.Has(f) {
			t.Errorf("field %d must not be echoed", f)
		}
	}
}

func TestRespondRejectsNonRequests(t *testing.T) {
	for _, mti := range []string{"0210", "0110", "0810", "0430", "020", "02000", ""} {
		if _, err := respond(iso8583.NewMessage(mti), fixedAuthCode); err == nil {
			t.Errorf("respond(%q) succeeded, want error", mti)
		}
	}
}

func TestResponseMTI(t *testing.T) {
	tests := map[string]string{"0100": "0110", "0200": "0210", "0220": "0230", "0400": "0410", "0800": "0810", "1200": "1210"}
	for in, want := range tests {
		got, err := responseMTI(in)
		if err != nil || got != want {
			t.Errorf("responseMTI(%s) = %s, %v; want %s", in, got, err, want)
		}
	}
}

func TestAuthCodeSequence(t *testing.T) {
	s := newServer(nil, nil, 0)
	if a, b := s.authCode(), s.authCode(); a != "000001" || b != "000002" {
		t.Errorf("authCode() = %s, %s", a, b)
	}
}

func with(fields map[int]string, f int, v string) map[int]string {
	out := make(map[int]string, len(fields))
	for k, val := range fields {
		out[k] = val
	}
	out[f] = v
	return out
}

func without(fields map[int]string, f int) map[int]string {
	out := make(map[int]string, len(fields))
	for k, val := range fields {
		if k != f {
			out[k] = val
		}
	}
	return out
}
