package primitives

import (
	"fmt"
	"iter"
	"slices"
	"strings"
)

const kBlocked = '`'

// ChoiceStep represents a single choice in deciding what the a given line in a puzzle should be,
// dividing the set of possible lines into two sets that can be iterated over.
type ChoiceStep struct {
	Choice    PossibleLines
	Remaining PossibleLines
}

type sealed = any

// PossibleLines represents a set of possible lines in our puzzle. A 'Line' is a string of values
// in the puzzle's boxes representing an entire line from start to end. It can include characters
// and blocked cells.
type PossibleLines interface {
	sealed // This interface is not meant to be implemented by anything other than the types below.

	// NumLetters returns the length of a line.
	NumLetters() int

	// MaxPossibilities returns an upper bound on the number of possible lines.
	//
	// This can be lower since some lines might include repeated words, etc.
	MaxPossibilities() int64

	// CharsAt adds the characters that can appear at a given index to the given set.
	CharsAt(accumulate *CharSet, index int)

	// DefinitelyBlockedAt returns true if the line is definitely blocked at the given index.
	DefinitelyBlockedAt(index int) bool

	// DefiniteWords returns a list of words that are definitely present in the line.
	//
	// i.e., these words are guaranteed to appear in the line.
	DefiniteWords() []string

	// FilterAny filters the set of possible lines to only include those that contain the given
	// character at the given index.
	FilterAny(constraint *CharSet, index int) PossibleLines

	// Filter filters the set of possible lines to only include those that contain the given
	// character at the given index.
	Filter(constraint rune, index int) PossibleLines

	// RemoveWordOptions strips the possible lines to no longer include a given set of word.
	RemoveWordOptions(word []string) PossibleLines

	// Iterate returns a sequence of all possible lines.
	Iterate() iter.Seq[ConcreteLine]

	// FirstOrNull returns the first possible line, or nil if there are no possible lines.
	FirstOrNull() *ConcreteLine

	// MakeChoice returns a choice step that divides the set of possible lines into two sets that
	// can be iterated over.
	//
	// Ideally, MakeChoice will return two groups that are roughly equal in size.
	MakeChoice() ChoiceStep

	String() string
}

// Impossible represents an empty set of possible lines.
type Impossible struct {
	numLetters int
}

func (i *Impossible) NumLetters() int {
	return i.numLetters
}

func (i *Impossible) MaxPossibilities() int64 {
	return 0
}

func (i *Impossible) CharsAt(accumulate *CharSet, index int) {
}

func (i *Impossible) DefinitelyBlockedAt(index int) bool {
	return false
}

func (i *Impossible) DefiniteWords() []string {
	return nil
}

func (i *Impossible) FilterAny(constraint *CharSet, index int) PossibleLines {
	return i
}

func (i *Impossible) Filter(constraint rune, index int) PossibleLines {
	return i
}

func (i *Impossible) RemoveWordOptions(words []string) PossibleLines {
	return i
}

func (i *Impossible) Iterate() iter.Seq[ConcreteLine] {
	return func(yield func(ConcreteLine) bool) {}
}

func (i *Impossible) FirstOrNull() *ConcreteLine {
	return nil
}

func (i *Impossible) MakeChoice() ChoiceStep {
	panic("Cannot call MakeChoice on Impossible")
}

func (i *Impossible) String() string {
	return fmt.Sprintf("Impossible(%d)", i.numLetters)
}

var ic = make([]Impossible, 25)

func MakeImpossible(numLetters int) *Impossible {
	if ic[numLetters] == (Impossible{}) {
		ic[numLetters] = Impossible{numLetters: numLetters}
	}
	return &ic[numLetters]
}

// Words represents a set of possible lines that are exactly filled with any one of the given words.
//
// Each word in 'Words' is exactly the same length, fully occupying the line.
type Words struct {
	allWords   []string // All words, starting with preferred, then obscure
	obscureIdx int      // Index of first obscure word, if 0, all words are obscure, if len(allWords), all words are preferred
	// letterMasks caches, for each index, the bitmask of allowed runes across all words.
	// It accelerates CharsAt and lets FilterAny early-return.
	letterMasks []CharSet
}

