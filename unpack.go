package iso8583

// Unpack decodes an ASCII-encoded message according to spec. It reads the
// MTI, the primary bitmap, the secondary bitmap when bit 1 is set, and then
// every announced field. The whole input must be consumed; left-over bytes
// are reported as ErrTrailingData.
//
// Errors are of type *FieldError carrying the field number and the offset at
// which decoding failed, and wrap one of ErrInvalidMTI, ErrTruncated,
// ErrInvalidBitmap, ErrNoSpec, ErrLengthPrefix, ErrInvalidLength,
// ErrInvalidChar or ErrTrailingData.
func Unpack(data []byte, spec Spec) (*Message, error) {
	if len(data) < mtiLen {
		return nil, &FieldError{Err: ErrTruncated}
	}
	mti := string(data[:mtiLen])
	if !isMTI(mti) {
		return nil, &FieldError{Err: ErrInvalidMTI}
	}
	offset := mtiLen

	fields, n, err := decodeBitmap(data[offset:])
	if err != nil {
		return nil, &FieldError{Field: 1, Offset: offset, Err: err}
	}
	offset += n

	msg := &Message{MTI: mti, fields: make(map[int]string, len(fields))}
	for _, f := range fields {
		fs, ok := spec[f]
		if !ok {
			return nil, &FieldError{Field: f, Offset: offset, Err: ErrNoSpec}
		}
		value, n, err := fs.unpack(data[offset:])
		if err != nil {
			return nil, &FieldError{Field: f, Offset: offset, Err: err}
		}
		msg.fields[f] = value
		offset += n
	}

	if offset != len(data) {
		return nil, &FieldError{Offset: offset, Err: ErrTrailingData}
	}
	return msg, nil
}
