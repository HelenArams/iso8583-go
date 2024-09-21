package iso8583

import (
	"errors"
	"strings"
	"testing"
)

func TestSpecValidate(t *testing.T) {
	valid := Spec{
		2:   {Name: "PAN", Type: Numeric, Length: 19, LenType: LLVAR},
		3:   {Name: "Processing code", Type: Numeric, Length: 6},
		52:  {Name: "PIN data", Type: Binary, Length: 8},
		55:  {Name: "ICC data", Type: Binary, Length: 255, LenType: LLLVAR},
		128: {Name: "MAC", Type: Binary, Length: 8},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
	if err := (Spec{}).Validate(); err != nil {
		t.Fatalf("empty spec: Validate() = %v, want nil", err)
	}

	tests := []struct {
		name string
		spec Spec
		want string // substring of the error message
	}{
		{"field 1", Spec{1: {Type: Binary, Length: 8}}, "field 1: invalid field spec: number out of range"},
		{"field 0", Spec{0: {Type: Numeric, Length: 4}}, "field 0: invalid field spec"},
		{"field 129", Spec{129: {Type: Numeric, Length: 4}}, "field 129"},
		{"zero length", Spec{3: {Type: Numeric}}, "length must be positive, got 0"},
		{"negative length", Spec{3: {Type: Numeric, Length: -1}}, "length must be positive"},
		{"llvar too long", Spec{2: {Type: Numeric, Length: 100, LenType: LLVAR}}, "length 100 exceeds LLVAR maximum of 99"},
		{"lllvar too long", Spec{55: {Type: Binary, Length: 1000, LenType: LLLVAR}}, "exceeds LLLVAR maximum of 999"},
		{"unknown type", Spec{3: {Type: FieldType(9), Length: 6}}, "unknown type FieldType(9)"},
		{"unknown length type", Spec{3: {Type: Numeric, Length: 6, LenType: LengthType(7)}}, "unknown length type LengthType(7)"},
		{"first bad field reported", Spec{3: {Type: Numeric, Length: 6}, 4: {Type: Numeric}, 7: {Type: Numeric}}, "field 4:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if err == nil {
				t.Fatal("Validate() = nil, want error")
			}
			if !errors.Is(err, ErrInvalidSpec) {
				t.Errorf("error %v does not wrap ErrInvalidSpec", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not contain %q", err, tt.want)
			}
		})
	}
}

func TestTypeNames(t *testing.T) {
	for typ, want := range map[FieldType]string{
		Numeric: "Numeric", Alpha: "Alpha", Alphanumeric: "Alphanumeric",
		Binary: "Binary", Track2: "Track2", FieldType(42): "FieldType(42)",
	} {
		if got := typ.String(); got != want {
			t.Errorf("FieldType(%d).String() = %q, want %q", uint8(typ), got, want)
		}
	}
	for lt, want := range map[LengthType]string{
		Fixed: "Fixed", LLVAR: "LLVAR", LLLVAR: "LLLVAR", LengthType(9): "LengthType(9)",
	} {
		if got := lt.String(); got != want {
			t.Errorf("LengthType(%d).String() = %q, want %q", uint8(lt), got, want)
		}
	}
}
