package money

import (
	"errors"
	"strings"
)

// ParseDecimal converts a decimal string like "1250.50" into minor units of
// currency. Thousand separators and spaces are ignored. Extra fractional
// digits are accepted only if they are zeros ("1250.00" for JPY), because
// rounding would mean guessing an amount.
func ParseDecimal(s, currency string) (int64, error) {
	exp, ok := Exponent(currency)
	if !ok {
		return 0, errors.New("unsupported currency")
	}
	s = strings.NewReplacer(",", "", " ", "", " ", "").Replace(strings.TrimSpace(s))
	whole, frac, _ := strings.Cut(s, ".")
	if whole == "" && frac == "" {
		return 0, errors.New("empty amount")
	}
	if whole == "" {
		whole = "0"
	}
	if !allDigits(whole) || !allDigits(frac) {
		return 0, errors.New("invalid amount")
	}
	if len(frac) > exp {
		if strings.Trim(frac[exp:], "0") != "" {
			return 0, errors.New("too many decimal places")
		}
		frac = frac[:exp]
	}
	frac += strings.Repeat("0", exp-len(frac))

	var v int64
	for _, r := range whole + frac {
		d := int64(r - '0')
		if v > (1<<63-1-d)/10 {
			return 0, errors.New("amount too large")
		}
		v = v*10 + d
	}
	return v, nil
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
