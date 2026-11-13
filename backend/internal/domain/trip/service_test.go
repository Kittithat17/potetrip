package trip

import (
	"strings"
	"testing"
)

func TestNewInviteCode(t *testing.T) {
	seen := map[string]bool{}
	for range 200 {
		c := newInviteCode()
		if len(c) != inviteLen {
			t.Fatalf("len %d", len(c))
		}
		for _, r := range c {
			if !strings.ContainsRune(inviteAlphabet, r) {
				t.Fatalf("bad rune %q in %s", r, c)
			}
		}
		seen[c] = true
	}
	if len(seen) < 199 {
		t.Fatalf("too many collisions: %d unique", len(seen))
	}
}

func TestNormalizePromptPay(t *testing.T) {
	ptr := func(s string) *string { return &s }
	ok := map[string]string{
		"081-234-5678":      "0812345678",
		"1 2345 67890 12 3": "1234567890123",
	}
	for in, want := range ok {
		got, err := normalizePromptPay(ptr(in))
		if err != nil || got == nil || *got != want {
			t.Errorf("%q -> %v, %v", in, got, err)
		}
	}
	if got, err := normalizePromptPay(ptr("  ")); err != nil || got != nil {
		t.Errorf("blank -> %v, %v", got, err)
	}
	for _, bad := range []string{"12345", "08x2345678"} {
		if _, err := normalizePromptPay(ptr(bad)); err == nil {
			t.Errorf("%q: want error", bad)
		}
	}
}
