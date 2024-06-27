package iso8583

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestUnpackGolden(t *testing.T) {
	tests := []struct {
		name   string
		data   string
		mti    string
		fields map[int]string
	}{
		{
			name: "no fields",
			data: "0800" + "0000000000000000",
			mti:  "0800",
		},
		{
			name:   "fixed numeric keeps padding",
			data:   "0200" + "1000000000000000" + "000000001000",
			mti:    "0200",
			fields: map[int]string{4: "000000001000"},
		},
		{
			name:   "fixed alphanumeric keeps padding",
			data:   "0200" + "0000000000800000" + "T1      ",
			mti:    "0200",
			fields: map[int]string{41: "T1      "},
		},
		{
			name:   "llvar numeric",
			data:   "0200" + "4000000000000000" + "16" + "4111111111111111",
			mti:    "0200",
			fields: map[int]string{2: "4111111111111111"},
		},
		{
			name:   "llvar track2",
			data:   "0200" + "0000000020000000" + "28" + "4111111111111111D25121011234",
			mti:    "0200",
			fields: map[int]string{35: "4111111111111111D25121011234"},
		},
		{
			name:   "lllvar alphanumeric",
			data:   "0200" + "0000000000010000" + "013" + "hello, world!",
			mti:    "0200",
			fields: map[int]string{48: "hello, world!"},
		},
		{
			name:   "lllvar binary prefix counts bytes",
			data:   "0200" + "0000000000000200" + "011" + "9F2608aabbccddeeff0011",
			mti:    "0200",
			fields: map[int]string{55: "9F2608aabbccddeeff0011"},
		},
		{
			name:   "empty variable field",
			data:   "0200" + "0000000000010000" + "000",
			mti:    "0200",
			fields: map[int]string{48: ""},
		},
		{
			name:   "lower-case bitmap",
			data:   "0200" + "0000000000000200" + "002" + "ABCD",
			mti:    "0200",
			fields: map[int]string{55: "ABCD"},
		},
		{
			name:   "field 64 without secondary bitmap",
			data:   "0800" + "0000000000000001" + "AABBCCDDEEFF0011",
			mti:    "0800",
			fields: map[int]string{64: "AABBCCDDEEFF0011"},
		},
		{
			name:   "field 65 via secondary bitmap",
			data:   "0800" + "8000000000000000" + "8000000000000000" + "00",
			mti:    "0800",
			fields: map[int]string{65: "00"},
		},
		{
			name:   "field 128",
			data:   "0800" + "8000000000000000" + "0400000000000001" + "301" + "AABBCCDDEEFF0011",
			mti:    "0800",
			fields: map[int]string{70: "301", 128: "AABBCCDDEEFF0011"},
		},
		{
			name: "secondary bitmap with nothing set",
			data: "0800" + "8000000000000000" + "0000000000000000",
			mti:  "0800",
		},
		{
			name: "several fields",
			data: "0210" + "6020000002008000" + "164111111111111111" + "000000" + "123456" + "00" + "EUR",
			mti:  "0210",
			fields: map[int]string{
				2: "4111111111111111", 3: "000000", 11: "123456", 39: "00", 49: "EUR",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := Unpack([]byte(tt.data), testSpec)
			if err != nil {
				t.Fatalf("Unpack() error = %v", err)
			}
			if msg.MTI != tt.mti {
				t.Errorf("MTI = %q, want %q", msg.MTI, tt.mti)
			}
			want := tt.fields
			if want == nil {
				want = map[int]string{}
			}
			if !reflect.DeepEqual(msg.fields, want) {
				t.Errorf("fields = %v, want %v", msg.fields, want)
			}
		})
	}
}