func MakeWordsFromPreferredAndObscure(preferred, obscure []string, numLetters int) PossibleLines {
	if len(preferred) == 0 && len(obscure) == 0 {
		return MakeImpossible(numLetters)
	}
	if len(preferred) == 1 && len(obscure) == 0 {
		return MakeDefinite(ConcreteLine{Line: []rune(preferred[0]), Words: []string{preferred[0]}})
	}
	if len(preferred) == 0 && len(obscure) == 1 {
		return MakeDefinite(ConcreteLine{Line: []rune(obscure[0]), Words: []string{obscure[0]}})
	}
	// Lazily allocate letterMasks on first use to avoid upfront cost when not needed.
	return &Words{allWords: append(preferred, obscure...), obscureIdx: len(preferred)}
}

func MakeWords(allWords []string, obscureIdx int, numLetters int) PossibleLines {
	if len(allWords) == 0 {
		return MakeImpossible(numLetters)
	}
	if len(allWords) == 1 {
		return MakeDefinite(ConcreteLine{Line: []rune(allWords[0]), Words: []string{allWords[0]}})
	}
	// Lazily allocate letterMasks on first use to avoid upfront cost when not needed.
	return &Words{allWords: allWords, obscureIdx: obscureIdx}
}

func (w *Words) NumLetters() int {
	return len(w.allWords[0])
}

func (w *Words) MaxPossibilities() int64 {
	return int64(len(w.allWords))
}

func (w *Words) CharsAt(accumulate *CharSet, index int) {
	if accumulate.IsFull() || (!accumulate.Contains(kBlocked) && (accumulate.Count()+1) == accumulate.Capacity()) {
		return
	}
	// Build masks lazily.
	if w.letterMasks == nil {
		w.letterMasks = make([]CharSet, w.NumLetters())
	}
	if w.letterMasks[index].bits == 0 {
		w.letterMasks[index] = CharSet{}
		for _, word := range w.allWords {
			r := rune(word[index])
			w.letterMasks[index].Add(r)
		}
	}
	accumulate.AddAll(&w.letterMasks[index])
}

func (w *Words) DefinitelyBlockedAt(index int) bool {
	return false
}

func (w *Words) DefiniteWords() []string {
	if len(w.allWords) == 1 {
		return []string{w.allWords[0]}
	}
	return nil
}

func (w *Words) FilterAny(constraint *CharSet, index int) PossibleLines {
	if constraint.IsFull() || (!constraint.Contains(kBlocked) && (constraint.Count()+1) == constraint.Capacity()) {
		return w
	}

	// If we have a mask and it is entirely contained by the constraint, nothing to filter.
	if w.letterMasks != nil && w.letterMasks[index].bits != 0 {
		mask := w.letterMasks[index]
		if constraint.ContainsAll(&mask) {
			return w
		}
	}

	// Lazy: First check if any of the words in the list don't match the filter.
	// Otherwise we don't need to copy the lists
	if !slices.ContainsFunc(w.allWords, func(word string) bool {
		return !constraint.Contains(rune(word[index]))
	}) {
		return w
	}

	var filtered []string
	var newNumPreferred int
	for idx, word := range w.allWords {
		if constraint.Contains(rune(word[index])) {
			if idx < w.obscureIdx {
				newNumPreferred++
			}
			if filtered == nil {
				filtered = make([]string, 0, len(w.allWords)-idx)
			}
			filtered = append(filtered, word)
		}
	}

	return MakeWords(filtered, newNumPreferred, w.NumLetters())
}

func (w *Words) Filter(constraint rune, index int) PossibleLines {
	if constraint == kBlocked {
		return MakeImpossible(w.NumLetters())
	}

	// Optimization: Check if all words already match the constraint.
	// If so, return w.
	if w.MaxPossibilities() > 0 {
		anyMismatch := slices.ContainsFunc(w.allWords, func(word string) bool {
			return rune(word[index]) != constraint
		})
		if !anyMismatch {
			return w
		}
	}

	var filtered []string
	newNumPreferred := 0
	for idx, word := range w.allWords {
		if rune(word[index]) == constraint {
			if idx < w.obscureIdx {
				newNumPreferred++
			}
			// Lazy: allocate filtered list with capacity of allWords-idx only if we
			// get here.
			if filtered == nil {
				filtered = make([]string, 0, len(w.allWords)-idx)
			}
			filtered = append(filtered, word)
		}
	}

	return MakeWords(filtered, newNumPreferred, w.NumLetters())
}

