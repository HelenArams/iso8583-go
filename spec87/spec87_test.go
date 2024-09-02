package spec87_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/HelenArams/iso8583-go"
	"github.com/HelenArams/iso8583-go/spec87"
)

func TestSpecIsValid(t *testing.T) {
	if err := spec87.Spec.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestSpecCoversExpectedFields(t *testing.T) {
	want := []int{2, 3, 4, 7, 11, 12, 13, 14, 22, 25, 32, 35, 37, 38, 39, 41, 42, 43, 49, 52, 54, 55, 70, 90, 128}
	if len(spec87.Spec) != len(want) {
		t.Errorf("spec has %d fields, want %d", len(spec87.Spec), len(want))
	}
	for _, f := range want {
		fs, ok := spec87.Spec[f]
		if !ok {
			t.Errorf("field %d missing", f)
			continue
		}
		if fs.Name == "" {
			t.Errorf("field %d has no name", f)
		}
	}
}

func TestBinaryLengthsCountBytes(t *testing.T) {
	for _, f := range []int{52, 128} {
		if fs := spec87.Spec[f]; fs.Type != iso8583.Binary || fs.Length != 8 {
			t.Errorf("field %d = %+v, want 8-byte Binary", f, fs)
		}
	}
}

func TestClone(t *testing.T) {
	custom := spec87.Clone()
	custom[55] = iso8583.FieldSpec{Name: "ICC data", Type: iso8583.Alphanumeric, Length: 999, LenType: iso8583.LLLVAR}
	custom[60] = iso8583.FieldSpec{Name: "Private", Type: iso8583.Alphanumeric, Length: 999, LenType: iso8583.LLLVAR}

	if spec87.Spec[55].Type != iso8583.Binary {
		t.Error("modifying the clone changed the shared spec")
	}
	if _, ok := spec87.Spec[60]; ok {
		t.Error("adding to the clone changed the shared spec")
	}
	if err := custom.Validate(); err != nil {
		t.Error(err)
	}
}

func TestAuthorizationRoundTrip(t *testing.T) {
	req := iso8583.NewMessage("0200")
	req.Set(2, "4111111111111111")
	req.Set(3, "000000")
	req.Set(4, "000000012345")
	req.Set(7, "0612153000")
	req.Set(11, "000123")
	req.Set(12, "173000")
	req.Set(13, "0612")
	req.Set(14, "2712")
	req.Set(22, "012")
	req.Set(25, "00")
	req.Set(32, "12345678901")
	req.Set(35, "4111111111111111=27121011234567890")
	req.Set(37, "416415000123")
	req.Set(41, "TERM0001")
	req.Set(42, "MERCHANT0000001")
	req.Set(43, "DEMO MERCHANT            YEREVAN      AM")
	req.Set(49, "051")
	req.Set(52, "A1B2C3D4E5F60718")
	req.Set(54, "0040840D000000012345")
	req.Set(55, "9F2608AABBCCDDEEFF00119F3602001A")
	req.Set(70, "301")
	req.Set(90, strings.Repeat("0", 42))
	req.Set(128, "0011223344556677")

	packed, err := iso8583.Pack(req, spec87.Spec)
	if err != nil {
		t.Fatalf("Pack() error = %v", err)
	}

	// MTI, primary and secondary bitmap, then field 2 with its LL prefix.
	if !strings.HasPrefix(string(packed), "0200"+"F23C048128E09600"+"0400004000000001"+"164111111111111111") {
		t.Errorf("unexpected header: %s", packed[:52])
	}

	got, err := iso8583.Unpack(packed, spec87.Spec)
	if err != nil {
		t.Fatalf("Unpack() error = %v", err)
	}
	if got.MTI != req.MTI || !reflect.DeepEqual(got.Fields(), req.Fields()) {
		t.Fatalf("round trip changed the message:\n%s\n%s", req, got)
	}
	for _, f := range req.Fields() {
		if got.Get(f) != req.Get(f) {
			t.Errorf("field %d = %q, want %q", f, got.Get(f), req.Get(f))
		}
	}
}

func TestResponseCodeIsAlphanumeric(t *testing.T) {
	// Some schemes use letters in field 39 (e.g. "N7"); the spec must allow it.
	resp := iso8583.NewMessage("0210")
	resp.Set(39, "N7")
	if _, err := iso8583.Pack(resp, spec87.Spec); err != nil {
		t.Fatal(err)
	}
}

func TestRejectsOversizedPAN(t *testing.T) {
	msg := iso8583.NewMessage("0200")
	msg.Set(2, strings.Repeat("4", 20))
	_, err := iso8583.Pack(msg, spec87.Spec)
	if !errors.Is(err, iso8583.ErrInvalidLength) {
		t.Fatalf("Pack() error = %v, want ErrInvalidLength", err)
	}
}
