package iso8583

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestAppendBitmap(t *testing.T) {
	tests := []struct {
		name   string
		fields []int
		want   string
	}{
		{"no fields", nil, "0000000000000000"},
		{"field 2", []int{2}, "4000000000000000"},
		{"field 64", []int{64}, "0000000000000001"},
		{"field 65 forces secondary", []int{65}, "8000000000000000" + "8000000000000000"},
		{"field 128", []int{128}, "8000000000000000" + "0000000000000001"},
		{"typical request", []int{2, 3, 4, 11, 41, 49}, "7020000000808000"},
		{"all primary", seq(2, 64), "7FFFFFFFFFFFFFFF"},
		{"all fields", seq(2, 128), "FFFFFFFFFFFFFFFF" + "FFFFFFFFFFFFFFFF"},
		{"mixed", []int{2, 64, 65, 128}, "C000000000000001" + "8000000000000001"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendBitmap(nil, tt.fields)
			if string(got) != tt.want {
				t.Errorf("appendBitmap(%v) = %s, want %s", tt.fields, got, tt.want)
			}
		})
	}
}

func TestAppendBitmapKeepsPrefix(t *testing.T) {
	got := appendBitmap([]byte("0200"), []int{2})
	if string(got) != "02004000000000000000" {
		t.Errorf("appendBitmap did not append to existing data: %s", got)
	}
}

func TestDecodeBitmap(t *testing.T) {
	tests := []struct {
		name   string
		data   string
		fields []int
		n      int
	}{
		{"no fields", "0000000000000000", []int{}, 16},
		{"field 2", "4000000000000000", []int{2}, 16},
		{"field 64", "0000000000000001", []int{64}, 16},
		{"field 65", "8000000000000000" + "8000000000000000", []int{65}, 32},
		{"field 128", "8000000000000000" + "0000000000000001", []int{128}, 32},
		{"lower-case hex", "c000000000000001" + "8000000000000001", []int{2, 64, 65, 128}, 32},
		{"empty secondary is tolerated", "8000000000000000" + "0000000000000000", []int{}, 32},
		{"trailing data ignored", "4000000000000000" + "16411111", []int{2}, 16},
		{"all fields", strings.Repeat("F", 32), seq(2, 128), 32},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields, n, err := decodeBitmap([]byte(tt.data))
			if err != nil {
				t.Fatalf("decodeBitmap() error = %v", err)
			}
			if n != tt.n {
				t.Errorf("consumed %d bytes, want %d", n, tt.n)
			}
			if !reflect.DeepEqual(fields, tt.fields) {
				t.Errorf("fields = %v, want %v", fields, tt.fields)
			}
		})
	}
}

func TestDecodeBitmapErrors(t *testing.T) {
	tests := []struct {
		name string
		data string
		want error
	}{
		{"empty", "", ErrTruncated},
		{"short primary", "40000000", ErrTruncated},
		{"missing secondary", "8000000000000000", ErrTruncated},
		{"short secondary", "8000000000000000" + "00000000", ErrTruncated},
		{"non-hex primary", "4G00000000000000", ErrInvalidBitmap},
		{"non-hex secondary", "8000000000000000" + "000000000000000Z", ErrInvalidBitmap},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := decodeBitmap([]byte(tt.data))
			if !errors.Is(err, tt.want) {
				t.Errorf("decodeBitmap() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestBitmapRoundTrip(t *testing.T) {
	for _, fields := range [][]int{
		{}, {2}, {64}, {65}, {128}, {2, 64}, {2, 65}, {64, 65}, {2, 64, 65, 128}, seq(2, 128),
	} {
		encoded := appendBitmap(nil, fields)
		decoded, n, err := decodeBitmap(encoded)
		if err != nil {
			t.Fatalf("fields %v: decode error %v", fields, err)
		}
		if n != len(encoded) {
			t.Errorf("fields %v: consumed %d of %d bytes", fields, n, len(encoded))
		}
		if !reflect.DeepEqual(decoded, fields) {
			t.Errorf("fields %v: round trip gave %v", fields, decoded)
		}
	}
}

// seq returns the integers from lo to hi inclusive.
func seq(lo, hi int) []int {
	s := make([]int, 0, hi-lo+1)
	for i := lo; i <= hi; i++ {
		s = append(s, i)
	}
	return s
}