func (w *Words) RemoveWordOptions(words []string) PossibleLines {
	// Figure out if any (or both) lists need filtering. For any that doesn't,
	// we don't need to allocate a new list.
	needsFiltering := slices.ContainsFunc(words, func(word string) bool {
		if len(word) != w.NumLetters() {
			return false
		}
		return slices.Contains(w.allWords, word)
	})

	if !needsFiltering {
		return w
	}

	var fp []string
	fPreferred := 0

	fp = make([]string, 0, len(w.allWords)-1)
	for idx, p := range w.allWords {
		if !slices.Contains(words, p) {
			fp = append(fp, p)
			if idx < w.obscureIdx {
				fPreferred++
			}
		}
	}

	return MakeWords(fp, fPreferred, w.NumLetters())
}

func (w *Words) FirstOrNull() *ConcreteLine {
	if len(w.allWords) == 0 {
		return nil
	}
	return &ConcreteLine{Line: []rune(w.allWords[0]), Words: []string{w.allWords[0]}}
}

func (w *Words) Iterate() iter.Seq[ConcreteLine] {
	return func(yield func(ConcreteLine) bool) {
		for _, word := range w.allWords {
			if !yield(ConcreteLine{Line: []rune(word), Words: []string{word}}) {
				return
			}
		}
	}
}

func (w *Words) MakeChoice() ChoiceStep {
	if w.MaxPossibilities() <= 1 {
		panic("Cannot call MakeChoice on entity with 1 or less options")
	}

	// Simply split allWords in half, and adjust obscureIdx accordingly.
	w1, w2 := w.allWords[:len(w.allWords)/2], w.allWords[len(w.allWords)/2:]
	var w1Idx, w2Idx int
	if w.obscureIdx < len(w1) {
		w1Idx = w.obscureIdx
		w2Idx = 0
	} else {
		w1Idx = len(w1)
		w2Idx = w.obscureIdx - len(w1)
	}

	return ChoiceStep{
		Choice:    MakeWords(w1, w1Idx, w.NumLetters()),
		Remaining: MakeWords(w2, w2Idx, w.NumLetters()),
	}
}

func arrayStr(arr []string) string {
	const maxPrint = 3

	if len(arr) == 0 {
		return "[]"
	}
	if len(arr) <= maxPrint {
		return fmt.Sprintf("[%s]", strings.Join(arr, ", "))
	}

	print, rest := arr[:maxPrint], arr[maxPrint:]
	return fmt.Sprintf("[%s, ...%d]", strings.Join(print, ", "), len(rest))
}

func (w *Words) String() string {
	return fmt.Sprintf("Words(%s, %s)", arrayStr(w.allWords[0:w.obscureIdx]), arrayStr(w.allWords[w.obscureIdx:]))
}

// BlockBefore represents a line that has a blocked cell at the beginning.
type BlockBefore struct {
	lines PossibleLines
}

func MakeBlockBefore(lines PossibleLines) PossibleLines {
	if isImpossible(lines) {
		return MakeImpossible(lines.NumLetters() + 1)
	}
	return &BlockBefore{lines: lines}
}

func (b *BlockBefore) NumLetters() int {
	return 1 + b.lines.NumLetters()
}

func (b *BlockBefore) MaxPossibilities() int64 {
	return b.lines.MaxPossibilities()
}

func (b *BlockBefore) CharsAt(accumulate *CharSet, index int) {
	if accumulate.IsFull() {
		return
	}
	if index == 0 {
		accumulate.Add(kBlocked)
	} else {
		b.lines.CharsAt(accumulate, index-1)
	}
}

func (b *BlockBefore) DefinitelyBlockedAt(index int) bool {
	if index == 0 {
		return true
	}
	return b.lines.DefinitelyBlockedAt(index - 1)
}

func (b *BlockBefore) DefiniteWords() []string {
	return b.lines.DefiniteWords()
}

func (b *BlockBefore) build(inner PossibleLines) PossibleLines {
	if isImpossible(inner) {
		return MakeImpossible(b.NumLetters())
	}
	if inner == b.lines {
		return b
	}
	return &BlockBefore{lines: inner}
}

func (b *BlockBefore) FilterAny(constraint *CharSet, index int) PossibleLines {
	if constraint.IsFull() {
		return b
	}

	if index == 0 {
		if constraint.Contains(kBlocked) {
			return b
		}
		return MakeImpossible(b.NumLetters())
	}
	return b.build(b.lines.FilterAny(constraint, index-1))
}

