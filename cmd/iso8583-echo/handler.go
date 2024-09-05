package main

import (
	"fmt"
	"strings"

	"github.com/HelenArams/iso8583-go"
)

// Response codes (field 39) issued by the responder.
const (
	rcApproved           = "00"
	rcDoNotHonor         = "05"
	rcInvalidTransaction = "12"
	rcFormatError        = "30"
)

// declineSuffix: amounts whose minor units end in 05 are declined, which
// gives clients a deterministic way to exercise their decline handling.
const declineSuffix = "05"

// echoedFields are copied from the request into the response unchanged.
var echoedFields = []int{2, 3, 4, 7, 11, 12, 13, 14, 32, 37, 41, 42, 49}

// requiredFields must be present in an authorization or financial request.
var requiredFields = []int{2, 3, 4, 11}

// respond builds the response to req. authCode supplies field 38 for
// approved authorizations. It fails only when req is not something a host
// would answer, such as a response or an unknown message type.
func respond(req *iso8583.Message, authCode func() string) (*iso8583.Message, error) {
	mti, err := responseMTI(req.MTI)
	if err != nil {
		return nil, err
	}

	resp := iso8583.NewMessage(mti)
	for _, f := range echoedFields {
		if req.Has(f) {
			resp.Set(f, req.Get(f))
		}
	}

	rc := decide(req)
	resp.Set(39, rc)
	if rc == rcApproved && isAuthorization(req.MTI) {
		resp.Set(38, authCode())
	}
	return resp, nil
}

// decide picks the response code for req.
func decide(req *iso8583.Message) string {
	switch req.MTI[1] {
	case '1', '2': // authorization, financial
		for _, f := range requiredFields {
			if !req.Has(f) {
				return rcFormatError
			}
		}
		if declines(req.Get(4)) {
			return rcDoNotHonor
		}
		return rcApproved
	case '4', '8': // reversal, network management
		return rcApproved
	default:
		return rcInvalidTransaction
	}
}

// declines reports whether amount, in minor units, ends in declineSuffix.
// Short amounts are padded first so that "5" and "000000000005" agree.
func declines(amount string) bool {
	if len(amount) < len(declineSuffix) {
		amount = strings.Repeat("0", len(declineSuffix)-len(amount)) + amount
	}
	return strings.HasSuffix(amount, declineSuffix)
}

// isAuthorization reports whether mti is an authorization or financial
// message, the classes that carry an authorization code when approved.
func isAuthorization(mti string) bool {
	return mti[1] == '1' || mti[1] == '2'
}

// responseMTI derives the response type from a request type by incrementing
// the message function digit: 0200 becomes 0210, 0100 becomes 0110, 0800
// becomes 0810. Advices (x2xx) are answered with x3xx.
func responseMTI(mti string) (string, error) {
	if len(mti) != 4 {
		return "", fmt.Errorf("invalid MTI %q", mti)
	}
	switch mti[2] {
	case '0', '2':
		return mti[:2] + string(mti[2]+1) + mti[3:], nil
	default:
		return "", fmt.Errorf("MTI %s is not a request or advice", mti)
	}
}
