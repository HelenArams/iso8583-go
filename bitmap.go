package iso8583

import (
	"encoding/hex"
	"fmt"
)

const (
	// bitmapBits is the number of fields one bitmap announces.
	bitmapBits = 64
	// bitmapHexLen is the length of one hex-encoded bitmap on the wire.
	bitmapHexLen = bitmapBits / 4
)

const hexUpper = "0123456789ABCDEF"

// appendBitmap appends the hex-encoded primary bitmap to dst, followed by the
// secondary bitmap when any field above 64 is present. Bit 1 of the primary
// bitmap is set exactly when the secondary bitmap follows. fields must be
// sorted ascending and lie within 2..MaxField.
func appendBitmap(dst []byte, fields []int) []byte {
	var bits [MaxField / 8]byte
	for _, f := range fields {
		bits[(f-1)/8] |= 0x80 >> uint((f-1)%8)
	}

	n := bitmapBits / 8
	if len(fields) > 0 && fields[len(fields)-1] > bitmapBits {
		bits[0] |= 0x80
		n = len(bits)
	}
	for _, b := range bits[:n] {
		dst = append(dst, hexUpper[b>>4], hexUpper[b&0x0F])
	}
	return dst
}

// TODO: some hosts send the bitmap as 8 raw bytes instead of 16 hex chars,
// would be nice to support that via an option on the spec.

// decodeBitmap parses the bitmap(s) at the start of data and returns the
// present field numbers in ascending order together with the number of bytes
// consumed. Bit 1 only announces the secondary bitmap and is not reported as
// a field. Hex digits of either case are accepted.
func decodeBitmap(data []byte) (fields []int, n int, err error) {
	var bits [MaxField / 8]byte

	if len(data) < bitmapHexLen {
		return nil, 0, fmt.Errorf("%w: primary bitmap needs %d bytes, have %d", ErrTruncated, bitmapHexLen, len(data))
	}
	if _, err := hex.Decode(bits[:8], data[:bitmapHexLen]); err != nil {
		return nil, 0, fmt.Errorf("%w: primary bitmap: %w", ErrInvalidBitmap, err)
	}
	n = bitmapHexLen
	count := bitmapBits

	if bits[0]&0x80 != 0 {
		if len(data) < 2*bitmapHexLen {
			return nil, 0, fmt.Errorf("%w: secondary bitmap needs %d bytes, have %d", ErrTruncated, bitmapHexLen, len(data)-bitmapHexLen)
		}
		if _, err := hex.Decode(bits[8:], data[bitmapHexLen:2*bitmapHexLen]); err != nil {
			return nil, 0, fmt.Errorf("%w: secondary bitmap: %w", ErrInvalidBitmap, err)
		}
		n = 2 * bitmapHexLen
		count = MaxField
	}

	fields = make([]int, 0, 16)
	for i := 1; i < count; i++ { // bit 0 is field 1, the secondary bitmap flag
		if bits[i/8]&(0x80>>uint(i%8)) != 0 {
			fields = append(fields, i+1)
		}
	}
	return fields, n, nil
}
