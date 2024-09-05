package netio

import (
	"bytes"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestWriteMessageFrame(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteMessage(&buf, []byte("0800" + "0000000000000000")); err != nil {
		t.Fatal(err)
	}
	want := "\x00\x14" + "0800" + "0000000000000000"
	if got := buf.String(); got != want {
		t.Errorf("frame = %q, want %q", got, want)
	}
}

func TestWriteMessageErrors(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteMessage(&buf, nil); !errors.Is(err, ErrEmptyMessage) {
		t.Errorf("empty: err = %v, want ErrEmptyMessage", err)
	}
	if err := WriteMessage(&buf, make([]byte, MaxMessageSize+1)); !errors.Is(err, ErrMessageTooLarge) {
		t.Errorf("oversized: err = %v, want ErrMessageTooLarge", err)
	}
	if buf.Len() != 0 {
		t.Errorf("rejected messages must not write anything, got %d bytes", buf.Len())
	}
	if err := WriteMessage(&buf, make([]byte, MaxMessageSize)); err != nil {
		t.Errorf("maximum size should be accepted: %v", err)
	}
}

func TestWriteMessagePropagatesWriteError(t *testing.T) {
	want := errors.New("boom")
	if err := WriteMessage(failingWriter{want}, []byte("x")); !errors.Is(err, want) {
		t.Errorf("err = %v, want %v", err, want)
	}
}

func TestReadMessage(t *testing.T) {
	stream := "\x00\x03abc" + "\x00\x01z" + "\xff\xff" + strings.Repeat("m", MaxMessageSize)
	r := strings.NewReader(stream)

	for _, want := range []string{"abc", "z", strings.Repeat("m", MaxMessageSize)} {
		got, err := ReadMessage(r)
		if err != nil {
			t.Fatalf("ReadMessage() error = %v", err)
		}
		if string(got) != want {
			t.Errorf("ReadMessage() = %q, want %q", got, want)
		}
	}
	if _, err := ReadMessage(r); !errors.Is(err, io.EOF) {
		t.Errorf("after last frame: err = %v, want io.EOF", err)
	}
}

func TestReadMessageErrors(t *testing.T) {
	tests := []struct {
		name   string
		stream string
		want   error
	}{
		{"empty stream", "", io.EOF},
		{"one header byte", "\x00", io.ErrUnexpectedEOF},
		{"zero length frame", "\x00\x00", ErrEmptyMessage},
		{"body missing", "\x00\x05", io.ErrUnexpectedEOF},
		{"body short", "\x00\x05abc", io.ErrUnexpectedEOF},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ReadMessage(strings.NewReader(tt.stream))
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
			if msg != nil {
				t.Errorf("msg = %q, want nil", msg)
			}
		})
	}
}

func TestRoundTripOverPipe(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	messages := [][]byte{
		[]byte("0800" + "0000000000000000"),
		[]byte("0200" + "4000000000000000" + "16" + "4111111111111111"),
		bytes.Repeat([]byte{0x00}, 300), // binary content and a length above 255
	}

	errc := make(chan error, 1)
	go func() {
		for _, m := range messages {
			if err := WriteMessage(client, m); err != nil {
				errc <- err
				return
			}
		}
		errc <- client.Close()
	}()

	_ = server.SetDeadline(time.Now().Add(5 * time.Second))
	for i, want := range messages {
		got, err := ReadMessage(server)
		if err != nil {
			t.Fatalf("message %d: %v", i, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("message %d = %q, want %q", i, got, want)
		}
	}
	if _, err := ReadMessage(server); !errors.Is(err, io.EOF) {
		t.Errorf("after close: err = %v, want io.EOF", err)
	}
	if err := <-errc; err != nil {
		t.Fatal(err)
	}
}

func TestReadMessageAcrossFragmentedReads(t *testing.T) {
	// Deliver the frame one byte at a time to make sure ReadMessage waits
	// for the whole header and body.
	frame := "\x00\x04" + "0800"
	got, err := ReadMessage(oneByteReader{strings.NewReader(frame)})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "0800" {
		t.Errorf("ReadMessage() = %q", got)
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

type oneByteReader struct{ r io.Reader }

func (o oneByteReader) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return o.r.Read(p)
}
