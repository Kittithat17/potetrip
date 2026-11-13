package money

import (
	"errors"
	"math/big"
	"strings"
)

// RateDecimals is the precision exchange rates are stored with (numeric(18,8)).
const RateDecimals = 8

const maxRateIntDigits = 18 - RateDecimals

var (
	ErrInvalidRate = errors.New("invalid exchange rate")
	ErrTooLarge    = errors.New("amount too large")
)

// ParseRate parses a user-entered rate such as "0.2345" or "1,234.5".
// It must be positive and fit numeric(18,8): at most 8 decimals.
func ParseRate(s string) (*big.Rat, error) {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	whole, frac, _ := strings.Cut(s, ".")
	if whole == "" {
		whole = "0"
	}
	if !allDigits(whole) || !allDigits(frac) || len(frac) > RateDecimals {
		return nil, ErrInvalidRate
	}
	r, ok := new(big.Rat).SetString(whole + "." + frac + "0")
	if !ok || r.Sign() <= 0 || len(strings.TrimLeft(whole, "0")) > maxRateIntDigits {
		return nil, ErrInvalidRate
	}
	return r, nil
}

// NormalizeRate rounds a provider rate (any precision) to 8 decimals.
func NormalizeRate(s string) (*big.Rat, error) {
	r, ok := new(big.Rat).SetString(strings.TrimSpace(s))
	if !ok || r.Sign() <= 0 {
		return nil, ErrInvalidRate
	}
	r, ok = new(big.Rat).SetString(r.FloatString(RateDecimals))
	if !ok || r.Sign() <= 0 {
		return nil, ErrInvalidRate
	}
	return r, nil
}

// RateString formats a rate with 8 decimals, trailing zeros removed.
func RateString(r *big.Rat) string {
	s := r.FloatString(RateDecimals)
	if strings.Contains(s, ".") {
		s = strings.TrimSuffix(strings.TrimRight(s, "0"), ".")
	}
	return s
}

// Convert turns amount (minor units of from) into minor units of to at rate
// (units of to per 1 unit of from), rounding half up. Both currencies must be
// supported and amount must not be negative.
func Convert(amount int64, from, to string, rate *big.Rat) (int64, error) {
	ef, ok1 := Exponent(from)
	et, ok2 := Exponent(to)
	if !ok1 || !ok2 {
		return 0, errors.New("unsupported currency")
	}
	if amount < 0 || rate == nil || rate.Sign() <= 0 {
		return 0, errors.New("invalid conversion")
	}
	v := new(big.Rat).Mul(new(big.Rat).SetInt64(amount), rate)
	scale := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(abs(et-ef))), nil))
	if et >= ef {
		v.Mul(v, scale)
	} else {
		v.Quo(v, scale)
	}
	// floor((2*num + den) / (2*den)) rounds a non-negative value half up.
	num := new(big.Int).Add(new(big.Int).Lsh(v.Num(), 1), v.Denom())
	q := new(big.Int).Quo(num, new(big.Int).Lsh(v.Denom(), 1))
	if !q.IsInt64() {
		return 0, ErrTooLarge
	}
	return q.Int64(), nil
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
