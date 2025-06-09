package primitives

import "strings"

// ConcreteLine represents a single possible line in a puzzle.
type ConcreteLine struct {
	Line  []rune
	Words []string
}

// Length returns the length of the line.
func (l *ConcreteLine) Length() int {
	return len(l.Line)
}

func (l *ConcreteLine) String() string {
	return strings.ToUpper(string(l.Line))
}
