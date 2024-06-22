package iso8583

import (
	"errors"
	"strings"
	"testing"
)

// testSpec exercises every field type and length type.
var testSpec = Spec{
	2:   {Name: "PAN", Type: Numeric, Length: 19, LenType: LLVAR},
	3:   {Name: "Processing code", Type: Numeric, Length: 6},
	4:   {Name: "Amount", Type: Numeric, Length: 12},
	11:  {Name: "STAN", Type: Numeric, Length: 6},
	35:  {Name: "Track 2", Type: Track2, Length: 37, LenType: LLVAR},
	39:  {Name: "Response code", Type: Alphanumeric, Length: 2},
	41:  {Name: "Terminal", Type: Alphanumeric, Length: 8},
	48:  {Name: "Additional data", Type: Alphanumeric, Length: 999, LenType: LLLVAR},
	49:  {Name: "Currency", Type: Alpha, Length: 3},
	52:  {Name: "PIN block", Type: Binary, Length: 8},
	55:  {Name: "ICC data", Type: Binary, Length: 255, LenType: LLLVAR},
	64:  {Name: "MAC", Type: Binary, Length: 8},
	65:  {Name: "Extended bitmap", Type: Binary, Length: 1},
	70:  {Name: "NMI code", Type: Numeric, Length: 3},
	128: {Name: "MAC", Type: Binary, Length: 8},
}

