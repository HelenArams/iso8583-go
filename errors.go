package iso8583

import "errors"

// ErrInvalidSpec is returned by Spec.Validate when an entry is unusable.
var ErrInvalidSpec = errors.New("invalid field spec")
