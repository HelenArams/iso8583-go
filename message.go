package iso8583

import (
	"fmt"
	"sort"
	"strings"
)

// MaxField is the highest data element number addressable with a primary and
// a secondary bitmap.
const MaxField = 128

// Message is an ISO 8583 message: a four-digit message type indicator and a
// set of data elements keyed by field number (2..128).
//
// Field 1 is the secondary bitmap. It is derived from the other fields by
// Pack and consumed by Unpack, so it cannot be set directly. The zero value
// is ready to use. A Message is not safe for concurrent use.
type Message struct {
	// MTI is the message type indicator, for example "0200".
	MTI string

	fields map[int]string
}

// NewMessage returns an empty message with the given MTI.
func NewMessage(mti string) *Message {
	return &Message{MTI: mti, fields: make(map[int]string)}
}

// Set stores value in field, replacing any previous value. Field numbers and
// values are validated against the spec when the message is packed.
func (m *Message) Set(field int, value string) {
	if m.fields == nil {
		m.fields = make(map[int]string)
	}
	m.fields[field] = value
}

// Get returns the value of field, or "" when it is not present. Use Has to
// tell an absent field from an empty one.
func (m *Message) Get(field int) string {
	return m.fields[field]
}

// Has reports whether field is present.
func (m *Message) Has(field int) bool {
	_, ok := m.fields[field]
	return ok
}

// Unset removes field from the message.
func (m *Message) Unset(field int) {
	delete(m.fields, field)
}

// Fields returns the numbers of all present fields in ascending order.
func (m *Message) Fields() []int {
	fields := make([]int, 0, len(m.fields))
	for f := range m.fields {
		fields = append(fields, f)
	}
	sort.Ints(fields)
	return fields
}

// String renders the message one field per line with card data masked, so
// it is safe to log: the PAN (field 2) and the PAN part of track 2 (field 35)
// keep only their first six and last four digits, the remainder of track 2
// and the PIN block (field 52) are hidden entirely.
func (m *Message) String() string {
	var b strings.Builder
	b.WriteString("MTI ")
	b.WriteString(m.MTI)
	for _, f := range m.Fields() {
		fmt.Fprintf(&b, "\n  %03d  %s", f, maskField(f, m.fields[f]))
	}
	return b.String()
}
