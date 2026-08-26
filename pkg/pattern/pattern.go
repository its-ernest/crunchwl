// Package pattern compiles SQL-style wordlist patterns into a sequence of
// segments that the generator can expand.
//
// Supported tokens:
//   - literal text  -> emitted verbatim
//   - '_' (single)  -> exactly one character from the charset
//   - '%' (multi)   -> a run of characters whose length is chosen at expansion
//     time from the [min, max] wildcard range
package pattern

// SegmentType distinguishes the three kinds of pattern segment.
type SegmentType int

const (
	// SegmentLiteral is fixed text emitted verbatim into every word.
	SegmentLiteral SegmentType = iota
	// SegmentSingle is the '_' token: a single charset character.
	SegmentSingle
	// SegmentMulti is the '%' token: a variable-length run of charset characters.
	SegmentMulti
)

// Segment is a single piece of a compiled pattern.
type Segment struct {
	Type SegmentType
	// Value holds the literal bytes when Type == SegmentLiteral.
	Value []byte
}

// CompiledPattern is a parsed pattern ready for expansion.
type CompiledPattern struct {
	Segments   []Segment
	MultiCount int
}

// Parse turns a pattern string into a CompiledPattern. Literal runs are kept
// contiguous; each '_' and '%' becomes its own segment. Parse never fails for
// this pattern grammar.
func Parse(patternStr string) (*CompiledPattern, error) {
	var segments []Segment
	raw := []byte(patternStr)
	multiCount := 0

	var currentLiteral []byte

	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		switch ch {
		case '_':
			if len(currentLiteral) > 0 {
				segments = append(segments, Segment{Type: SegmentLiteral, Value: currentLiteral})
				currentLiteral = nil
			}
			segments = append(segments, Segment{Type: SegmentSingle})
		case '%':
			if len(currentLiteral) > 0 {
				segments = append(segments, Segment{Type: SegmentLiteral, Value: currentLiteral})
				currentLiteral = nil
			}
			segments = append(segments, Segment{Type: SegmentMulti})
			multiCount++
		default:
			currentLiteral = append(currentLiteral, ch)
		}
	}

	if len(currentLiteral) > 0 {
		segments = append(segments, Segment{Type: SegmentLiteral, Value: currentLiteral})
	}

	return &CompiledPattern{
		Segments:   segments,
		MultiCount: multiCount,
	}, nil
}
