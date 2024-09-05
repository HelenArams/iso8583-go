// Package netio frames ISO 8583 messages for transport over a byte stream
// such as TCP. Each message is preceded by its length as a two-byte
// big-endian unsigned integer, the convention most hosts and simulators use.
//
//	+--------+--------+----------------------------+
//	| len hi | len lo | message (len bytes)        |
//	+--------+--------+----------------------------+
//
// The functions take io.Reader and io.Writer so they work equally with a
// net.Conn, a bufio wrapper or an in-memory buffer in tests.
package netio

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
	// HeaderSize is the size of the length prefix in bytes.
	HeaderSize = 2
	// MaxMessageSize is the largest message the two-byte prefix can describe.
	MaxMessageSize = 1<<16 - 1
)

var (
	// ErrMessageTooLarge is returned by WriteMessage for messages longer
	// than MaxMessageSize.
	ErrMessageTooLarge = errors.New("netio: message exceeds 65535 bytes")
	// ErrEmptyMessage is returned when a zero-length message is written or
	// a zero-length frame is read; neither is meaningful in ISO 8583.
	ErrEmptyMessage = errors.New("netio: empty message")
)

// WriteMessage writes msg to w preceded by its two-byte length. Header and
// body are written with a single Write call so that a net.Conn sends them
// in one segment where possible.
func WriteMessage(w io.Writer, msg []byte) error {
	if len(msg) == 0 {
		return ErrEmptyMessage
	}
	if len(msg) > MaxMessageSize {
		return ErrMessageTooLarge
	}
	frame := make([]byte, HeaderSize+len(msg))
	binary.BigEndian.PutUint16(frame, uint16(len(msg)))
	copy(frame[HeaderSize:], msg)
	_, err := w.Write(frame)
	return err
}

// ReadMessage reads one length-prefixed message from r and returns its body.
// It returns io.EOF when the stream ends cleanly between messages and
// io.ErrUnexpectedEOF when it ends inside a frame.
func ReadMessage(r io.Reader) ([]byte, error) {
	var header [HeaderSize]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}
	n := int(binary.BigEndian.Uint16(header[:]))
	if n == 0 {
		return nil, ErrEmptyMessage
	}

	msg := make([]byte, n)
	if _, err := io.ReadFull(r, msg); err != nil {
		if errors.Is(err, io.EOF) {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	return msg, nil
}
