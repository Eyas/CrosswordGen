package xw_generator

import (
	"context"
	"iter"
	"math/rand/v2"
	"slices"

	"crosswarped.com/ggg/xw_generator/generator/internal"
	"crosswarped.com/ggg/xw_generator/generator/primitives"
)

// Direction is an enum representing the direction of a line in a grid, either 'Horizontal' or 'Vertical'.
type Direction int

const (
	DirectionHorizontal Direction = iota
	DirectionVertical
)

type Generator struct {
	LineLength     int
	PreferredWords []string
	ObscureWords   []string
	ExcludedWords  []string
	MinWordLength  *int
	MaxWordLength  *int

	rand *rand.Rand

	// Do not access this field directly, use the allPossibleLines method instead.
	lazyAllPossibleLines primitives.PossibleLines
}

type GeneratorParams struct {
	MinWordLength int
	MaxWordLength int
}

func CreateGenerator(lineLength int, preferredWords, obscureWords, excludedWords []string, rand *rand.Rand, params GeneratorParams) *Generator {
	var minWordLength, maxWordLength *int
	if params.MinWordLength > 0 {
		minWordLength = &params.MinWordLength
	}
	if params.MaxWordLength > 0 {
		maxWordLength = &params.MaxWordLength
	}
	return &Generator{
		LineLength:     lineLength,
		PreferredWords: preferredWords,
		ObscureWords:   obscureWords,
		ExcludedWords:  excludedWords,
		MinWordLength:  minWordLength,
		MaxWordLength:  maxWordLength,
		rand:           rand,
	}
}

func (g *Generator) allPossibleLines(ctx context.Context) (primitives.PossibleLines, error) {
	var err error
	if g.lazyAllPossibleLines == nil {
		g.lazyAllPossibleLines, err = internal.AllPossibleLines(ctx, internal.AllPossibleLinesParams{
			LineLength:     g.LineLength,
			PreferredWords: g.PreferredWords,
			ObscureWords:   g.ObscureWords,
			ExcludedWords:  g.ExcludedWords,
		})
	}
	return g.lazyAllPossibleLines, err
}

// gridState represents the state of a grid being generated so far.
type gridState struct {
	down   []primitives.PossibleLines
	across []primitives.PossibleLines

	rand *rand.Rand
}

// getUndecidedIndexWLOG returns an index of an undecided line (i.e. a line that is not yet decided),
// preferring to return the "least undecided" line (i.e. the line with the lest possible lines).
func getUndecidedIndexWLOG(lines []primitives.PossibleLines, rand *rand.Rand) *int {
	type option struct {
		idx  int
		line primitives.PossibleLines
	}
	var least int64
	var opts []option
	for i, line := range lines {
		p := line.MaxPossibilities()
		if p <= 1 {
			continue
		}
		if least == 0 || p < least {
			least = p
		}
		opts = append(opts, option{idx: i, line: line})
	}

	if len(opts) == 0 {
		return nil
	}

	opts = slices.DeleteFunc(opts, func(o option) bool {
		return o.line.MaxPossibilities() != least
	})

	if len(opts) == 0 {
		return nil
	}

	// Shuffles the equivalent options:
	rand.Shuffle(len(opts), func(i, j int) {
		opts[i], opts[j] = opts[j], opts[i]
	})

	return &opts[0].idx
}

func (s gridState) getUndecidedIndexDown() *int {
	return getUndecidedIndexWLOG(s.down, s.rand)
}

func (s gridState) getUndecidedIndexAcross() *int {
	return getUndecidedIndexWLOG(s.across, s.rand)
}

func prefilter(ctx context.Context, s gridState, dir Direction) (gridState, bool) {
	if slices.ContainsFunc(s.down, impossible) || slices.ContainsFunc(s.across, impossible) {
		return s, false
	}
	if ctx.Err() != nil {
		return s, false
	}

	var toFilter, constraint []primitives.PossibleLines
	if dir == DirectionHorizontal {
		toFilter = s.across
		constraint = s.down
	} else {
		toFilter = s.down
		constraint = s.across
	}

	// i and j here are abstracted wlog based on toFilter/constraint, not truly
	// connected to Horizontal vs Vertical.
	//
	// available[i][j] is the set of characters that can be placed at (x, y) in the grid.
	available := make([][]primitives.CharSet, len(constraint))
	for i, constraintLine := range constraint {
		available[i] = make([]primitives.CharSet, constraintLine.NumLetters())

		for j := range constraintLine.NumLetters() {
			available[i][j] = *primitives.DefaultCharSet()
			constraintLine.CharsAt(&available[i][j], j)
		}
	}

	anyChanged := false
	for j := range toFilter {
		tf := toFilter[j]

		// if all characters in available[i] are full, then the line cannot be filtered
		// any further.
		allFull := true
		for i := range tf.NumLetters() {
			if !available[i][j].IsFull() {
				allFull = false
				break
			}
		}
		if allFull {
			continue
		}

		newTf := tf
		for i := range tf.NumLetters() {
			newTf = newTf.FilterAny(&available[i][j], i)
		}
		if newTf != tf {
			anyChanged = true
			toFilter[j] = newTf
		}
	}

	if dir == DirectionHorizontal {
		return gridState{across: toFilter, down: constraint, rand: s.rand}, anyChanged
	} else {
		return gridState{down: toFilter, across: constraint, rand: s.rand}, anyChanged
	}
}

