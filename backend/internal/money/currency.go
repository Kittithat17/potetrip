// Package money holds currency metadata. Amounts are always integers in a
// currency's minor unit (ISO 4217 exponent), never floats.
package money

import "strings"

// exponents lists supported ISO 4217 currencies and their minor-unit digits.
var exponents = map[string]int{
	"THB": 2, "USD": 2, "EUR": 2, "GBP": 2, "JPY": 0, "KRW": 0, "CNY": 2,
	"HKD": 2, "TWD": 2, "SGD": 2, "MYR": 2, "IDR": 2, "PHP": 2, "VND": 0,
	"LAK": 2, "KHR": 2, "MMK": 2, "INR": 2, "AUD": 2, "NZD": 2, "CAD": 2,
	"CHF": 2, "SEK": 2, "NOK": 2, "DKK": 2, "AED": 2, "TRY": 2, "MOP": 2,
	"CZK": 2, "HUF": 2, "PLN": 2, "ISK": 0,
}

// Normalize upper-cases and trims a currency code.
func Normalize(code string) string { return strings.ToUpper(strings.TrimSpace(code)) }

// Supported reports whether code is a known currency.
func Supported(code string) bool {
	_, ok := exponents[code]
	return ok
}

// Exponent returns the number of minor-unit digits, and false if unknown.
func Exponent(code string) (int, bool) {
	e, ok := exponents[code]
	return e, ok
}
