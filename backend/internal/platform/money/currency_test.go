package money

import "testing"

func TestExponent(t *testing.T) {
	for code, want := range map[string]int{"THB": 2, "JPY": 0, "KRW": 0, "USD": 2} {
		got, ok := Exponent(code)
		if !ok || got != want {
			t.Errorf("Exponent(%s) = %d,%v want %d", code, got, ok, want)
		}
	}
	if Supported("XXX") {
		t.Error("XXX should not be supported")
	}
	if Normalize(" thb ") != "THB" {
		t.Error("normalize")
	}
}
