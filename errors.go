package iso8583

import "errors"

// Sentinel errors. Errors returned by Pack and Unpack wrap one of these and
// can be tested with errors.Is.
var (
	// ErrInvalidSpec is returned by Spec.Validate when an entry is unusable.
	ErrInvalidSpec = errors.New("invalid field spec")
	// ErrInvalidBitmap reports a bitmap that is not valid hexadecimal.
	ErrInvalidBitmap = errors.New("invalid bitmap")
	// ErrTruncated reports input that ends before a message part is complete.
	ErrTruncated = errors.New("unexpected end of data")
)
