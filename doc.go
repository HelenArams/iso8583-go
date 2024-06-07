// Package iso8583 implements a minimal, dependency-free packer and unpacker
// for ISO 8583 (1987) messages in ASCII encoding.
//
// A message consists of a four-digit message type indicator (MTI), a
// hex-encoded primary bitmap announcing which data elements are present, an
// optional secondary bitmap for elements 65 to 128, and the elements
// themselves in ascending order. The layout of each element (type, length and
// whether the length is fixed or carried in an LL/LLL prefix) is described by
// a Spec, which the caller supplies to Pack and Unpack.
//
// The package covers the subset of the standard needed to build test tooling
// and to learn how the wire format works. It is not a certified
// implementation of any card-scheme interface.
package iso8583
