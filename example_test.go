package iso8583_test

import (
	"errors"
	"fmt"

	"github.com/HelenArams/iso8583-go"
	"github.com/HelenArams/iso8583-go/spec87"
)

func ExamplePack() {
	msg := iso8583.NewMessage("0200")
	msg.Set(2, "4111111111111111")
	msg.Set(3, "000000")
	msg.Set(4, "1000") // zero-padded to 12 digits on the wire
	msg.Set(11, "000001")
	msg.Set(41, "TERM0001")
	msg.Set(49, "978")

	data, err := iso8583.Pack(msg, spec87.Spec)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", data)
	// Output:
	// 02007020000000808000164111111111111111000000000000001000000001TERM0001978
}

func ExampleUnpack() {
	data := []byte("0210" + "7020000002808000" +
		"164111111111111111" + "000000" + "000000001000" + "000001" + "00" + "TERM0001" + "978")

	msg, err := iso8583.Unpack(data, spec87.Spec)
	if err != nil {
		panic(err)
	}
	fmt.Println(msg.MTI, msg.Get(39))
	fmt.Println(msg) // card data is masked
	// Output:
	// 0210 00
	// MTI 0210
	//   002  411111******1111
	//   003  000000
	//   004  000000001000
	//   011  000001
	//   039  00
	//   041  TERM0001
	//   049  978
}

func ExampleFieldError() {
	// Field 3 is declared numeric; "00000A" is not.
	data := []byte("0200" + "2000000000000000" + "00000A")

	_, err := iso8583.Unpack(data, spec87.Spec)

	var fe *iso8583.FieldError
	if errors.As(err, &fe) {
		fmt.Println("field:", fe.Field, "offset:", fe.Offset)
		fmt.Println("numeric problem:", errors.Is(err, iso8583.ErrInvalidChar))
	}
	fmt.Println(err)
	// Output:
	// field: 3 offset: 20
	// numeric problem: true
	// iso8583: field 3 at offset 20: invalid character: 'A' at position 5 in Numeric field
}

func ExampleSpec() {
	// Start from the standard layout and override what your interface changes.
	spec := spec87.Clone()
	spec[55] = iso8583.FieldSpec{Name: "ICC data", Type: iso8583.Alphanumeric, Length: 999, LenType: iso8583.LLLVAR}
	spec[60] = iso8583.FieldSpec{Name: "Private use", Type: iso8583.Alphanumeric, Length: 999, LenType: iso8583.LLLVAR}

	fmt.Println(spec.Validate())
	// Output:
	// <nil>
}
