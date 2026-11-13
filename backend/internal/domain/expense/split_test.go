package expense

import (
	"math"
	"slices"
	"testing"

	"github.com/google/uuid"
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

func TestSplitItems(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	order := map[uuid.UUID]int{a: 0, b: 1, c: 2}

	// Bill ฿1,000 (items ฿900 + ฿100 service). Ramen ฿600 for A only, gyoza
	// ฿300 shared by all three; service is spread by price.
	items := []Item{
		{Name: "ramen", Amount: 60000, MemberIDs: []uuid.UUID{a}},
		{Name: "gyoza", Amount: 30000, MemberIDs: []uuid.UUID{c, a, b}},
	}
	got, err := SplitItems(100000, items, order)
	if err != nil {
		t.Fatal(err)
	}
	// ramen gets 66667 (first part absorbs the remainder), gyoza 33333 -> 11111 each.
	want := []Share{{a, 66667 + 11111}, {b, 11111}, {c, 11111}}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	var sum int64
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("share %d = %+v, want %+v", i, got[i], want[i])
		}
		sum += got[i].AmountBase
	}
	if sum != 100000 {
		t.Errorf("shares sum to %d", sum)
	}
}

func TestSplitItemsAlwaysExact(t *testing.T) {
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	order := map[uuid.UUID]int{}
	for i, id := range ids {
		order[id] = i
	}
	for total := int64(1); total < 400; total += 7 {
		items := []Item{
			{Amount: 333, MemberIDs: ids[:3]},
			{Amount: 101, MemberIDs: ids[1:]},
			{Amount: 7, MemberIDs: ids[2:3]},
		}
		got, err := SplitItems(total, items, order)
		if err != nil {
			t.Fatal(err)
		}
		var sum int64
		for _, s := range got {
			sum += s.AmountBase
		}
		if sum != total {
			t.Fatalf("total %d: shares sum to %d", total, sum)
		}
	}
}
