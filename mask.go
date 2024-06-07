package iso8583

import "strings"

// Fields whose content must never appear in logs.
const (
	fieldPAN      = 2
	fieldTrack2   = 35
	fieldPINBlock = 52
)

// maskField returns value with cardholder data replaced by asterisks.
func maskField(field int, value string) string {
	switch field {
	case fieldPAN:
		return maskPAN(value)
	case fieldTrack2:
		return maskTrack2(value)
	case fieldPINBlock:
		return strings.Repeat("*", len(value))
	default:
		return value
	}
}

// maskPAN keeps the first six and last four digits of a PAN, the most a
// receipt is allowed to show, and hides everything in between. Values too
// short to have a middle part are masked completely.
func maskPAN(pan string) string {
	const head, tail = 6, 4
	if len(pan) <= head+tail {
		return strings.Repeat("*", len(pan))
	}
	return pan[:head] + strings.Repeat("*", len(pan)-head-tail) + pan[len(pan)-tail:]
}

// maskTrack2 masks the PAN in front of the field separator ('=' or 'D') and
// hides everything after it: expiry date, service code and discretionary
// data. Track data without a separator is treated as a bare PAN.
func maskTrack2(track string) string {
	sep := strings.IndexAny(track, "=D")
	if sep < 0 {
		return maskPAN(track)
	}
	return maskPAN(track[:sep]) + track[sep:sep+1] + strings.Repeat("*", len(track)-sep-1)
}
