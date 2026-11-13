package money

import "testing"

func TestParseDecimal(t *testing.T) {
	ok := []struct {
		in, cur string
		want    int64
	}{
		{"1250.00", "THB", 125000},
		{"1,250.5", "THB", 125050},
		{"0.01", "THB", 1},
		{".5", "THB", 50},
		{"100", "THB", 10000},
		{"3000", "JPY", 3000},
		{"3000.00", "JPY", 3000},
		{" 12 ", "USD", 1200},
	}
	for _, tc := range ok {
		got, err := ParseDecimal(tc.in, tc.cur)
		if err != nil || got != tc.want {
			t.Errorf("ParseDecimal(%q,%s) = %d,%v want %d", tc.in, tc.cur, got, err, tc.want)
		}
	}
	bad := [][2]string{{"", "THB"}, {"-5", "THB"}, {"1.234", "THB"}, {"10.5", "JPY"}, {"abc", "THB"}, {"1", "XXX"}, {"99999999999999999999", "THB"}}
	for _, b := range bad {
		if _, err := ParseDecimal(b[0], b[1]); err == nil {
			t.Errorf("ParseDecimal(%q,%s): want error", b[0], b[1])
		}
	}
}
