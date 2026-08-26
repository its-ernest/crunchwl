// Package generator expands a compiled pattern into every matching word and
// writes them to a buffered writer. It enumerates multi-wildcard lengths
// recursively, then emits each fixed-length template with an odometer over the
// charset for the variable positions.
package generator

import (
	"bufio"

	"github.com/its-ernest/crunchwl/pkg/pattern"
)

// Config controls a single generation run.
type Config struct {
	Pattern *pattern.CompiledPattern
	Charset []byte
	// MinWildcard / MaxWildcard bound the length chosen for each '%' segment.
	MinWildcard int
	MaxWildcard int
	// FirstCharRange optionally restricts the first variable slot to a subset of
	// the charset. The engine uses it to partition work across cores; when nil
	// the full charset is used.
	FirstCharRange []byte
}

// Generate writes every word described by cfg to writer.
func Generate(cfg Config, writer *bufio.Writer) error {
	var multiLengths []int
	if cfg.Pattern.MultiCount > 0 {
		multiLengths = make([]int, cfg.Pattern.MultiCount)
	}

	return expandMulti(0, multiLengths, cfg, writer)
}

// expandMulti recursively assigns a concrete length (from MinWildcard to
// MaxWildcard) to each '%' segment, then delegates to runFixedPattern once every
// multi length is fixed.
func expandMulti(depth int, multiLengths []int, cfg Config, writer *bufio.Writer) error {
	if depth == cfg.Pattern.MultiCount {
		return runFixedPattern(multiLengths, cfg, writer)
	}

	for l := cfg.MinWildcard; l <= cfg.MaxWildcard; l++ {
		multiLengths[depth] = l
		if err := expandMulti(depth+1, multiLengths, cfg, writer); err != nil {
			return err
		}
	}
	return nil
}

// runFixedPattern emits every combination for one concrete assignment of the
// multi-wildcard lengths. It reuses a single buffer across lines to avoid
// per-word allocations.
func runFixedPattern(multiLengths []int, cfg Config, writer *bufio.Writer) error {
	var variablePositions int
	multiIdx := 0

	for _, seg := range cfg.Pattern.Segments {
		switch seg.Type {
		case pattern.SegmentSingle:
			variablePositions++
		case pattern.SegmentMulti:
			variablePositions += multiLengths[multiIdx]
			multiIdx++
		}
	}

	if variablePositions == 0 {
		// Pure literal template
		for _, seg := range cfg.Pattern.Segments {
			if _, err := writer.Write(seg.Value); err != nil {
				return err
			}
		}
		if err := writer.WriteByte('\n'); err != nil {
			return err
		}
		return nil
	}

	// Calculate exact total line length (literals + single wildcards + multi wildcards + '\n')
	totalLen := 1 // newline byte
	mIdx := 0
	for _, seg := range cfg.Pattern.Segments {
		switch seg.Type {
		case pattern.SegmentLiteral:
			totalLen += len(seg.Value)
		case pattern.SegmentSingle:
			totalLen++
		case pattern.SegmentMulti:
			totalLen += multiLengths[mIdx]
			mIdx++
		}
	}

	baseLen := len(cfg.Charset)
	indices := make([]int, variablePositions)
	buf := make([]byte, 0, totalLen)

	firstSlotRange := cfg.Charset
	if len(cfg.FirstCharRange) > 0 {
		firstSlotRange = cfg.FirstCharRange
	}

	for fIdx := 0; fIdx < len(firstSlotRange); fIdx++ {
		indices[0] = findIndex(cfg.Charset, firstSlotRange[fIdx])
		for i := 1; i < variablePositions; i++ {
			indices[i] = 0
		}

		for {
			buf = buf[:0]
			varCursor := 0
			mIdx = 0

			for _, seg := range cfg.Pattern.Segments {
				switch seg.Type {
				case pattern.SegmentLiteral:
					buf = append(buf, seg.Value...)
				case pattern.SegmentSingle:
					buf = append(buf, cfg.Charset[indices[varCursor]])
					varCursor++
				case pattern.SegmentMulti:
					count := multiLengths[mIdx]
					mIdx++
					for c := 0; c < count; c++ {
						buf = append(buf, cfg.Charset[indices[varCursor]])
						varCursor++
					}
				}
			}
			buf = append(buf, '\n')
			if _, err := writer.Write(buf); err != nil {
				return err
			}

			if variablePositions == 1 {
				break
			}

			pos := variablePositions - 1
			for pos > 0 {
				indices[pos]++
				if indices[pos] < baseLen {
					break
				}
				indices[pos] = 0
				pos--
			}
			if pos == 0 {
				break
			}
		}
	}
	return nil
}

// findIndex returns the position of target within slice, or 0 if absent.
func findIndex(slice []byte, target byte) int {
	for i, b := range slice {
		if b == target {
			return i
		}
	}
	return 0
}
