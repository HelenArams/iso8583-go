package iso8583

import (
	"reflect"
	"testing"
)

func TestMessageAccessors(t *testing.T) {
	m := NewMessage("0200")
	if m.Has(2) {
		t.Fatal("new message should have no fields")
	}

	m.Set(2, "4111111111111111")
	m.Set(11, "000001")
	m.Set(4, "")

	if got := m.Get(2); got != "4111111111111111" {
		t.Errorf("Get(2) = %q", got)
	}
	if !m.Has(4) || m.Get(4) != "" {
		t.Error("empty value should still count as present")
	}
	if m.Has(3) || m.Get(3) != "" {
		t.Error("absent field should report empty and not present")
	}

	m.Set(11, "000002")
	if got := m.Get(11); got != "000002" {
		t.Errorf("Set should replace: Get(11) = %q", got)
	}

	m.Unset(11)
	m.Unset(99) // unsetting an absent field is a no-op
	if m.Has(11) {
		t.Error("Unset(11) did not remove the field")
	}

	if got, want := m.Fields(), []int{2, 4}; !reflect.DeepEqual(got, want) {
		t.Errorf("Fields() = %v, want %v", got, want)
	}
}

func TestMessageZeroValue(t *testing.T) {
	var m Message
	if m.Has(2) || m.Get(2) != "" || len(m.Fields()) != 0 {
		t.Fatal("zero Message should behave as empty")
	}
	m.Unset(2)
	m.Set(2, "4111")
	if !m.Has(2) {
		t.Fatal("Set on zero Message should initialise storage")
	}
	if got := m.String(); got != "MTI \n  002  ****" {
		t.Errorf("String() = %q", got)
	}
}

func TestMessageFieldsSorted(t *testing.T) {
	m := NewMessage("0800")
	for _, f := range []int{128, 70, 2, 64, 65, 11, 3} {
		m.Set(f, "x")
	}
	if got, want := m.Fields(), []int{2, 3, 11, 64, 65, 70, 128}; !reflect.DeepEqual(got, want) {
		t.Errorf("Fields() = %v, want %v", got, want)
	}
}

func TestMessageString(t *testing.T) {
	m := NewMessage("0200")
	m.Set(2, "4111111111111111")
	m.Set(3, "000000")
	m.Set(35, "4111111111111111=2512101123400001")
	m.Set(52, "0123456789ABCDEF")
	m.Set(128, "FFFFFFFFFFFFFFFF")

	want := "MTI 0200" +
		"\n  002  411111******1111" +
		"\n  003  000000" +
		"\n  035  411111******1111=****************" +
		"\n  052  ****************" +
		"\n  128  FFFFFFFFFFFFFFFF"
	if got := m.String(); got != want {
		t.Errorf("String() =\n%s\nwant\n%s", got, want)
	}
}

func TestMaskPAN(t *testing.T) {
	tests := []struct{ pan, want string }{
		{"", ""},
		{"4111", "****"},
		{"4111111111", "**********"},                   // 10 digits: nothing to keep
		{"41111111111", "411111*1111"},                 // 11 digits: one masked
		{"4111111111111111", "411111******1111"},       // 16 digits
		{"4111111111111111111", "411111*********1111"}, // 19 digits
	}
	for _, tt := range tests {
		if got := maskPAN(tt.pan); got != tt.want {
			t.Errorf("maskPAN(%q) = %q, want %q", tt.pan, got, tt.want)
		}
	}
}

func TestMaskTrack2(t *testing.T) {
	tests := []struct{ track, want string }{
		{"", ""},
		{"4111111111111111=2512101", "411111******1111=*******"},
		{"4111111111111111D2512101", "411111******1111D*******"},
		{"4111111111111111", "411111******1111"}, // no separator
		{"=2512", "=****"},                       // empty PAN
	}
	for _, tt := range tests {
		if got := maskTrack2(tt.track); got != tt.want {
			t.Errorf("maskTrack2(%q) = %q, want %q", tt.track, got, tt.want)
		}
	}
}
