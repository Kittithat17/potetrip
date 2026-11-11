package expense

import (
	"errors"
	"math/big"
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
