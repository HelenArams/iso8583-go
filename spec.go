package iso8583

import (
	"fmt"
	"sort"
)

// FieldType is the data type of a field. The names mirror the attribute
// letters used in the standard (n, a, an/ans, b, z).
type FieldType uint8

const (
	// Numeric (n) holds decimal digits only.
	Numeric FieldType = iota
	// Alpha (a) holds letters and spaces.
	Alpha
	// Alphanumeric (an, ans) holds any printable ASCII character.
	Alphanumeric
	// Binary (b) is carried as hexadecimal digits. Its Length is counted in
	// bytes, so the wire representation is twice as long.
	Binary
	// Track2 (z) holds track 2 data: digits with '=' or 'D' as separator.
	Track2
)

var fieldTypeNames = [...]string{
	Numeric:      "Numeric",
	Alpha:        "Alpha",
	Alphanumeric: "Alphanumeric",
	Binary:       "Binary",
	Track2:       "Track2",
}

// String returns the type name, or "FieldType(n)" for unknown values.
func (t FieldType) String() string {
	if int(t) < len(fieldTypeNames) {
		return fieldTypeNames[t]
	}
	return fmt.Sprintf("FieldType(%d)", uint8(t))
}

// LengthType says how the length of a field is conveyed on the wire.
type LengthType uint8

const (
	// Fixed fields occupy exactly Length characters.
	Fixed LengthType = iota
	// LLVAR fields carry a two-digit length prefix (up to 99).
	LLVAR
	// LLLVAR fields carry a three-digit length prefix (up to 999).
	LLLVAR
)

var lengthTypeNames = [...]string{
	Fixed:  "Fixed",
	LLVAR:  "LLVAR",
	LLLVAR: "LLLVAR",
}

// String returns the length type name, or "LengthType(n)" for unknown values.
func (l LengthType) String() string {
	if int(l) < len(lengthTypeNames) {
		return lengthTypeNames[l]
	}
	return fmt.Sprintf("LengthType(%d)", uint8(l))
}

func (l LengthType) digits() int {
	switch l {
	case LLVAR:
		return 2
	case LLLVAR:
		return 3
	default:
		return 0
	}
}

func (l LengthType) maxLength() int {
	switch l {
	case LLVAR:
		return 99
	case LLLVAR:
		return 999
	default:
		return 0
	}
}

// FieldSpec describes the layout of one data element.
type FieldSpec struct {
	// Name is a human-readable description, e.g. "Primary account number".
	Name string
	// Type constrains the characters the field may contain.
	Type FieldType
	// Length is the exact length of a Fixed field and the maximum length of
	// an LLVAR or LLLVAR field. For Binary fields it counts bytes.
	Length int
	// LenType selects fixed-length or LLVAR/LLLVAR encoding.
	LenType LengthType
}

// Spec maps field numbers to their layout. Fields that are absent from the
// spec cannot be packed or unpacked; field 1 (the secondary bitmap) must not
// be described.
type Spec map[int]FieldSpec

// Validate checks that every entry describes a usable field: the number is
// within 2..MaxField, the type and length type are known, the length is
// positive and fits the length prefix. It returns the first problem found,
// in field order, wrapped in ErrInvalidSpec.
func (s Spec) Validate() error {
	fields := make([]int, 0, len(s))
	for f := range s {
		fields = append(fields, f)
	}
	sort.Ints(fields)

	for _, f := range fields {
		if err := s[f].validate(); err != nil {
			return fmt.Errorf("iso8583: field %d: %w: %w", f, ErrInvalidSpec, err)
		}
		if f < 2 || f > MaxField {
			return fmt.Errorf("iso8583: field %d: %w: number out of range (2..%d)", f, ErrInvalidSpec, MaxField)
		}
	}
	return nil
}

func (fs FieldSpec) validate() error {
	if int(fs.Type) >= len(fieldTypeNames) {
		return fmt.Errorf("unknown type %s", fs.Type)
	}
	if int(fs.LenType) >= len(lengthTypeNames) {
		return fmt.Errorf("unknown length type %s", fs.LenType)
	}
	if fs.Length <= 0 {
		return fmt.Errorf("length must be positive, got %d", fs.Length)
	}
	if maxLen := fs.LenType.maxLength(); maxLen > 0 && fs.Length > maxLen {
		return fmt.Errorf("length %d exceeds %s maximum of %d", fs.Length, fs.LenType, maxLen)
	}
	return nil
}
