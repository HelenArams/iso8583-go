package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/HelenArams/iso8583-go"
)

// options are the request parameters taken from the command line.
type options struct {
	addr     string
	mti      string
	pan      string
	expiry   string
	amount   string
	currency string
	stan     int
	terminal string
	merchant string
	timeout  time.Duration
	wire     bool
}

// Card acceptor name and location (field 43): 22 characters of name, 13 of
// city and a 2-letter country code, space padded to 40.
const acceptorNameLocation = "DEMO MERCHANT            YEREVAN      AM"

// validate rejects values that Pack would refuse or that make no sense for
// an authorization, with messages phrased for a person at a terminal.
func (o options) validate() error {
	switch {
	case len(o.mti) != 4 || !digits(o.mti):
		return fmt.Errorf("mti %q must be four digits", o.mti)
	case o.mti[2] != '0' && o.mti[2] != '2':
		return fmt.Errorf("mti %q is not a request or advice", o.mti)
	case len(o.pan) < 12 || len(o.pan) > 19 || !digits(o.pan):
		return errors.New("pan must be 12 to 19 digits")
	case o.expiry != "" && (len(o.expiry) != 4 || !digits(o.expiry)):
		return fmt.Errorf("expiry %q must be YYMM", o.expiry)
	case o.amount == "" || len(o.amount) > 12 || !digits(o.amount):
		return errors.New("amount must be 1 to 12 digits in minor units")
	case len(o.currency) != 3 || !digits(o.currency):
		return fmt.Errorf("currency %q must be a three-digit numeric code", o.currency)
	case o.stan < 1 || o.stan > 999_999:
		return fmt.Errorf("stan %d must be between 1 and 999999", o.stan)
	case o.terminal == "" || len(o.terminal) > 8:
		return errors.New("terminal must be 1 to 8 characters")
	case o.merchant == "" || len(o.merchant) > 15:
		return errors.New("merchant must be 1 to 15 characters")
	case o.timeout <= 0:
		return errors.New("timeout must be positive")
	}
	return nil
}

// buildRequest assembles the authorization request described by o, using
// now for the date and time fields.
func buildRequest(o options, now time.Time) (*iso8583.Message, error) {
	if err := o.validate(); err != nil {
		return nil, err
	}

	m := iso8583.NewMessage(o.mti)
	m.Set(2, o.pan)
	m.Set(3, "000000") // purchase, from and to default accounts
	m.Set(4, o.amount)
	m.Set(7, now.UTC().Format("0102150405"))
	m.Set(11, fmt.Sprintf("%06d", o.stan))
	m.Set(12, now.Format("150405"))
	m.Set(13, now.Format("0102"))
	if o.expiry != "" {
		m.Set(14, o.expiry)
	}
	m.Set(22, "012") // PAN keyed manually, terminal cannot accept a PIN
	m.Set(25, "00")  // normal presentment
	m.Set(37, retrievalReference(now, o.stan))
	m.Set(41, o.terminal)
	m.Set(42, o.merchant)
	m.Set(43, acceptorNameLocation)
	m.Set(49, o.currency)
	return m, nil
}

// retrievalReference builds a 12-character RRN in the widespread YDDDHHnnnnnn
// shape: last digit of the year, day of the year, hour and the STAN.
func retrievalReference(now time.Time, stan int) string {
	return fmt.Sprintf("%d%03d%02d%06d", now.Year()%10, now.YearDay(), now.Hour(), stan)
}

func digits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