func TestPackGolden(t *testing.T) {
	tests := []struct {
		name   string
		mti    string
		fields map[int]string
		want   string
	}{
		{
			name: "no fields",
			mti:  "0800",
			want: "0800" + "0000000000000000",
		},
		{
			name:   "fixed numeric",
			mti:    "0200",
			fields: map[int]string{3: "000000"},
			want:   "0200" + "2000000000000000" + "000000",
		},
		{
			name:   "fixed numeric is zero padded",
			mti:    "0200",
			fields: map[int]string{4: "1000"},
			want:   "0200" + "1000000000000000" + "000000001000",
		},
		{
			name:   "fixed alphanumeric is space padded",
			mti:    "0200",
			fields: map[int]string{41: "T1"},
			want:   "0200" + "0000000000800000" + "T1      ",
		},
		{
			name:   "fixed alpha",
			mti:    "0200",
			fields: map[int]string{49: "USD"},
			want:   "0200" + "0000000000008000" + "USD",
		},
		{
			name:   "fixed binary as hex",
			mti:    "0200",
			fields: map[int]string{52: "0123456789abcdef"},
			want:   "0200" + "0000000000001000" + "0123456789abcdef",
		},
		{
			name:   "llvar numeric",
			mti:    "0200",
			fields: map[int]string{2: "4111111111111111"},
			want:   "0200" + "4000000000000000" + "16" + "4111111111111111",
		},
		{
			name:   "llvar track2",
			mti:    "0200",
			fields: map[int]string{35: "4111111111111111=25121011234"},
			want:   "0200" + "0000000020000000" + "28" + "4111111111111111=25121011234",
		},
		{
			name:   "lllvar alphanumeric",
			mti:    "0200",
			fields: map[int]string{48: "hello, world!"},
			want:   "0200" + "0000000000010000" + "013" + "hello, world!",
		},
		{
			name:   "lllvar binary counts bytes",
			mti:    "0200",
			fields: map[int]string{55: "9F2608AABBCCDDEEFF0011"},
			want:   "0200" + "0000000000000200" + "011" + "9F2608AABBCCDDEEFF0011",
		},
		{
			name:   "empty variable field",
			mti:    "0200",
			fields: map[int]string{48: ""},
			want:   "0200" + "0000000000010000" + "000",
		},
		{
			name:   "field 64 stays in primary bitmap",
			mti:    "0800",
			fields: map[int]string{64: "AABBCCDDEEFF0011"},
			want:   "0800" + "0000000000000001" + "AABBCCDDEEFF0011",
		},
		{
			name:   "field 65 adds secondary bitmap",
			mti:    "0800",
			fields: map[int]string{65: "00"},
			want:   "0800" + "8000000000000000" + "8000000000000000" + "00",
		},
		{
			name:   "field 128",
			mti:    "0800",
			fields: map[int]string{70: "301", 128: "AABBCCDDEEFF0011"},
			want:   "0800" + "8000000000000000" + "0400000000000001" + "301" + "AABBCCDDEEFF0011",
		},
		{
			name: "fields are emitted in ascending order",
			mti:  "0200",
			fields: map[int]string{
				49: "EUR", 2: "4111111111111111", 11: "123456", 3: "000000",
			},
			want: "0200" + "6020000000008000" + "164111111111111111" + "000000" + "123456" + "EUR",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewMessage(tt.mti)
			for f, v := range tt.fields {
				msg.Set(f, v)
			}
			got, err := Pack(msg, testSpec)
			if err != nil {
				t.Fatalf("Pack() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("Pack() =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

func TestPackErrors(t *testing.T) {
	tests := []struct {
		name   string
		mti    string
		fields map[int]string
		field  int // expected FieldError.Field
		offset int // expected FieldError.Offset
		want   error
		msg    string // substring expected in the error text
	}{
		{"empty MTI", "", nil, 0, 0, ErrInvalidMTI, "offset 0: MTI must be four digits"},
		{"short MTI", "200", nil, 0, 0, ErrInvalidMTI, ""},
		{"long MTI", "02000", nil, 0, 0, ErrInvalidMTI, ""},
		{"alpha MTI", "02A0", nil, 0, 0, ErrInvalidMTI, ""},
		{"field 1 is reserved", "0200", map[int]string{1: "00"}, 1, 0, ErrInvalidField, "bitmap at offset 0"},
		{"field 0", "0200", map[int]string{0: "x"}, 0, 0, ErrInvalidField, ""},
		{"field 129", "0200", map[int]string{129: "x"}, 129, 0, ErrInvalidField, "field 129 at offset 0"},
		{"negative field", "0200", map[int]string{-3: "x"}, -3, 0, ErrInvalidField, ""},
		{"no spec", "0200", map[int]string{7: "0101120000"}, 7, 0, ErrNoSpec, "field 7 at offset 0: field not in spec"},
		{"fixed too long", "0200", map[int]string{3: "0000001"}, 3, 20, ErrInvalidLength, "value is 7, fixed length is 6"},
		{"non-numeric in numeric", "0200", map[int]string{3: "00000A"}, 3, 20, ErrInvalidChar, `'A' at position 5 in Numeric field`},
		{"digit in alpha", "0200", map[int]string{49: "US1"}, 49, 20, ErrInvalidChar, "in Alpha field"},
		{"control char in alphanumeric", "0200", map[int]string{41: "TERM\n001"}, 41, 20, ErrInvalidChar, "in Alphanumeric field"},
		{"non-ascii in alphanumeric", "0200", map[int]string{41: "TERMÉ"}, 41, 20, ErrInvalidChar, ""},
		{"letter in track2", "0200", map[int]string{35: "4111=25X"}, 35, 20, ErrInvalidChar, "in Track2 field"},
		{"non-hex binary", "0200", map[int]string{52: "0123456789ABCDEG"}, 52, 20, ErrInvalidChar, "in Binary field"},
		{"odd hex digits", "0200", map[int]string{52: "0123456789ABCDE"}, 52, 20, ErrInvalidLength, "odd number of hex digits"},
		{"short binary is not padded", "0200", map[int]string{52: "0123"}, 52, 20, ErrInvalidLength, "value is 2, fixed length is 8"},
		{"long binary", "0200", map[int]string{52: "0123456789ABCDEF00"}, 52, 20, ErrInvalidLength, "value is 9, fixed length is 8"},
		{"short track2 fixed", "0200", map[int]string{65: ""}, 65, 36, ErrInvalidLength, ""},
		{"llvar too long", "0200", map[int]string{2: strings.Repeat("4", 20)}, 2, 20, ErrInvalidLength, "value is 20, maximum is 19"},
		{"lllvar binary too long", "0200", map[int]string{55: strings.Repeat("AB", 256)}, 55, 20, ErrInvalidLength, "value is 256, maximum is 255"},
		{"offset points past earlier fields", "0200", map[int]string{3: "000000", 4: "1", 11: "ABC"}, 11, 38, ErrInvalidChar, "field 11 at offset 38"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewMessage(tt.mti)
			for f, v := range tt.fields {
				msg.Set(f, v)
			}
			out, err := Pack(msg, testSpec)
			if err == nil {
				t.Fatalf("Pack() = %s, want error", out)
			}
			if out != nil {
				t.Errorf("Pack() returned data %q alongside error", out)
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

func TestPackNilMessage(t *testing.T) {
	if _, err := Pack(nil, testSpec); err == nil {
		t.Fatal("Pack(nil) should fail")
	}
}

func TestPackLengthPrefixOverflow(t *testing.T) {
	// A spec that fails Validate must still not produce a corrupt prefix.
	spec := Spec{48: {Type: Alphanumeric, Length: 150, LenType: LLVAR}}
	msg := NewMessage("0200")
	msg.Set(48, strings.Repeat("x", 120))
	_, err := Pack(msg, spec)
	if !errors.Is(err, ErrInvalidLength) {
		t.Fatalf("Pack() error = %v, want ErrInvalidLength", err)
	}
	if !strings.Contains(err.Error(), "LLVAR prefix cannot exceed 99") {
		t.Errorf("unexpected message: %v", err)
	}
}

func TestAppendLength(t *testing.T) {
	tests := []struct {
		n, width int
		want     string
	}{
		{0, 2, "00"}, {7, 2, "07"}, {99, 2, "99"},
		{0, 3, "000"}, {42, 3, "042"}, {999, 3, "999"},
	}
	for _, tt := range tests {
		if got := appendLength(nil, tt.n, tt.width); string(got) != tt.want {
			t.Errorf("appendLength(%d, %d) = %s, want %s", tt.n, tt.width, got, tt.want)
		}
	}
}