func (b *BlockBefore) Filter(constraint rune, index int) PossibleLines {
	if index == 0 {
		if constraint == kBlocked {
			return b
		}
		return MakeImpossible(b.NumLetters())
	}
	return b.build(b.lines.Filter(constraint, index-1))
}

func (b *BlockBefore) RemoveWordOptions(words []string) PossibleLines {
	return b.build(b.lines.RemoveWordOptions(words))
}

func (b *BlockBefore) FirstOrNull() *ConcreteLine {
	c := b.lines.FirstOrNull()
	if c == nil {
		return nil
	}
	return &ConcreteLine{Line: append([]rune{kBlocked}, c.Line...), Words: c.Words}
}

func (b *BlockBefore) MakeChoice() ChoiceStep {
	c := b.lines.MakeChoice()
	return ChoiceStep{
		Choice:    &BlockBefore{lines: c.Choice},
		Remaining: &BlockBefore{lines: c.Remaining},
	}
}

func (b *BlockBefore) Iterate() iter.Seq[ConcreteLine] {
	return func(yield func(ConcreteLine) bool) {
		for line := range b.lines.Iterate() {
			if !yield(ConcreteLine{Line: append([]rune{kBlocked}, line.Line...), Words: line.Words}) {
				return
			}
		}
	}
}

func (b *BlockBefore) String() string {
	return fmt.Sprintf("BlockBefore(%s)", b.lines.String())
}

// BlockAfter represents a line that has a blocked cell at the end.
type BlockAfter struct {
	lines PossibleLines
}

func MakeBlockAfter(lines PossibleLines) PossibleLines {
	if isImpossible(lines) {
		return MakeImpossible(lines.NumLetters() + 1)
	}
	return &BlockAfter{lines: lines}
}

func (b *BlockAfter) NumLetters() int {
	return 1 + b.lines.NumLetters()
}

func (b *BlockAfter) MaxPossibilities() int64 {
	return b.lines.MaxPossibilities()
}

func (b *BlockAfter) CharsAt(accumulate *CharSet, index int) {
	if accumulate.IsFull() {
		return
	}
	if index == b.lines.NumLetters() {
		accumulate.Add(kBlocked)
	} else {
		b.lines.CharsAt(accumulate, index)
	}
}

func (b *BlockAfter) DefinitelyBlockedAt(index int) bool {
	if index == b.lines.NumLetters() {
		return true
	}
	return b.lines.DefinitelyBlockedAt(index)
}

func (b *BlockAfter) DefiniteWords() []string {
	return b.lines.DefiniteWords()
}

func (b *BlockAfter) build(inner PossibleLines) PossibleLines {
	if isImpossible(inner) {
		return MakeImpossible(b.NumLetters())
	}
	if inner == b.lines {
		return b
	}
	return &BlockAfter{lines: inner}
}

func (b *BlockAfter) FilterAny(constraint *CharSet, index int) PossibleLines {
	if constraint.IsFull() {
		return b
	}

	if index == b.lines.NumLetters() {
		if constraint.Contains(kBlocked) {
			return b
		}
		return MakeImpossible(b.NumLetters())
	}
	return b.build(b.lines.FilterAny(constraint, index))
}

func (b *BlockAfter) Filter(constraint rune, index int) PossibleLines {
	if index == b.lines.NumLetters() {
		if constraint == kBlocked {
			return b
		}
		return MakeImpossible(b.NumLetters())
	}
	return b.build(b.lines.Filter(constraint, index))
}

func (b *BlockAfter) RemoveWordOptions(words []string) PossibleLines {
	return b.build(b.lines.RemoveWordOptions(words))
}

func (b *BlockAfter) FirstOrNull() *ConcreteLine {
	c := b.lines.FirstOrNull()
	if c == nil {
		return nil
	}
	return &ConcreteLine{Line: append(c.Line, kBlocked), Words: c.Words}
}

func (b *BlockAfter) Iterate() iter.Seq[ConcreteLine] {
	return func(yield func(ConcreteLine) bool) {
		for line := range b.lines.Iterate() {
			if !yield(ConcreteLine{Line: append(line.Line, kBlocked), Words: line.Words}) {
				return
			}
		}
	}
}

func (b *BlockAfter) MakeChoice() ChoiceStep {
	c := b.lines.MakeChoice()
	return ChoiceStep{
		Choice:    &BlockAfter{lines: c.Choice},
		Remaining: &BlockAfter{lines: c.Remaining},
	}
}