func (g *Generator) PossibleGrids(ctx context.Context) iter.Seq[Grid] {
	return func(yield func(Grid) bool) {
		gs := gridState{
			down:   make([]primitives.PossibleLines, g.LineLength),
			across: make([]primitives.PossibleLines, g.LineLength),
			rand:   g.rand,
		}

		apl, err := g.allPossibleLines(ctx)
		if err != nil {
			return
		}

		for i := range gs.down {
			gs.down[i] = apl
		}
		for i := range gs.across {
			gs.across[i] = apl
		}

		seenReprs := make(map[string]bool)
		for grid := range possibleGridsAtRoot(ctx, &gs) {
			repr := grid.Repr()
			if seenReprs[repr] {
				continue
			}
			seenReprs[repr] = true
			if !yield(grid) {
				return
			}
		}
	}
}

func impossible(p primitives.PossibleLines) bool {
	return p.MaxPossibilities() == 0
}

func countWhere[T any](s []T, f func(T) bool) int {
	count := 0
	for _, v := range s {
		if f(v) {
			count++
		}
	}
	return count
}

func possibleGridsAtRoot(ctx context.Context, root *gridState) iter.Seq[Grid] {
	return func(yield func(Grid) bool) {
		if ctx.Err() != nil {
			return
		}

		// If we are at a point in our tree some row/column is unfillable, prune this tree.
		if slices.ContainsFunc(root.down, impossible) || slices.ContainsFunc(root.across, impossible) {
			return
		}

		// If there are any repeated words already, this is not a valid grid.
		existingWords := make(map[string]bool)
		hasDupes := false
		for _, line := range root.down {
			for _, word := range line.DefiniteWords() {
				if existingWords[word] {
					hasDupes = true
				}
				existingWords[word] = true
			}
		}
		for _, line := range root.across {
			for _, word := range line.DefiniteWords() {
				if existingWords[word] {
					hasDupes = true
				}
				existingWords[word] = true
			}
		}
		if hasDupes {
			return
		}

		priorNumBlocked := 0
		lineLength := len(root.down)
		for i := range lineLength {
			priorNumBlocked += countWhere(root.down, func(p primitives.PossibleLines) bool {
				return p.DefinitelyBlockedAt(i)
			})
		}

		// Prefilter
		direction := DirectionHorizontal
		for try := range 4 {
			newState, changed := prefilter(ctx, *root, direction)
			if !changed && try > 1 {
				break
			}

			root = &newState
			if direction == DirectionVertical {
				direction = DirectionHorizontal
			} else {
				direction = DirectionVertical
			}
		}
		if slices.ContainsFunc(root.down, impossible) || slices.ContainsFunc(root.across, impossible) {
			return
		}

		// If board is > 35% blocked, it's not worth iterating in it.
		numDefinitelyBlocked := 0
		for i := range lineLength {
			numDefinitelyBlocked += countWhere(root.down, func(p primitives.PossibleLines) bool {
				return p.DefinitelyBlockedAt(i)
			})
		}

		if numDefinitelyBlocked > ((lineLength * lineLength * 35) / 100) {
			return
		}

		// If board is entirely divided, s.t. no word spans two "halves" of the
		// board, we want to stop.
		//
		// We already can't have entire blocked lines. But we can have:
		// _ _ _ ` ` `
		// ` ` ` _ _ _
		//
		// This can still be better, e.g. it doesn't account for a "quadrant"
		// being cordoned off.
		if numDefinitelyBlocked > priorNumBlocked {
			if isBoardDefinitelyDivided(root) {
				return
			}
		}

		undecidedDown := root.getUndecidedIndexDown()
		undecidedAcross := root.getUndecidedIndexAcross()

		if undecidedDown == nil && undecidedAcross == nil {
			across := make([][]rune, len(root.across))

			for i, ac := range root.across {
				a := ac.FirstOrNull()
				d := root.down[i].FirstOrNull()

				if d == nil || a == nil {
					return
				}

				// If any column and row are completely the same, this is not a viable grid.
				if slices.Equal(d.Line, a.Line) {
					return
				}

				across[i] = a.Line
			}

			yield(NewGrid(across))
			return
		}

		var possibleGrids iter.Seq[Grid]

		if undecidedAcross == nil {
			possibleGrids = iterateAllPossibleGrids(ctx, root, *undecidedDown, DirectionVertical)
		} else if undecidedDown == nil {
			possibleGrids = iterateAllPossibleGrids(ctx, root, *undecidedAcross, DirectionHorizontal)
		} else if root.down[*undecidedDown].MaxPossibilities() <= root.across[*undecidedAcross].MaxPossibilities() {
			possibleGrids = iterateAllPossibleGrids(ctx, root, *undecidedDown, DirectionVertical)
		} else {
			possibleGrids = iterateAllPossibleGrids(ctx, root, *undecidedAcross, DirectionHorizontal)
		}

		for grid := range possibleGrids {
			if !yield(grid) {
				return
			}
		}
	}
}

