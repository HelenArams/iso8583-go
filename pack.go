package iso8583

import "errors"

// mtiLen is the length of the message type indicator.
const mtiLen = 4

// Pack encodes msg according to spec: the MTI, the hex-encoded bitmap (and
// secondary bitmap when any field above 64 is present) and then every field
// in ascending order. Fixed-length numeric fields are left-padded with zeros
// and alpha fields right-padded with spaces; other mismatches are errors.
//
// Errors are of type *FieldError and wrap a sentinel: ErrInvalidMTI,
// ErrInvalidField, ErrNoSpec, ErrInvalidLength or ErrInvalidChar.
func Pack(msg *Message, spec Spec) ([]byte, error) {
	if msg == nil {
		return nil, errors.New("iso8583: nil message")
	}
	if !isMTI(msg.MTI) {
		return nil, &FieldError{Err: ErrInvalidMTI}
	}

	fields := msg.Fields()
	for _, f := range fields {
		if f < 2 || f > MaxField {
			return nil, &FieldError{Field: f, Err: ErrInvalidField}
		}
		if _, ok := spec[f]; !ok {
			return nil, &FieldError{Field: f, Err: ErrNoSpec}
		}
	}

	var out []byte
	out = append(out, msg.MTI...)
	out = appendBitmap(out, fields)
	for _, f := range fields {
		start := len(out)
		var err error
		if out, err = spec[f].pack(out, msg.fields[f]); err != nil {
			return nil, &FieldError{Field: f, Offset: start, Err: err}
		}
	}
	return out, nil
}

// isMTI reports whether s is a four-digit message type indicator.
func isMTI(s string) bool {
	return len(s) == mtiLen && Numeric.check(s) == nil
}
