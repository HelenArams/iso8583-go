package iso8583

import "testing"

func benchMessage() *Message {
	m := NewMessage("0200")
	m.Set(2, "4111111111111111")
	m.Set(3, "000000")
	m.Set(4, "000000001000")
	m.Set(11, "000123")
	m.Set(35, "4111111111111111=2712101123400001")
	m.Set(41, "TERM0001")
	m.Set(48, "some additional data for the lllvar path")
	m.Set(49, "EUR")
	m.Set(52, "0123456789ABCDEF")
	m.Set(55, "9F2608AABBCCDDEEFF00119F3602001A9F2701809F10120110A00003220000000000000000000000FF")
	m.Set(70, "301")
	m.Set(128, "1122334455667788")
	return m
}

func BenchmarkPack(b *testing.B) {
	m := benchMessage()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Pack(m, testSpec); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnpack(b *testing.B) {
	data, err := Pack(benchMessage(), testSpec)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Unpack(data, testSpec); err != nil {
			b.Fatal(err)
		}
	}
}
