package internal

import (
	"context"
	"math/rand/v2"

	"crosswarped.com/ggg/xw_generator/generator/primitives"
)

type AllPossibleLinesParams struct {
	PreferredWords []string
	ObscureWords   []string
	ExcludedWords  []string
	LineLength     int
	MinWordLength  *int
	MaxWordLength  *int
}

type params struct {
	preferredWords []string
	obscureWords   []string
	excludedWords  []string
	lineLength     int
	minWordLength  int
	maxWordLength  int
}

func asParams(p AllPossibleLinesParams) params {
	pp := params{
		preferredWords: p.PreferredWords,
		obscureWords:   p.ObscureWords,
		excludedWords:  p.ExcludedWords,
		lineLength:     p.LineLength,
	}

	if p.MinWordLength == nil {
		pp.minWordLength = 3
	} else {
		pp.minWordLength = *p.MinWordLength
	}

	if p.MaxWordLength == nil {
		pp.maxWordLength = p.LineLength
	} else {
		pp.maxWordLength = *p.MaxWordLength
	}

	return pp
}

type allPossibleLineState struct {
	lineLength    int
	minWordLength int
	maxWordLength int

	preferredWordsByLength map[int][]string
	obscureWordsByLength   map[int][]string

	excludedWords map[string]bool

	memoizedLines map[int]primitives.PossibleLines
}

func (s *allPossibleLineState) allPossibleLines(ctx context.Context, atLength int) primitives.PossibleLines {
	if ctx.Err() != nil {
		return primitives.MakeImpossible(atLength)
	}

	memo, ok := s.memoizedLines[atLength]
	if ok {
		return memo
	}

	if atLength > s.lineLength {
		panic("atLength > lineLength -- this should never happen")
	}

	if atLength < s.minWordLength {
		return primitives.MakeImpossible(atLength)
	}

	words := primitives.MakeWords(s.preferredWordsByLength[atLength], s.obscureWordsByLength[atLength], atLength)

	var blockBetweenPossibilities []primitives.PossibleLines
	// recurse into all combination of [ANYTHING]*[ANYTHING]
	//
	// For length 10:
	// 0 1 2 3 4 5 6 7 8 9
	// _ _ _ _ _ _ _ _ _ _
	//       ^     ^
	// Blockage can be anywhere etween idx 3 and len-4 (inclusive).
	betweenIdxStart := s.minWordLength
	betweenIdxFromEnd := (1 + s.minWordLength)
	if atLength >= (betweenIdxStart + betweenIdxFromEnd) {
		blockBetweenPossibilities = make([]primitives.PossibleLines, 0, atLength-6)
		for i := betweenIdxStart; i <= atLength-betweenIdxFromEnd; i++ {
			firstLength := i                   // Always >= minWordLength.
			secondLength := atLength - (i + 1) // Always >= minWordLength.

			blockBetweenPossibilities = append(blockBetweenPossibilities, primitives.MakeBlockBetween(
				s.allPossibleLines(ctx, firstLength),
				s.allPossibleLines(ctx, secondLength),
			))
		}

		// Shuffle the possibilities
		rand.Shuffle(len(blockBetweenPossibilities), func(i, j int) {
			blockBetweenPossibilities[i], blockBetweenPossibilities[j] = blockBetweenPossibilities[j], blockBetweenPossibilities[i]
		})
	}

	// recurse into *[ANYTHING], and [ANYTHING]*
	var blockBefore, blockAfter primitives.PossibleLines
	if smaller := s.allPossibleLines(ctx, atLength-1); !isImpossible(smaller) {
		blockBefore = primitives.MakeBlockBefore(smaller)
		blockAfter = primitives.MakeBlockAfter(smaller)
	}

	if blockBefore == nil && blockAfter == nil && len(blockBetweenPossibilities) == 0 {
		s.memoizedLines[atLength] = words
		return words
	}

	allPossibilities := []primitives.PossibleLines{words}
	if blockBefore != nil {
		allPossibilities = append(allPossibilities, blockBefore)
	}
	if blockAfter != nil {
		allPossibilities = append(allPossibilities, blockAfter)
	}
	// Append all elements from blockBetweenPossibilities efficiently
	if len(blockBetweenPossibilities) > 0 {
		allPossibilities = append(allPossibilities, blockBetweenPossibilities...)
	}
	compound := primitives.MakeCompound(allPossibilities)
	s.memoizedLines[atLength] = compound
	return compound
}

// AllPossibleLines returns a set of all possible lines for the given parameters.
func AllPossibleLines(ctx context.Context, p AllPossibleLinesParams) (primitives.PossibleLines, error) {
	params := asParams(p)
	state := allPossibleLineState{
		lineLength:    params.lineLength,
		minWordLength: params.minWordLength,
		maxWordLength: params.maxWordLength,
	}
	state.memoizedLines = make(map[int]primitives.PossibleLines)

	state.preferredWordsByLength = make(map[int][]string)
	state.obscureWordsByLength = make(map[int][]string)
	state.excludedWords = make(map[string]bool)

	for _, word := range params.excludedWords {
		state.excludedWords[word] = true
	}

	for _, word := range params.preferredWords {
		if len(word) < params.minWordLength || len(word) > params.maxWordLength {
			continue
		}
		if _, ok := state.excludedWords[word]; ok {
			continue
		}
		state.preferredWordsByLength[len(word)] = append(state.preferredWordsByLength[len(word)], word)
	}

	for _, word := range params.obscureWords {
		if len(word) < params.minWordLength || len(word) > params.maxWordLength {
			continue
		}
		if _, ok := state.excludedWords[word]; ok {
			continue
		}
		state.obscureWordsByLength[len(word)] = append(state.obscureWordsByLength[len(word)], word)
	}

	for i := params.minWordLength; i <= params.lineLength; i++ {
		if _, ok := state.preferredWordsByLength[i]; !ok {
			state.preferredWordsByLength[i] = []string{}
		}
		if _, ok := state.obscureWordsByLength[i]; !ok {
			state.obscureWordsByLength[i] = []string{}
		}
	}

	possibleLines := state.allPossibleLines(ctx, params.lineLength)
	return possibleLines, ctx.Err()
}

func isImpossible(p primitives.PossibleLines) bool {
	_, isImpossible := p.(*primitives.Impossible)
	return isImpossible
}