func (b *BlockAfter) String() string {
	return fmt.Sprintf("BlockAfter(%s)", b.lines.String())
}

// BlockBetween represents a line that has a blocked cell in the middle.
type BlockBetween struct {
	first  PossibleLines
	second PossibleLines
}

func MakeBlockBetween(first, second PossibleLines) PossibleLines {
	if isImpossible(first) || isImpossible(second) {
		return MakeImpossible(first.NumLetters() + second.NumLetters() + 1)
	}
	return &BlockBetween{first: first, second: second}
}

func (b *BlockBetween) NumLetters() int {
	return 1 + b.first.NumLetters() + b.second.NumLetters()
}

func (b *BlockBetween) MaxPossibilities() int64 {
	return b.first.MaxPossibilities() * b.second.MaxPossibilities()
}

func (b *BlockBetween) CharsAt(accumulate *CharSet, index int) {
	if accumulate.IsFull() {
		return
	}
	if index == b.first.NumLetters() {
		accumulate.Add(kBlocked)
	} else if index < b.first.NumLetters() {
		b.first.CharsAt(accumulate, index)
	} else {
		b.second.CharsAt(accumulate, index-b.first.NumLetters()-1)
	}
}

func (b *BlockBetween) DefinitelyBlockedAt(index int) bool {
	if index == b.first.NumLetters() {
		return true
	}
	if index < b.first.NumLetters() {
		return b.first.DefinitelyBlockedAt(index)
	}
	return b.second.DefinitelyBlockedAt(index - b.first.NumLetters() - 1)
}

func (b *BlockBetween) build(first, second PossibleLines) PossibleLines {
	if isImpossible(first) || isImpossible(second) {
		return MakeImpossible(b.NumLetters())
	}
	if first == b.first && second == b.second {
		return b
	}
	return &BlockBetween{first: first, second: second}
}

func (b *BlockBetween) DefiniteWords() []string {
	return append(b.first.DefiniteWords(), b.second.DefiniteWords()...)
}

func (b *BlockBetween) FilterAny(constraint *CharSet, index int) PossibleLines {
	if constraint.IsFull() {
		return b
	}

	if index == b.first.NumLetters() {
		if constraint.Contains(kBlocked) {
			return b
		}
		return MakeImpossible(b.NumLetters())
	}

	f := b.first
	s := b.second
	if index < f.NumLetters() {
		f = f.FilterAny(constraint, index)
	} else {
		s = s.FilterAny(constraint, index-f.NumLetters()-1)
	}

	return b.build(f, s)
}

func (b *BlockBetween) Filter(constraint rune, index int) PossibleLines {
	if index == b.first.NumLetters() {
		if constraint == kBlocked {
			return b
		}
		return MakeImpossible(b.NumLetters())
	}

	f := b.first
	s := b.second
	if index < f.NumLetters() {
		f = f.Filter(constraint, index)
	} else {
		s = s.Filter(constraint, index-f.NumLetters()-1)
	}

	return b.build(f, s)
}

func (b *BlockBetween) RemoveWordOptions(words []string) PossibleLines {
	return b.build(b.first.RemoveWordOptions(words), b.second.RemoveWordOptions(words))
}

func (b *BlockBetween) FirstOrNull() *ConcreteLine {
	f := b.first.FirstOrNull()
	s := b.second.FirstOrNull()
	if f == nil || s == nil {
		return nil
	}
	return &ConcreteLine{Line: append(append(f.Line, kBlocked), s.Line...), Words: append(f.Words, s.Words...)}
}

func (b *BlockBetween) Iterate() iter.Seq[ConcreteLine] {
	return func(yield func(ConcreteLine) bool) {
		for first := range b.first.Iterate() {
			for second := range b.second.Iterate() {
				if !yield(ConcreteLine{
					Line:  append(append(first.Line, kBlocked), second.Line...),
					Words: append(first.Words, second.Words...),
				}) {
					return
				}
			}
		}
	}
}

