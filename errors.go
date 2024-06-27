package iso8583

import (
	"errors"
	"fmt"
)

// Sentinel errors. Errors returned by Pack and Unpack wrap one of these and
// can be tested with errors.Is.
var (
	// ErrInvalidSpec is returned by Spec.Validate when an entry is unusable.
	ErrInvalidSpec = errors.New("invalid field spec")
	// ErrInvalidMTI reports a message type indicator that is not four digits.
	ErrInvalidMTI = errors.New("MTI must be four digits")
	// ErrInvalidField reports a field number outside 2..MaxField.
	ErrInvalidField = errors.New("field number out of range (2..128)")
	// ErrNoSpec reports a field that the spec does not describe.
	ErrNoSpec = errors.New("field not in spec")
	// ErrInvalidBitmap reports a bitmap that is not valid hexadecimal.
	ErrInvalidBitmap = errors.New("invalid bitmap")
	// ErrInvalidLength reports a value whose length does not fit its spec.
	ErrInvalidLength = errors.New("invalid length")
	// ErrInvalidChar reports a character not allowed by the field type.
	ErrInvalidChar = errors.New("invalid character")
	// ErrLengthPrefix reports an LL or LLL prefix that is not numeric.
	ErrLengthPrefix = errors.New("invalid length prefix")
	// ErrTrailingData reports bytes left over after the last field.
	ErrTrailingData = errors.New("trailing data after last field")
	// ErrTruncated reports input that ends before a message part is complete.
	ErrTruncated = errors.New("unexpected end of data")
)

// FieldError describes a problem with one part of a message. Field is the
// data element number; it is 1 for the bitmaps and 0 for the MTI or for the
// message as a whole. Offset is the byte position within the packed message
// at which that part starts.
type FieldError struct {
	Field  int
	Offset int
	Err    error
}

func (e *FieldError) Error() string {
	switch e.Field {
	case 0:
		return fmt.Sprintf("iso8583: offset %d: %v", e.Offset, e.Err)
	case 1:
		return fmt.Sprintf("iso8583: bitmap at offset %d: %v", e.Offset, e.Err)
	default:
		return fmt.Sprintf("iso8583: field %d at offset %d: %v", e.Field, e.Offset, e.Err)
	}
}

// Unwrap returns the underlying sentinel error.
func (e *FieldError) Unwrap() error { return e.Err }