func TestUnpackErrors(t *testing.T) {
	tests := []struct {
		name   string
		data   string
		field  int
		offset int
		want   error
		msg    string
	}{
		{"empty input", "", 0, 0, ErrTruncated, "offset 0: unexpected end of data"},
		{"partial MTI", "02", 0, 0, ErrTruncated, ""},
		{"non-numeric MTI", "02X0" + "0000000000000000", 0, 0, ErrInvalidMTI, "MTI must be four digits"},
		{"missing bitmap", "0200", 1, 4, ErrTruncated, "bitmap at offset 4: unexpected end of data: primary bitmap needs 16 bytes, have 0"},
		{"short bitmap", "0200" + "40000000", 1, 4, ErrTruncated, ""},
		{"non-hex bitmap", "0200" + "4000-00000000000", 1, 4, ErrInvalidBitmap, "invalid bitmap"},
		{"missing secondary bitmap", "0200" + "8000000000000000", 1, 4, ErrTruncated, "secondary bitmap needs 16 bytes, have 0"},
		{"field not in spec", "0200" + "0200000000000000" + "0101120000", 7, 20, ErrNoSpec, "field 7 at offset 20: field not in spec"},
		{"fixed field truncated", "0200" + "2000000000000000" + "0000", 3, 20, ErrTruncated, "field needs 6 bytes, have 4"},
		{"llvar prefix truncated", "0200" + "4000000000000000" + "1", 2, 20, ErrTruncated, "LLVAR prefix needs 2 bytes, have 1"},
		{"lllvar prefix truncated", "0200" + "0000000000010000" + "01", 48, 20, ErrTruncated, "LLLVAR prefix needs 3 bytes"},
		{"llvar body truncated", "0200" + "4000000000000000" + "16" + "41111111", 2, 20, ErrTruncated, "field needs 16 bytes, have 8"},
		{"binary body truncated", "0200" + "0000000000001000" + "0123", 52, 20, ErrTruncated, "field needs 16 bytes, have 4"},
		{"non-numeric llvar prefix", "0200" + "4000000000000000" + "1A" + "4111111111111111", 2, 20, ErrLengthPrefix, `invalid length prefix: "1A"`},
		{"space in lllvar prefix", "0200" + "0000000000010000" + " 13" + "hello, world!", 48, 20, ErrLengthPrefix, ""},
		{"non-numeric in numeric field", "0200" + "2000000000000000" + "00000A", 3, 20, ErrInvalidChar, `'A' at position 5 in Numeric field`},
		{"digit in alpha field", "0200" + "0000000000008000" + "US1", 49, 20, ErrInvalidChar, "in Alpha field"},
		{"letter in track2", "0200" + "0000000020000000" + "08" + "4111=25X", 35, 20, ErrInvalidChar, "in Track2 field"},
		{"non-hex binary", "0200" + "0000000000001000" + "0123456789ABCDEG", 52, 20, ErrInvalidChar, "in Binary field"},
		{"trailing data", "0200" + "2000000000000000" + "000000" + "junk", 0, 26, ErrTrailingData, "offset 26: trailing data after last field"},
		{"error offset after earlier fields", "0200" + "3000000000000000" + "000000" + "00000000100X", 4, 26, ErrInvalidChar, "field 4 at offset 26"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := Unpack([]byte(tt.data), testSpec)
			if err == nil {
				t.Fatalf("Unpack() = %v, want error", msg)
			}
			if msg != nil {
				t.Errorf("Unpack() returned a message alongside error")
			}
			if !errors.Is(err, tt.want) {
				t.Errorf("error %v does not wrap %v", err, tt.want)
			}
			var fe *FieldError
			if !errors.As(err, &fe) {
				t.Fatalf("error %T is not *FieldError", err)
			}
			if fe.Field != tt.field || fe.Offset != tt.offset {
				t.Errorf("FieldError{Field: %d, Offset: %d}, want {%d, %d}", fe.Field, fe.Offset, tt.field, tt.offset)
			}
			if !strings.Contains(err.Error(), tt.msg) {
				t.Errorf("error %q does not contain %q", err, tt.msg)
			}
		})
	}
}

func TestUnpackNilSpec(t *testing.T) {
	// A nil spec is legal and simply knows no fields.
	msg, err := Unpack([]byte("0800"+"0000000000000000"), nil)
	if err != nil || msg.MTI != "0800" {
		t.Fatalf("Unpack() = %v, %v", msg, err)
	}
	if _, err := Unpack([]byte("0200"+"2000000000000000"+"000000"), nil); !errors.Is(err, ErrNoSpec) {
		t.Fatalf("Unpack() error = %v, want ErrNoSpec", err)
	}
}
