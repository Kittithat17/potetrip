package expense

import (
	"math"
	"slices"
	"testing"
)

func TestSplitEqual(t *testing.T) {
	cases := []struct {
		total int64
		n     int
		want  []int64
	}{
		{10000, 3, []int64{3334, 3333, 3333}},
		{10001, 3, []int64{3334, 3334, 3333}},
		{9999, 3, []int64{3333, 3333, 3333}},
		{1, 3, []int64{1, 0, 0}},
		{500, 1, []int64{500}},
	}
	for _, tc := range cases {
		got, err := SplitEqual(tc.total, tc.n)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("SplitEqual(%d,%d) = %v want %v", tc.total, tc.n, got, tc.want)
		}
	}
	if _, err := SplitEqual(100, 0); err == nil {
		t.Error("want error for zero people")
	}
}

func TestAllocateProportional(t *testing.T) {
	// 1000 JPY split 600/400 converted into 6789 satang.
	got, err := Allocate(6789, []int64{600, 400})
	if err != nil {
		t.Fatal(err)
	}
	if got[0]+got[1] != 6789 || got[0] != 4074 || got[1] != 2715 {
		t.Fatalf("got %v", got)
	}

	// Zero weights never receive remainder.
	got, _ = Allocate(10, []int64{0, 1, 1, 1})
	if !slices.Equal(got, []int64{0, 4, 3, 3}) {
		t.Fatalf("got %v", got)
	}
}

func TestAllocateNoOverflow(t *testing.T) {
	big := int64(math.MaxInt64 / 2)
	got, err := Allocate(big, []int64{big, big})
	if err != nil {
		t.Fatal(err)
	}
	if got[0]+got[1] != big {
		t.Fatalf("sum mismatch: %v", got)
	}
}

func TestAllocateErrors(t *testing.T) {
	for _, w := range [][]int64{{}, {0, 0}, {1, -1}} {
		if _, err := Allocate(100, w); err == nil {
			t.Errorf("weights %v: want error", w)
		}
	}
	if _, err := Allocate(-1, []int64{1}); err == nil {
		t.Error("negative total: want error")
	}
}
