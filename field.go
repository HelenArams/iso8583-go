package iso8583

import (
	"fmt"
	"strings"
)

// allows reports whether c may appear in a field of type t.
func (t FieldType) allows(c byte) bool {
	switch t {
	case Numeric:
		return '0' <= c && c <= '9'
	case Alpha:
		return c == ' ' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
	case Alphanumeric:
		return ' ' <= c && c <= '~'
	case Binary:
		return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
	case Track2:
		return '0' <= c && c <= '9' || c == '=' || c == 'D'
	default:
		return false
	}
}

// check validates every character of value against t. Binary values must
// also have an even number of hex digits.
func (t FieldType) check(value string) error {
	for i := 0; i < len(value); i++ {
		if !t.allows(value[i]) {
			return fmt.Errorf("%w: %q at position %d in %s field", ErrInvalidChar, value[i], i, t)
		}
	}
	if t == Binary && len(value)%2 != 0 {
		return fmt.Errorf("%w: odd number of hex digits (%d)", ErrInvalidLength, len(value))
	}
	return nil
}

// units returns the length of value in the unit the spec counts: bytes for
// Binary fields, characters otherwise.
func (fs FieldSpec) units(value string) int {
	if fs.Type == Binary {
		return len(value) / 2
	}
	return len(value)
}

// pad extends a short value to the fixed length of the field: numeric
// values get leading zeros, alpha and alphanumeric values trailing spaces.
// Binary and track 2 values have no natural padding and are returned
// unchanged with ok set to false.
func (fs FieldSpec) pad(value string) (padded string, ok bool) {
	missing := fs.Length - len(value)
	switch fs.Type {
	case Numeric:
		return strings.Repeat("0", missing) + value, true
	case Alpha, Alphanumeric:
		return value + strings.Repeat(" ", missing), true
	default:
		return value, false
	}
}

// pack appends value, preceded by its length prefix for variable fields, to
// dst. Fixed fields shorter than their length are padded, see pad.
func (fs FieldSpec) pack(dst []byte, value string) ([]byte, error) {
	if err := fs.Type.check(value); err != nil {
		return dst, err
	}
	n := fs.units(value)

	if fs.LenType == Fixed {
		switch {
		case n > fs.Length:
			return dst, fmt.Errorf("%w: value is %d, fixed length is %d", ErrInvalidLength, n, fs.Length)
		case n < fs.Length:
			var ok bool
			if value, ok = fs.pad(value); !ok {
				return dst, fmt.Errorf("%w: value is %d, fixed length is %d", ErrInvalidLength, n, fs.Length)
			}
		}
		return append(dst, value...), nil
	}

	if n > fs.Length {
		return dst, fmt.Errorf("%w: value is %d, maximum is %d", ErrInvalidLength, n, fs.Length)
	}
	if n > fs.LenType.maxLength() {
		return dst, fmt.Errorf("%w: value is %d, %s prefix cannot exceed %d", ErrInvalidLength, n, fs.LenType, fs.LenType.maxLength())
	}
	dst = appendLength(dst, n, fs.LenType.digits())
	return append(dst, value...), nil
}

// appendLength appends n as a zero-padded decimal of the given width.
func appendLength(dst []byte, n, width int) []byte {
	var buf [3]byte
	for i := width - 1; i >= 0; i-- {
		buf[i] = '0' + byte(n%10)
		n /= 10
	}
	return append(dst, buf[:width]...)
}

// unpack decodes one field from the start of data and returns its value and
// the number of bytes consumed. Padding of fixed fields is preserved so that
// packing the result reproduces the input.
func (fs FieldSpec) unpack(data []byte) (value string, n int, err error) {
	length := fs.Length
	if digits := fs.LenType.digits(); digits > 0 {
		if len(data) < digits {
			return "", 0, fmt.Errorf("%w: %s prefix needs %d bytes, have %d", ErrTruncated, fs.LenType, digits, len(data))
		}
		if length, err = parseLength(data[:digits]); err != nil {
			return "", 0, err
		}
		n = digits
	}

	width := length
	if fs.Type == Binary {
		width *= 2
	}
	if len(data) < n+width {
		return "", 0, fmt.Errorf("%w: field needs %d bytes, have %d", ErrTruncated, width, len(data)-n)
	}
	value = string(data[n : n+width])
	if err := fs.Type.check(value); err != nil {
		return "", 0, err
	}
	return value, n + width, nil
}

// parseLength parses a zero-padded decimal length prefix.
func parseLength(prefix []byte) (int, error) {
	n := 0
	for _, c := range prefix {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("%w: %q", ErrLengthPrefix, prefix)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
