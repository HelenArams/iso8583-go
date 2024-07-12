package iso8583

import (
	"reflect"
	"strings"
	"testing"
)

// TestRoundTrip packs a message, unpacks the result and checks that the
// original comes back, padding of fixed fields included.
func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		mti    string
		fields map[int]string
		want   map[int]string // nil means "same as fields"
	}{
		{"empty", "0800", map[int]string{}, nil},
		{"fixed numeric", "0200", map[int]string{3: "000000", 4: "000000001000"}, nil},
		{"fixed alpha", "0200", map[int]string{49: "USD"}, nil},
		{"fixed alphanumeric", "0210", map[int]string{39: "00", 41: "TERM0001"}, nil},
		{"fixed binary", "0200", map[int]string{52: "0123456789ABCDEF"}, nil},
		{"fixed binary lower case", "0200", map[int]string{52: "0123456789abcdef"}, nil},
		{"llvar numeric", "0200", map[int]string{2: "4111111111111111"}, nil},
		{"llvar numeric max", "0200", map[int]string{2: strings.Repeat("9", 19)}, nil},
		{"llvar track2", "0200", map[int]string{35: "4111111111111111=2512101123400001"}, nil},
		{"lllvar alphanumeric", "0200", map[int]string{48: "name=value;other=42 ~!@#$%^&*()"}, nil},
		{"lllvar alphanumeric max", "0200", map[int]string{48: strings.Repeat("x", 999)}, nil},
		{"lllvar binary", "0200", map[int]string{55: "9F2608AABBCCDDEEFF00119F3602001A"}, nil},
		{"lllvar binary max", "0200", map[int]string{55: strings.Repeat("AB", 255)}, nil},
		{"empty variable", "0200", map[int]string{2: "", 48: ""}, nil},
		{"field 64", "0800", map[int]string{64: "AABBCCDDEEFF0011"}, nil},
		{"field 65", "0800", map[int]string{65: "FF"}, nil},
		{"field 128", "0800", map[int]string{128: "AABBCCDDEEFF0011"}, nil},
		{"fields 2 64 65 128", "0800", map[int]string{
			2: "4111111111111111", 64: "0000000000000000", 65: "00", 128: "FFFFFFFFFFFFFFFF",
		}, nil},
		{
			"padding is applied once",
			"0200",
			map[int]string{4: "1000", 41: "T1", 49: "A"},
			map[int]string{4: "000000001000", 41: "T1      ", 49: "A  "},
		},
		{
			"everything in the test spec",
			"0200",
			map[int]string{
				2: "4111111111111111", 3: "000000", 4: "000000001000", 11: "123456",
				35: "4111111111111111=2512101", 39: "00", 41: "TERM0001", 48: "hello",
				49: "EUR", 52: "0123456789ABCDEF", 55: "9F2608AABBCCDDEEFF00",
				64: "AABBCCDDEEFF0011", 65: "01", 70: "301", 128: "1122334455667788",
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewMessage(tt.mti)
			for f, v := range tt.fields {
				msg.Set(f, v)
			}
			packed, err := Pack(msg, testSpec)
			if err != nil {
				t.Fatalf("Pack() error = %v", err)
			}
			got, err := Unpack(packed, testSpec)
			if err != nil {
				t.Fatalf("Unpack(%s) error = %v", packed, err)
			}
			want := tt.want
			if want == nil {
				want = tt.fields
			}
			if got.MTI != tt.mti {
				t.Errorf("MTI = %q, want %q", got.MTI, tt.mti)
			}
			if !reflect.DeepEqual(got.fields, want) {
				t.Errorf("fields = %v, want %v", got.fields, want)
			}

			// Packing the unpacked message must reproduce the bytes exactly.
			again, err := Pack(got, testSpec)
			if err != nil {
				t.Fatalf("second Pack() error = %v", err)
			}
			if string(again) != string(packed) {
				t.Errorf("second Pack() = %s, want %s", again, packed)
			}
		})
	}
}

// TestUnpackRejectsEveryTruncation cuts a valid message at every position
// and expects a clean ErrTruncated (or trailing/invalid data when the cut
// happens to leave a self-consistent prefix) rather than a panic.
func TestUnpackRejectsEveryTruncation(t *testing.T) {
	msg := NewMessage("0200")
	msg.Set(2, "4111111111111111")
	msg.Set(3, "000000")
	msg.Set(48, "hello")
	msg.Set(52, "0123456789ABCDEF")
	msg.Set(128, "AABBCCDDEEFF0011")
	packed, err := Pack(msg, testSpec)
	if err != nil {
		t.Fatal(err)
	}
	for cut := 0; cut < len(packed); cut++ {
		if _, err := Unpack(packed[:cut], testSpec); err == nil {
			t.Errorf("Unpack(%s) succeeded on input cut at %d", packed[:cut], cut)
		}
	}
}
