package primitives

import (
	"fmt"
	"iter"
	"slices"
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

	// RemoveWordOption strips the possible lines to no longer include a given word.
	RemoveWordOption(word string) PossibleLines

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

func (i *Impossible) RemoveWordOption(word string) PossibleLines {
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
	preferred []string
	obscure   []string
}

func MakeWords(preferred, obscure []string) PossibleLines {
	if len(preferred) == 0 && len(obscure) == 0 {
		return MakeImpossible(0)
	}
	if len(preferred) == 1 && len(obscure) == 0 {
		return MakeDefinite(ConcreteLine{Line: []rune(preferred[0]), Words: []string{preferred[0]}})
	}
	if len(preferred) == 0 && len(obscure) == 1 {
		return MakeDefinite(ConcreteLine{Line: []rune(obscure[0]), Words: []string{obscure[0]}})
	}
	return &Words{preferred: preferred, obscure: obscure}
}

func (w *Words) NumLetters() int {
	if len(w.preferred) == 0 {
		return len(w.obscure[0])
	}
	return len(w.preferred[0])
}

func (w *Words) MaxPossibilities() int64 {
	return int64(len(w.preferred) + len(w.obscure))
}

func (w *Words) CharsAt(accumulate *CharSet, index int) {
	if accumulate.IsFull() {
		return
	}
	for _, word := range w.preferred {
		accumulate.Add(rune(word[index]))
	}
	for _, word := range w.obscure {
		accumulate.Add(rune(word[index]))
	}
}

func (w *Words) DefinitelyBlockedAt(index int) bool {
	return false
}

func (w *Words) DefiniteWords() []string {
	if len(w.preferred) == 1 && len(w.obscure) == 0 {
		return []string{w.preferred[0]}
	}
	if len(w.preferred) == 0 && len(w.obscure) == 1 {
		return []string{w.obscure[0]}
	}
	return nil
}

func (w *Words) FilterAny(constraint *CharSet, index int) PossibleLines {
	if constraint.IsFull() {
		return w
	}

	// Lazy: First check if any of the words in either list don't match the filter.
	// Otherwise we don't need to copy the lists
	if !slices.ContainsFunc(w.preferred, func(word string) bool {
		return !constraint.Contains(rune(word[index]))
	}) && !slices.ContainsFunc(w.obscure, func(word string) bool {
		return !constraint.Contains(rune(word[index]))
	}) {
		return w
	}

	filteredPreferred := make([]string, 0, len(w.preferred)/2)
	filteredObscure := make([]string, 0, len(w.obscure)/2)
	for _, word := range w.preferred {
		if constraint.Contains(rune(word[index])) {
			filteredPreferred = append(filteredPreferred, word)
		}
	}
	for _, word := range w.obscure {
		if constraint.Contains(rune(word[index])) {
			filteredObscure = append(filteredObscure, word)
		}
	}
	if len(filteredPreferred) == 0 && len(filteredObscure) == 0 {
		return MakeImpossible(w.NumLetters())
	}
	return MakeWords(filteredPreferred, filteredObscure)
}

func (w *Words) Filter(constraint rune, index int) PossibleLines {
	if constraint == kBlocked {
		return MakeImpossible(w.NumLetters())
	}
	filteredPreferred := make([]string, 0, len(w.preferred)/2)
	filteredObscure := make([]string, 0, len(w.obscure)/2)
	for _, word := range w.preferred {
		if rune(word[index]) == constraint {
			filteredPreferred = append(filteredPreferred, word)
		}
	}
	for _, word := range w.obscure {
		if rune(word[index]) == constraint {
			filteredObscure = append(filteredObscure, word)
		}
	}
	if len(filteredPreferred) == 0 && len(filteredObscure) == 0 {
		return MakeImpossible(w.NumLetters())
	}
	return MakeWords(filteredPreferred, filteredObscure)
}

func (w *Words) RemoveWordOption(word string) PossibleLines {
	if len(word) != w.NumLetters() {
		return w
	}

	// If word is not in either list, return w,
	// otherwise, we trim it either/both lists.
	if !slices.Contains(w.preferred, word) && !slices.Contains(w.obscure, word) {
		return w
	}

	fp := make([]string, 0, len(w.preferred))
	fo := make([]string, 0, len(w.obscure))
	for _, p := range w.preferred {
		if p != word {
			fp = append(fp, p)
		}
	}
	for _, o := range w.obscure {
		if o != word {
			fo = append(fo, o)
		}
	}
	return MakeWords(fp, fo)
}

func (w *Words) FirstOrNull() *ConcreteLine {
	if len(w.preferred) > 0 {
		return &ConcreteLine{Line: []rune(w.preferred[0]), Words: []string{w.preferred[0]}}
	}
	if len(w.obscure) > 0 {
		return &ConcreteLine{Line: []rune(w.obscure[0]), Words: []string{w.obscure[0]}}
	}
	return nil
}

func (w *Words) Iterate() iter.Seq[ConcreteLine] {
	return func(yield func(ConcreteLine) bool) {
		for _, word := range w.preferred {
			if !yield(ConcreteLine{Line: []rune(word), Words: []string{word}}) {
				return
			}
		}
		for _, word := range w.obscure {
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

	if len(w.preferred) == 1 && len(w.obscure) == 1 {
		return ChoiceStep{
			Choice:    &Definite{line: ConcreteLine{Line: []rune(w.preferred[0]), Words: []string{w.preferred[0]}}},
			Remaining: &Definite{line: ConcreteLine{Line: []rune(w.obscure[0]), Words: []string{w.obscure[0]}}},
		}
	}

	// If preferred and obscure are about the same length, return a choice between them,
	// so we prefer regular over obscure by default.
	//
	// Let's use if preferred is 30-70% of the length of obscure:
	prefLen := len(w.preferred)
	obscLen := len(w.obscure)
	if prefLen > (3*obscLen)/10 && prefLen < (7*obscLen)/10 {
		return ChoiceStep{
			Choice:    &Definite{line: ConcreteLine{Line: []rune(w.preferred[0]), Words: []string{w.preferred[0]}}},
			Remaining: &Definite{line: ConcreteLine{Line: []rune(w.obscure[0]), Words: []string{w.obscure[0]}}},
		}
	}

	// Othrewise, we split both in half.
	pref1, pref2 := w.preferred[:len(w.preferred)/2], w.preferred[len(w.preferred)/2:]
	obsc1, obsc2 := w.obscure[:len(w.obscure)/2], w.obscure[len(w.obscure)/2:]

	choice := MakeWords(pref1, obsc1)
	remaining := MakeWords(pref2, obsc2)

	return ChoiceStep{Choice: choice, Remaining: remaining}
}

func arrayStr(arr []string) string {
	if len(arr) == 0 {
		return "[]"
	}
	if len(arr) == 1 {
		return fmt.Sprintf("[%s]", arr[0])
	}

	return fmt.Sprintf("[%s, ...%d]", arr[0], len(arr)-1)
}

func (w *Words) String() string {
	return fmt.Sprintf("Words(%s, %s)", arrayStr(w.preferred), arrayStr(w.obscure))
}

// BlockBefore represents a line that has a blocked cell at the beginning.
type BlockBefore struct {
	lines PossibleLines
}

func MakeBlockBefore(lines PossibleLines) *BlockBefore {
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
		return MakeImpossible(b.NumLetters() + 1)
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

func (b *BlockBefore) RemoveWordOption(word string) PossibleLines {
	if len(word) > b.lines.NumLetters() {
		return b
	}

	return b.build(b.lines.RemoveWordOption(word))
}

func (b *BlockBefore) FirstOrNull() *ConcreteLine {
	c := b.lines.FirstOrNull()
	if c == nil {
		return nil
	}
	return &ConcreteLine{Line: append([]rune{kBlocked}, c.Line...), Words: c.Words}
}

func (b *BlockBefore) MakeChoice() ChoiceStep {
	return ChoiceStep{
		Choice:    &BlockBefore{lines: b.lines.MakeChoice().Choice},
		Remaining: &BlockBefore{lines: b.lines.MakeChoice().Remaining},
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

func MakeBlockAfter(lines PossibleLines) *BlockAfter {
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
		return MakeImpossible(b.NumLetters() + 1)
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

func (b *BlockAfter) RemoveWordOption(word string) PossibleLines {
	if len(word) > b.lines.NumLetters() {
		return b
	}

	return b.build(b.lines.RemoveWordOption(word))
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
	return ChoiceStep{
		Choice:    &BlockAfter{lines: b.lines.MakeChoice().Choice},
		Remaining: &BlockAfter{lines: b.lines.MakeChoice().Remaining},
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

func MakeBlockBetween(first, second PossibleLines) *BlockBetween {
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

func (b *BlockBetween) RemoveWordOption(word string) PossibleLines {
	if len(word) > b.first.NumLetters() && len(word) > b.second.NumLetters() {
		return b
	}

	return b.build(b.first.RemoveWordOption(word), b.second.RemoveWordOption(word))
}

func (b *BlockBetween) FirstOrNull() *ConcreteLine {
	f := b.first.FirstOrNull()
	if f == nil {
		return nil
	}
	return &ConcreteLine{Line: append(f.Line, kBlocked), Words: f.Words}
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
	if b.first.MaxPossibilities() > 1 {
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

func MakeCompound(possibilities []PossibleLines) PossibleLines {
	if len(possibilities) == 0 {
		return MakeImpossible(0)
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
		filtered := make([]PossibleLines, 0, len(possibilities))
		for _, p := range possibilities {
			if isImpossible(p) {
				continue
			}
			c, ok := p.(*Compound)
			if ok {
				filtered = append(filtered, c.possibilities...)
			} else {
				filtered = append(filtered, p)
			}
		}
		return MakeCompound(filtered)
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

	filtered := make([]PossibleLines, 0, len(c.possibilities))
	for _, p := range c.possibilities {
		f := p.FilterAny(constraint, index)
		if isImpossible(f) {
			continue
		}
		if c, ok := f.(*Compound); ok {
			filtered = append(filtered, c.possibilities...)
		} else {
			filtered = append(filtered, f)
		}
	}

	if len(filtered) == 0 {
		return MakeImpossible(c.NumLetters())
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	return MakeCompound(filtered)
}

func (c *Compound) Filter(constraint rune, index int) PossibleLines {
	filtered := make([]PossibleLines, 0, len(c.possibilities))
	for _, p := range c.possibilities {
		f := p.Filter(constraint, index)
		if isImpossible(f) {
			continue
		}
		filtered = append(filtered, f)
	}

	if len(filtered) == 0 {
		return MakeImpossible(c.NumLetters())
	}

	if len(filtered) == 1 {
		return filtered[0]
	}
	return MakeCompound(filtered)
}

func isImpossible(p PossibleLines) bool {
	_, isImpossible := p.(*Impossible)
	return isImpossible
}

func (c *Compound) RemoveWordOption(word string) PossibleLines {
	filtered := make([]PossibleLines, 0, len(c.possibilities))
	for _, p := range c.possibilities {
		f := p.RemoveWordOption(word)
		if isImpossible(f) {
			continue
		}
		filtered = append(filtered, f)
	}

	if len(filtered) == 0 {
		return MakeImpossible(c.NumLetters())
	}

	if len(filtered) == 1 {
		return filtered[0]
	}
	return MakeCompound(filtered)
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

	choice, remaining := c.possibilities[:len(c.possibilities)/2], c.possibilities[len(c.possibilities)/2:]

	return ChoiceStep{
		Choice:    MakeCompound(choice),
		Remaining: MakeCompound(remaining),
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

func (d *Definite) RemoveWordOption(word string) PossibleLines {
	if len(word) > len(d.line.Line) {
		return d
	}

	if slices.Contains(d.line.Words, word) {
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
