// Package spec87 provides a ready-made iso8583.Spec for the data elements of
// ISO 8583:1987 that appear in most card authorization and network
// management traffic, using ASCII encoding throughout.
//
// Real interfaces always deviate from the standard in places (field 55 and
// the private-use fields 60-63 and 120-127 in particular), so treat Spec as
// a starting point: call Clone and adjust the entries you need.
package spec87

import (
	"maps"

	"github.com/HelenArams/iso8583-go"
)

// Spec describes 25 commonly used data elements. Binary fields hold hex
// digits and their lengths count bytes, so field 52 (64 bits) is 8.
var Spec = iso8583.Spec{
	2:   {Name: "Primary account number", Type: iso8583.Numeric, Length: 19, LenType: iso8583.LLVAR},
	3:   {Name: "Processing code", Type: iso8583.Numeric, Length: 6},
	4:   {Name: "Amount, transaction", Type: iso8583.Numeric, Length: 12},
	7:   {Name: "Transmission date and time", Type: iso8583.Numeric, Length: 10},
	11:  {Name: "System trace audit number", Type: iso8583.Numeric, Length: 6},
	12:  {Name: "Time, local transaction", Type: iso8583.Numeric, Length: 6},
	13:  {Name: "Date, local transaction", Type: iso8583.Numeric, Length: 4},
	14:  {Name: "Date, expiration", Type: iso8583.Numeric, Length: 4},
	22:  {Name: "Point of service entry mode", Type: iso8583.Numeric, Length: 3},
	25:  {Name: "Point of service condition code", Type: iso8583.Numeric, Length: 2},
	32:  {Name: "Acquiring institution identification code", Type: iso8583.Numeric, Length: 11, LenType: iso8583.LLVAR},
	35:  {Name: "Track 2 data", Type: iso8583.Track2, Length: 37, LenType: iso8583.LLVAR},
	37:  {Name: "Retrieval reference number", Type: iso8583.Alphanumeric, Length: 12},
	38:  {Name: "Authorization identification response", Type: iso8583.Alphanumeric, Length: 6},
	39:  {Name: "Response code", Type: iso8583.Alphanumeric, Length: 2},
	41:  {Name: "Card acceptor terminal identification", Type: iso8583.Alphanumeric, Length: 8},
	42:  {Name: "Card acceptor identification code", Type: iso8583.Alphanumeric, Length: 15},
	43:  {Name: "Card acceptor name and location", Type: iso8583.Alphanumeric, Length: 40},
	49:  {Name: "Currency code, transaction", Type: iso8583.Numeric, Length: 3},
	52:  {Name: "PIN data", Type: iso8583.Binary, Length: 8},
	54:  {Name: "Additional amounts", Type: iso8583.Alphanumeric, Length: 120, LenType: iso8583.LLLVAR},
	55:  {Name: "ICC system related data", Type: iso8583.Binary, Length: 255, LenType: iso8583.LLLVAR},
	70:  {Name: "Network management information code", Type: iso8583.Numeric, Length: 3},
	90:  {Name: "Original data elements", Type: iso8583.Numeric, Length: 42},
	128: {Name: "Message authentication code", Type: iso8583.Binary, Length: 8},
}

// Clone returns a copy of Spec that can be modified without affecting other
// users of the package.
func Clone() iso8583.Spec {
	return maps.Clone(Spec)
}
