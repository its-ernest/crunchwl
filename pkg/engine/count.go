package engine

import (
	"math/big"

	"github.com/its-ernest/crunchwl/pkg/pattern"
)

// Count returns the exact number of words a pattern expands to.
//
// Each literal contributes a factor of 1, each single wildcard "_" contributes
// baseLen possibilities, and each multi wildcard "%" can take any length in
// [minW, maxW], contributing sum(baseLen^l) for l in that range. The factors
// multiply across all segments.
func Count(pat *pattern.CompiledPattern, charset []byte, minW, maxW int) *big.Int {
	baseLen := len(charset)
	if baseLen == 0 {
		return big.NewInt(0)
	}

	numSingles := 0
	for _, seg := range pat.Segments {
		if seg.Type == pattern.SegmentSingle {
			numSingles++
		}
	}

	k := pat.MultiCount
	count := big.NewInt(1)

	bl := big.NewInt(int64(baseLen))
	if numSingles > 0 {
		tmp := new(big.Int).Exp(bl, big.NewInt(int64(numSingles)), nil)
		count.Mul(count, tmp)
	}

	if k > 0 {
		if minW > maxW {
			return big.NewInt(0)
		}
		sum := big.NewInt(0)
		term := new(big.Int)
		for l := minW; l <= maxW; l++ {
			term.Exp(bl, big.NewInt(int64(l)), nil)
			sum.Add(sum, term)
		}
		sk := new(big.Int).Exp(sum, big.NewInt(int64(k)), nil)
		count.Mul(count, sk)
	}

	return count
}
