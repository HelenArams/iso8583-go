package iso8583

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
)

// FuzzUnpack checks that Unpack never panics on arbitrary input and that any
// message it accepts survives a Pack/Unpack round trip.
func FuzzUnpack(f *testing.F) {
	seeds := []string{
		"",
		"0200",
		"0800" + "0000000000000000",
		"0200" + "4000000000000000" + "16" + "4111111111111111",
		"0200" + "6020000002008000" + "164111111111111111" + "000000" + "123456" + "00" + "EUR",
		"0200" + "0000000000010000" + "013" + "hello, world!",
		"0200" + "0000000000000200" + "011" + "9F2608AABBCCDDEEFF0011",
		"0800" + "8000000000000000" + "0400000000000001" + "301" + "AABBCCDDEEFF0011",
		"0800" + "8000000000000000" + "8000000000000000" + "00",
		"0200" + "4000000000000000" + "1A" + "4111111111111111",
		"0200" + "2000000000000000" + "000000" + "junk",
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		msg, err := Unpack(data, testSpec)
		if err != nil {
			var fe *FieldError
			if !errors.As(err, &fe) {
				t.Fatalf("Unpack(%q) error %T is not *FieldError: %v", data, err, err)
			}
			if fe.Offset < 0 || fe.Offset > len(data) {
				t.Fatalf("Unpack(%q) reported offset %d outside input", data, fe.Offset)
			}
			if msg != nil {
				t.Fatalf("Unpack(%q) returned a message with error %v", data, err)
			}
			return
		}

		packed, err := Pack(msg, testSpec)
		if err != nil {
			t.Fatalf("Pack(Unpack(%q)) error = %v", data, err)
		}
		// Unpack tolerates a secondary bitmap with no bits set, which Pack
		// never emits; apart from that the bytes must match up to hex case.
		switch {
		case len(packed) == len(data):
			if !bytes.EqualFold(packed, data) {
				t.Fatalf("Pack(Unpack(%q)) = %q", data, packed)
			}
		case len(packed) != len(data)-bitmapHexLen:
			t.Fatalf("Pack(Unpack(%q)) = %q, unexpected length", data, packed)
		}

		again, err := Unpack(packed, testSpec)
		if err != nil {
			t.Fatalf("Unpack(Pack(Unpack(%q))) error = %v", data, err)
		}
		if again.MTI != msg.MTI || !reflect.DeepEqual(again.fields, msg.fields) {
			t.Fatalf("round trip of %q changed the message: %v -> %v", data, msg, again)
		}
	})
}