func (b *BlockBetween) MakeChoice() ChoiceStep {
	if b.first.MaxPossibilities() > b.second.MaxPossibilities() {
		firstChoice := b.first.MakeChoice()
		return ChoiceStep{
			Choice:    &BlockBetween{first: firstChoice.Choice, second: b.second},
			Remaining: &BlockBetween{first: firstChoice.Remaining, second: b.second},
		}
	}

	secondChoice := b.second.MakeChoice()
	return ChoiceStep{
		Choice:    &BlockBetween{first: b.first, second: secondChoice.Choice},
		Remaining: &BlockBetween{first: b.first, second: secondChoice.Remaining},
	}
}

func (b *BlockBetween) String() string {
	return fmt.Sprintf("BlockBetween(%s, %s)", b.first.String(), b.second.String())
}

// Compound represents a set of possible lines that are the union of the given sets.
type Compound struct {
	possibilities []PossibleLines
}

func MakeCompound(possibilities []PossibleLines, numLetters int) PossibleLines {
	if len(possibilities) == 0 {
		return MakeImpossible(numLetters)
	}
	if len(possibilities) == 1 {
		return possibilities[0]
	}
	// If any of possibilities is impossible OR compound, then we want to flatten into a shorter list:
	if slices.ContainsFunc(possibilities, func(p PossibleLines) bool {
		if isImpossible(p) {
			return true
		}
		if _, ok := p.(*Compound); ok {
			return true
		}
		return false
	}) {
		// Precompute the length so we can allocate in one go.
		length := 0
		for _, p := range possibilities {
			if isImpossible(p) {
				continue
			}
			if c, ok := p.(*Compound); ok {
				length += len(c.possibilities)
			} else {
				length++
			}
		}

		filtered := make([]PossibleLines, 0, length)
		for _, p := range possibilities {
			if isImpossible(p) {
				continue
			}
			if c, ok := p.(*Compound); ok {
				filtered = append(filtered, c.possibilities...)
			} else {
				filtered = append(filtered, p) // This is the only case where we're not a compound.
			}
		}
		return MakeCompound(filtered, numLetters)
	}

	return &Compound{possibilities: possibilities}
}

func (c *Compound) NumLetters() int {
	return c.possibilities[0].NumLetters()
}

func (c *Compound) MaxPossibilities() int64 {
	sum := int64(0)
	for _, p := range c.possibilities {
		sum += p.MaxPossibilities()
	}
	return sum
}

func (c *Compound) CharsAt(accumulate *CharSet, index int) {
	for _, p := range c.possibilities {
		p.CharsAt(accumulate, index)
		if accumulate.IsFull() {
			return
		}
	}
}

func (c *Compound) DefinitelyBlockedAt(index int) bool {
	for _, p := range c.possibilities {
		if !p.DefinitelyBlockedAt(index) {
			return false
		}
	}
	return true
}

func (c *Compound) DefiniteWords() []string {
	return nil
}