func changed(before, after *gridState) bool {
	for i := range before.down {
		if before.down[i].MaxPossibilities() != after.down[i].MaxPossibilities() {
			return true
		}
		if before.across[i].MaxPossibilities() != after.across[i].MaxPossibilities() {
			return true
		}
	}
	return false
}

func isBoardDefinitelyDivided(state *gridState) bool {
	type blockExplorationState = int
	const (
		blockExplorationStateUnvisited blockExplorationState = iota
		blockExplorationStateVisited
		blockExplorationStateBlocked
	)

	grid := make([][]blockExplorationState, len(state.down))
	for i := range grid {
		grid[i] = make([]blockExplorationState, len(state.across))
	}

	for i := range grid {
		for j := range grid[i] {
			if state.down[i].DefinitelyBlockedAt(j) || state.across[j].DefinitelyBlockedAt(i) {
				grid[i][j] = blockExplorationStateBlocked
			}
		}
	}

	explore := make([]struct{ i, j int }, 0)
	// Enqueue the first univisited cell
	for i := range grid {
		for j := range grid[i] {
			if grid[i][j] == blockExplorationStateUnvisited {
				explore = append(explore, struct{ i, j int }{i, j})
				break
			}
		}
		if len(explore) > 0 {
			break
		}
	}

	for len(explore) > 0 {
		// Pop the first element
		ij := explore[0]
		explore = explore[1:]
		i, j := ij.i, ij.j

		if grid[i][j] != blockExplorationStateUnvisited {
			continue
		}

		grid[i][j] = blockExplorationStateVisited

		if (i-1) >= 0 && grid[i-1][j] == blockExplorationStateUnvisited {
			explore = append(explore, struct{ i, j int }{i - 1, j})
		}
		if (i+1) < len(grid) && grid[i+1][j] == blockExplorationStateUnvisited {
			explore = append(explore, struct{ i, j int }{i + 1, j})
		}
		if (j-1) >= 0 && grid[i][j-1] == blockExplorationStateUnvisited {
			explore = append(explore, struct{ i, j int }{i, j - 1})
		}
		if (j+1) < len(grid[i]) && grid[i][j+1] == blockExplorationStateUnvisited {
			explore = append(explore, struct{ i, j int }{i, j + 1})
		}
	}

	// If anything was still unreached, the board is definitely divided.
	if slices.ContainsFunc(grid, func(row []blockExplorationState) bool {
		return slices.ContainsFunc(row, func(cell blockExplorationState) bool {
			return cell == blockExplorationStateUnvisited
		})
	}) {
		return true
	}

	return false
}

