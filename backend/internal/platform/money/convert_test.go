package money

import (
	"math/big"
	"testing"
)

func rat(s string) *big.Rat {
	r, _ := new(big.Rat).SetString(s)
	return r
}

func TestConvert(t *testing.T) {
	cases := []struct {
		amount   int64
		from, to string
		rate     string
		want     int64
	}{
		{1500, "JPY", "THB", "0.2345", 35175},               // ¥1,500 -> ฿351.75
		{1, "JPY", "THB", "0.23456", 23},                    // 23.456 satang rounds down
		{3, "JPY", "THB", "0.235", 71},                      // 70.5 rounds half up
		{12550, "THB", "JPY", "4.2645", 535},                // ฿125.50 -> ¥535.19…
		{10000, "USD", "THB", "33.5", 335000},               // same exponent
		{100000, "VND", "THB", "0.00136", 13600},            // ₫100,000 -> ฿136.00
		{12345, "THB", "THB", "1", 12345},                   // identity
		{0, "EUR", "THB", "38.1", 0},                        // zero stays zero
		{999999999, "KRW", "THB", "0.02512345", 2512344997}, // big
	}
	for _, c := range cases {
		got, err := Convert(c.amount, c.from, c.to, rat(c.rate))
		if err != nil || got != c.want {
			t.Errorf("Convert(%d %s->%s @%s) = %d, %v; want %d", c.amount, c.from, c.to, c.rate, got, err, c.want)
		}
	}
	if _, err := Convert(1<<62, "JPY", "VND", rat("1000")); err != ErrTooLarge {
		t.Errorf("overflow: got %v", err)
	}
	if _, err := Convert(1, "XXX", "THB", rat("1")); err == nil {
		t.Error("unknown currency accepted")
	}
}

func TestParseRate(t *testing.T) {
	ok := map[string]string{"0.2345": "0.2345", "1,234.5": "1234.5", ".5": "0.5", "25": "25", "0.00000001": "0.00000001"}
	for in, want := range ok {
		r, err := ParseRate(in)
		if err != nil || RateString(r) != want {
			t.Errorf("ParseRate(%q) = %v, %v; want %s", in, r, err, want)
		}
	}
	for _, in := range []string{"", "0", "-1", "abc", "1.123456789", "1e5", "12345678901"} {
		if _, err := ParseRate(in); err == nil {
			t.Errorf("ParseRate(%q) accepted", in)
		}
	}
}

func TestNormalizeRate(t *testing.T) {
	r, err := NormalizeRate("0.002064512")
	if err != nil || RateString(r) != "0.00206451" {
		t.Fatalf("got %v %v", r, err)
	}
	if _, err := NormalizeRate("0.000000001"); err == nil {
		t.Error("rate rounding to zero accepted")
	}
}
