package expense

import (
	"errors"
	"math/big"
	"slices"

	"github.com/google/uuid"
)

// Allocate splits total into len(weights) integer parts proportional to
// weights. Parts are floored; the leftover minor units are handed out one
// at a time to parts in order (skipping zero-weight parts), so the result
// always sums exactly to total.
func Allocate(total int64, weights []int64) ([]int64, error) {
	if total < 0 {
		return nil, errors.New("total must not be negative")
	}
	var sum int64
	for _, w := range weights {
		if w < 0 {
			return nil, errors.New("weights must not be negative")
		}
		sum += w
	}
	if sum <= 0 {
		return nil, errors.New("weights must sum to more than zero")
	}

	out := make([]int64, len(weights))
	bigTotal, bigSum := big.NewInt(total), big.NewInt(sum)
	var allocated int64
	for i, w := range weights {
		// total*w can overflow int64 for large amounts, so use big.Int.
		q := new(big.Int).Mul(bigTotal, big.NewInt(w))
		q.Quo(q, bigSum)
		out[i] = q.Int64()
		allocated += out[i]
	}
	for rem := total - allocated; rem > 0; {
		for i, w := range weights {
			if rem == 0 {
				break
			}
			if w > 0 {
				out[i]++
				rem--
			}
		}
	}
	return out, nil
}

// SplitEqual splits total across n people; earlier people absorb the remainder.
func SplitEqual(total int64, n int) ([]int64, error) {
	weights := make([]int64, n)
	for i := range weights {
		weights[i] = 1
	}
	return Allocate(total, weights)
}

// SplitItems shares totalBase across receipt lines. The total is first
// allocated to lines in proportion to their prices, so tax, service charge
// or a discount on the bill spreads evenly; each line's part is then split
// equally among the people who had it (in trip order). Shares come back in
// trip order and always sum exactly to totalBase.
func SplitItems(totalBase int64, items []Item, order map[uuid.UUID]int) ([]Share, error) {
	weights := make([]int64, len(items))
	for i, it := range items {
		weights[i] = it.Amount
	}
	perItem, err := Allocate(totalBase, weights)
	if err != nil {
		return nil, err
	}
	owed := map[uuid.UUID]int64{}
	for i, it := range items {
		parts, err := SplitEqual(perItem[i], len(it.MemberIDs))
		if err != nil {
			return nil, err
		}
		for j, id := range it.MemberIDs {
			owed[id] += parts[j]
		}
	}
	ids := make([]uuid.UUID, 0, len(owed))
	for id, v := range owed {
		if v > 0 {
			ids = append(ids, id)
		}
	}
	slices.SortFunc(ids, func(a, b uuid.UUID) int { return order[a] - order[b] })
	out := make([]Share, len(ids))
	for i, id := range ids {
		out[i] = Share{MemberID: id, AmountBase: owed[id]}
	}
	return out, nil
}