func iterateAllPossibleGrids(ctx context.Context, root *gridState, index int, dir Direction) iter.Seq[Grid] {
	return func(yield func(Grid) bool) {
		if ctx.Err() != nil {
			return
		}

		var optionAxis, oppositeAxis []primitives.PossibleLines

		if dir == DirectionHorizontal {
			optionAxis = root.across
			oppositeAxis = root.down
		} else {
			optionAxis = root.down
			oppositeAxis = root.across
		}

		// Trim situations where horizontal and vertal words are same.
		for i := range optionAxis {
			if optionAxis[i].MaxPossibilities() > 1 {
				continue
			}
			if oppositeAxis[i].MaxPossibilities() > 1 {
				continue
			}

			optA := optionAxis[i].FirstOrNull()
			oppA := oppositeAxis[i].FirstOrNull()
			if optA == nil || oppA == nil {
				return
			}
			if slices.Equal(optA.Line, oppA.Line) {
				return
			}
		}

		options := optionAxis[index]

		// The below loop "makes decisions" and recurses. If we already
		// have one possibility, that means it's already pre-decided.
		if options.MaxPossibilities() <= 1 {
			return
		}

		if options.MaxPossibilities() >= 10 {
			for options.MaxPossibilities() > 1 {
				c := options.MakeChoice()

				// Clone oppositeAxis into attemptOpposite.
				attemptOpposite := make([]primitives.PossibleLines, len(oppositeAxis))
				copy(attemptOpposite, oppositeAxis)

				optionFinal := sliceSelectFunc(optionAxis, func(regular primitives.PossibleLines, idx int) primitives.PossibleLines {
					if idx == index {
						return c.Choice
					}
					return regular
				})

				// If any word appears more than once, this is not a valid grid.
				{
					duplicate := false
					for k := range attemptOpposite {
						first := attemptOpposite[k]
						second := optionFinal[k]
						if first.MaxPossibilities() > 1 || second.MaxPossibilities() > 1 {
							continue
						}
						f := first.FirstOrNull()
						s := second.FirstOrNull()
						if f == nil || s == nil {
							continue
						}
						if slices.Equal(f.Line, s.Line) {
							duplicate = true
							break

						}
					}
					if duplicate {
						return
					}
				}

				var newRoot *gridState
				if dir == DirectionHorizontal {
					newRoot = &gridState{
						down:   attemptOpposite,
						across: optionFinal,
						rand:   root.rand,
					}
				} else {
					newRoot = &gridState{
						down:   optionFinal,
						across: attemptOpposite,
						rand:   root.rand,
					}
				}

				if numDefiniteBlocks(c.Choice) > numDefiniteBlocks(options) {
					if isBoardDefinitelyDivided(newRoot) {
						return
					}
				}
				for final := range possibleGridsAtRoot(ctx, newRoot) {
					if !yield(final) {
						return
					}
				}

				options = c.Remaining
			}

			if options.MaxPossibilities() == 0 {
				return
			}
		}

		for attempt := range options.Iterate() {
			// If any word appears more than once, this is not a valid grid.
			wordCounts := make(map[string]int)
			hasDuplicate := false
			for _, word := range attempt.Words {
				wordCounts[word]++
				if wordCounts[word] > 1 {
					hasDuplicate = true
				}
			}
			if hasDuplicate {
				continue
			}

			// Clone oppositeAxis into attemptOpposite.
			attemptOpposite := make([]primitives.PossibleLines, len(oppositeAxis))
			copy(attemptOpposite, oppositeAxis)

			for i := range attempt.Line {
				// WLOG say we dir is Horizontal, and opopsite is Vertical.
				// we have:
				//
				// W O R D
				// _ _ _ _
				// _ _ _ _
				// _ _ _ _
				//
				// Then go over each COL (i), filtering s.t. possible lines
				// only include cases where col[i]'s |attempt|th character == attempt[i].
				var constriant = attempt.Line[i]

				attemptOpposite[i] = attemptOpposite[i].RemoveWordOptions(attempt.Words).Filter(constriant, index)

				if attemptOpposite[i].MaxPossibilities() == 1 {
					ao := attemptOpposite[i].FirstOrNull()
					if ao == nil || slices.Equal(ao.Line, attempt.Line) {
						return
					}
				}
			}

			if slices.ContainsFunc(attemptOpposite, impossible) {
				continue
			}

			oppositeFinal := attemptOpposite
			optionFinal := sliceSelectFunc(optionAxis,
				func(regular primitives.PossibleLines, idx int) primitives.PossibleLines {
					if idx == index {
						return primitives.MakeDefinite(attempt)
					}
					return regular.RemoveWordOptions(attempt.Words)
				})

			{
				duplicate := false
				for k := range attemptOpposite {
					first := attemptOpposite[k]
					second := optionFinal[k]
					if first.MaxPossibilities() > 1 || second.MaxPossibilities() > 1 {
						continue
					}
					f := first.FirstOrNull()
					s := second.FirstOrNull()
					if f == nil || s == nil {
						continue
					}
					if slices.Equal(f.Line, s.Line) {
						duplicate = true
						break
					}
				}
				if duplicate {
					return
				}
			}

			var newRoot *gridState
			if dir == DirectionHorizontal {
				newRoot = &gridState{
					down:   oppositeFinal,
					across: optionFinal,
					rand:   root.rand,
				}
			} else {
				newRoot = &gridState{
					down:   optionFinal,
					across: oppositeFinal,
					rand:   root.rand,
				}
			}

			for final := range possibleGridsAtRoot(ctx, newRoot) {
				if !yield(final) {
					return
				}
			}

		}
	}
}

func sliceSelectFunc[From any, To any](slice []From, f func(From, int) To) []To {
	result := make([]To, len(slice))
	for i, v := range slice {
		result[i] = f(v, i)
	}
	return result
}

func numDefiniteBlocks(state primitives.PossibleLines) int {
	acc := 0
	for i := range state.NumLetters() {
		if state.DefinitelyBlockedAt(i) {
			acc++
		}
	}
	return acc
}