func (c *Compound) FilterAny(constraint *CharSet, index int) PossibleLines {
	if constraint.IsFull() {
		return c
	}

	var filtered []PossibleLines
	anyChangeInSubParts := false
	for ip, p := range c.possibilities {
		f := p.FilterAny(constraint, index)
		if !anyChangeInSubParts && p != f {
			// This is the first change, so we're gonna start building 'filtered' instead.
			anyChangeInSubParts = true
			filtered = append(filtered, c.possibilities[:ip]...)
		}

		if isImpossible(f) {
			continue
		}

		if !anyChangeInSubParts {
			continue
		}

		if c, ok := f.(*Compound); ok {
			filtered = append(filtered, c.possibilities...)
		} else {
			filtered = append(filtered, f)
		}
	}
	if !anyChangeInSubParts {
		return c
	}

	if len(filtered) == 0 {
		return MakeImpossible(c.NumLetters())
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	return MakeCompound(filtered, c.NumLetters())
}

func (c *Compound) Filter(constraint rune, index int) PossibleLines {
	var filtered []PossibleLines
	anyChangeInSubParts := false

	for ip, p := range c.possibilities {
		f := p.Filter(constraint, index)
		if !anyChangeInSubParts && p != f {
			// This is the first change, so we're gonna start building 'filtered' instead.
			anyChangeInSubParts = true
			filtered = append(filtered, c.possibilities[:ip]...)
		}

		if isImpossible(f) {
			continue
		}

		if anyChangeInSubParts {
			filtered = append(filtered, f)
		}
	}

	if !anyChangeInSubParts {
		return c
	}

	return MakeCompound(filtered, c.NumLetters())
}

func isImpossible(p PossibleLines) bool {
	_, isImpossible := p.(*Impossible)
	return isImpossible
}

func (c *Compound) RemoveWordOptions(words []string) PossibleLines {
	anyChanged := false
	var maybeFiltered []PossibleLines
	for i, p := range c.possibilities {
		f := p.RemoveWordOptions(words)
		if f == p && !anyChanged {
			// No filtering has occurred before and still no filtering is needed.
			continue
		}

		if f != p && !anyChanged {
			// We are the first to change.
			anyChanged = true
			if i > 0 {
				maybeFiltered = c.possibilities[:i]
			}
		}

		if !isImpossible(f) {
			maybeFiltered = append(maybeFiltered, f)
		}
	}

	if !anyChanged {
		return c
	}

	return MakeCompound(maybeFiltered, c.NumLetters())
}

func (c *Compound) FirstOrNull() *ConcreteLine {
	for _, p := range c.possibilities {
		if f := p.FirstOrNull(); f != nil {
			return f
		}
	}
	return nil
}

func (c *Compound) Iterate() iter.Seq[ConcreteLine] {
	return func(yield func(ConcreteLine) bool) {
		for _, p := range c.possibilities {
			for line := range p.Iterate() {
				if !yield(line) {
					return
				}
			}
		}
	}
}

func (c *Compound) MakeChoice() ChoiceStep {
	if len(c.possibilities) <= 1 {
		panic("BUG: Whenever this was created, it should have already been reduced to returning c.possibilities[1] alone")
	}
	if c.MaxPossibilities() <= 1 {
		panic("Cannot make a choice if MaxPossibilities <= 1")
	}
	// Weighted split: partition by MaxPossibilities sum to balance the two sides.
	total := int64(0)
	for _, p := range c.possibilities {
		total += p.MaxPossibilities()
	}
	half := total / 2
	acc := int64(0)
	splitIdx := 1
	for i, p := range c.possibilities {
		acc += p.MaxPossibilities()
		// ensure non-empty left side
		if acc >= half && i+1 < len(c.possibilities) {
			splitIdx = i + 1
			break
		}
	}

	choice, remaining := c.possibilities[:splitIdx], c.possibilities[splitIdx:]

	return ChoiceStep{
		Choice:    MakeCompound(choice, c.NumLetters()),
		Remaining: MakeCompound(remaining, c.NumLetters()),
	}
}

func (c *Compound) String() string {
	return fmt.Sprintf("Compound(%v and %d others)", c.possibilities[0], len(c.possibilities)-1)
}

// Definite represents a single possible line.
type Definite struct {
	line ConcreteLine
}

func MakeDefinite(line ConcreteLine) *Definite {
	return &Definite{line: line}
}

func (d *Definite) NumLetters() int {
	return len(d.line.Line)
}

func (d *Definite) MaxPossibilities() int64 {
	return 1
}

func (d *Definite) CharsAt(accumulate *CharSet, index int) {
	accumulate.Add(rune(d.line.Line[index]))
}

func (d *Definite) DefinitelyBlockedAt(index int) bool {
	return d.line.Line[index] == kBlocked
}

func (d *Definite) DefiniteWords() []string {
	return d.line.Words
}

func (d *Definite) FilterAny(constraint *CharSet, index int) PossibleLines {
	if constraint.IsFull() {
		return d
	}

	if constraint.Contains(rune(d.line.Line[index])) {
		return d
	}
	return MakeImpossible(d.NumLetters())
}

func (d *Definite) Filter(constraint rune, index int) PossibleLines {
	if constraint == rune(d.line.Line[index]) {
		return d
	}
	return MakeImpossible(d.NumLetters())
}

func (d *Definite) RemoveWordOptions(words []string) PossibleLines {
	if slices.ContainsFunc(words, func(word string) bool {
		if len(word) != d.NumLetters() {
			return false
		}
		return slices.Contains(d.line.Words, word)
	}) {
		return MakeImpossible(d.NumLetters())
	}
	return d
}

func (d *Definite) Iterate() iter.Seq[ConcreteLine] {
	return func(yield func(ConcreteLine) bool) {
		yield(d.line)
	}
}

func (d *Definite) FirstOrNull() *ConcreteLine {
	return &d.line
}

func (d *Definite) MakeChoice() ChoiceStep {
	panic("Cannot make a choice on a definite line")
}

func (d *Definite) String() string {
	return fmt.Sprintf("Definite(%s)", string(d.line.Line))
}
