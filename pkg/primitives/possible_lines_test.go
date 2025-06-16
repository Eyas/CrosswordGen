package primitives

import (
	"reflect"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Helper function to check if a PossibleLines is Impossible
func isActuallyImpossible(pl PossibleLines) bool {
	_, ok := pl.(*Impossible)
	return ok
}

func everything(from, to rune) []rune {
	chars := make([]rune, 0, to-from+1)
	for c := from; c <= to; c++ {
		chars = append(chars, c)
	}
	return chars
}

func TestImpossible(t *testing.T) {
	impossible := MakeImpossible(5)

	t.Run("Properties", func(t *testing.T) {
		if impossible.NumLetters() != 5 {
			t.Errorf("Expected NumLetters to be 5, got %d", impossible.NumLetters())
		}

		if impossible.MaxPossibilities() != 0 {
			t.Errorf("Expected MaxPossibilities to be 0, got %d", impossible.MaxPossibilities())
		}
	})

	t.Run("CharsAt", func(t *testing.T) {
		charSetForCharsAt := DefaultCharSet()
		impossible.CharsAt(charSetForCharsAt, 0)
		if charSetForCharsAt.Count() != 0 {
			t.Errorf("Expected CharsAt to add no characters to set, got %d", charSetForCharsAt.Count())
		}
	})

	t.Run("DefinitelyBlockedAt", func(t *testing.T) {
		if impossible.DefinitelyBlockedAt(0) {
			t.Error("Expected DefinitelyBlockedAt to be false")
		}
	})

	t.Run("DefiniteWords", func(t *testing.T) {
		if impossible.DefiniteWords() != nil {
			t.Errorf("Expected DefiniteWords to be nil, got %v", impossible.DefiniteWords())
		}
	})

	t.Run("FilterAny", func(t *testing.T) {
		constraintSetForFilterAny := DefaultCharSet()
		constraintSetForFilterAny.Add('a')
		if !isActuallyImpossible(impossible.FilterAny(constraintSetForFilterAny, 0)) {
			t.Error("Expected FilterAny to return Impossible")
		}
	})

	t.Run("Filter", func(t *testing.T) {
		if !isActuallyImpossible(impossible.Filter('a', 0)) {
			t.Error("Expected Filter to return Impossible")
		}
	})

	t.Run("RemoveWordOption", func(t *testing.T) {
		if !isActuallyImpossible(impossible.RemoveWordOption("test")) {
			t.Error("Expected RemoveWordOption to return Impossible")
		}
	})

	t.Run("Iterate", func(t *testing.T) {
		count := 0
		for range impossible.Iterate() {
			count++
		}
		if count != 0 {
			t.Errorf("Expected Iterate to yield 0 items, got %d", count)
		}
	})

	t.Run("FirstOrNull", func(t *testing.T) {
		if impossible.FirstOrNull() != nil {
			t.Error("Expected FirstOrNull to return nil")
		}
	})

	t.Run("Caching", func(t *testing.T) {
		impossible2 := MakeImpossible(5)
		if impossible != impossible2 {
			t.Error("Expected MakeImpossible to return cached instance for same length")
		}
		impossible3 := MakeImpossible(6)
		if impossible == impossible3 {
			t.Error("Expected MakeImpossible to return different instance for different length")
		}
	})
}

func TestWords(t *testing.T) {
	// Test MakeWords
	t.Run("MakeWords", func(t *testing.T) {
		t.Run("EmptyPreferredAndObscure", func(t *testing.T) {
			pl := MakeWords([]string{}, []string{})
			if !isActuallyImpossible(pl) {
				t.Errorf("MakeWords with empty slices should return Impossible, got %T", pl)
			}
		})

		t.Run("SinglePreferredWord", func(t *testing.T) {
			pl := MakeWords([]string{"apple"}, []string{})
			if _, ok := pl.(*Definite); !ok {
				t.Errorf("MakeWords with single preferred word should return Definite, got %T", pl)
			}
			if pl.NumLetters() != 5 {
				t.Errorf("Expected NumLetters 5, got %d", pl.NumLetters())
			}
		})

		t.Run("SingleObscureWord", func(t *testing.T) {
			pl := MakeWords([]string{}, []string{"banana"})
			if _, ok := pl.(*Definite); !ok {
				t.Errorf("MakeWords with single obscure word should return Definite, got %T", pl)
			}
			if pl.NumLetters() != 6 {
				t.Errorf("Expected NumLetters 6, got %d", pl.NumLetters())
			}
		})

		// Test cases where MakeWords returns a Words object.
		for _, tc := range []struct {
			name               string
			preferred, obscure []string
			expectedNumLetters int
		}{
			{"two preferred", []string{"apple", "bobby"}, []string{}, 5},
			{"two obscure", []string{}, []string{"banana", "nababa"}, 6},
			{"one preferred, one obscure", []string{"apple"}, []string{"bobby"}, 5},
		} {
			t.Run(tc.name, func(t *testing.T) {
				pl := MakeWords(tc.preferred, tc.obscure)
				if _, ok := pl.(*Words); !ok {
					t.Errorf("MakeWords with %s should return Words, got %T", tc.name, pl)
				}
				if pl.NumLetters() != tc.expectedNumLetters {
					t.Errorf("Expected NumLetters %d, got %d", tc.expectedNumLetters, pl.NumLetters())
				}
			})
		}
	})

	// Setup for testing Words methods
	preferred := []string{"cat", "car"}
	obscure := []string{"cot", "cop"}
	wordsInstance := MakeWords(preferred, obscure)
	words, ok := wordsInstance.(*Words)
	if !ok {
		t.Fatalf("MakeWords did not return a *Words instance as expected for method testing, got %T. Skipping Words method tests.", wordsInstance)
		return
	}

	t.Run("Properties", func(t *testing.T) {
		if words.NumLetters() != 3 {
			t.Errorf("Expected NumLetters 3, got %d", words.NumLetters())
		}

		if words.MaxPossibilities() != 4 {
			t.Errorf("Expected MaxPossibilities 4, got %d", words.MaxPossibilities())
		}
	})

	t.Run("CharsAt", func(t *testing.T) {
		cs := DefaultCharSet()
		words.CharsAt(cs, 0) // C from cat, car, cot, cop
		if !cs.Contains('c') {
			t.Error("CharsAt(0) should contain 'c'")
		}
		if cs.Count() != 1 {
			t.Errorf("CharsAt(0) should only contain 'c', count was %d", cs.Count())
		}

		cs = DefaultCharSet() // Re-initialize instead of Clear()
		words.CharsAt(cs, 1)  // A from cat, car; O from cot, cop
		if !cs.Contains('a') || !cs.Contains('o') {
			t.Error("CharsAt(1) should contain 'a' and 'o'")
		}
		if cs.Count() != 2 {
			t.Errorf("CharsAt(1) should contain 2 chars, count was %d", cs.Count())
		}
	})

	t.Run("DefinitelyBlockedAt", func(t *testing.T) {
		if words.DefinitelyBlockedAt(0) {
			t.Error("DefinitelyBlockedAt should be false for Words")
		}
	})

	t.Run("DefiniteWords", func(t *testing.T) {
		if words.DefiniteWords() != nil {
			t.Errorf("Expected DefiniteWords to be nil for multiple options, got %v", words.DefiniteWords())
		}
		// Test DefiniteWords for a case that MakeWords returns *Definite
		oneWordPl := MakeWords([]string{"one"}, []string{})
		if oneWordDefinite, ok := oneWordPl.(*Definite); ok {
			expectedDefiniteWords := []string{"one"}
			actualDefiniteWords := oneWordDefinite.DefiniteWords()
			if diff := cmp.Diff(expectedDefiniteWords, actualDefiniteWords); diff != "" {
				t.Errorf("DefiniteWords mismatch (-want +got): %s", diff)
			}
		} else {
			t.Errorf("Expected MakeWords with 'one' to return *Definite, got %T", oneWordPl)
		}
	})

	t.Run("FilterAny", func(t *testing.T) {
		filterSet := DefaultCharSet()
		filterSet.Add('a') // Words with A at index 1: car, cat
		filteredAny := words.FilterAny(filterSet, 1)
		if filteredAny.MaxPossibilities() != 2 {
			t.Errorf("FilterAny for 'a' at index 1 should yield 2 possibilities, got %d", filteredAny.MaxPossibilities())
		}
		firstFilteredAny := filteredAny.FirstOrNull()
		if firstFilteredAny == nil || !slices.Contains([]string{"car", "cat"}, string(firstFilteredAny.Line)) {
			t.Errorf("FilterAny for 'a' at index 1 should be car or cat, got %v", string(firstFilteredAny.Line))
		}
	})

	// words = ({"cat", "car"}, {"cot", "cop"})
	for _, tc := range []struct {
		name          string
		filterSet     []rune
		index         int
		want          PossibleLines
		wantUnchanged bool
	}{
		{"unchanged with everything", everything('`', 'z'), 0, words, true},
		{"unchanged with everything - different index", everything('`', 'z'), 1, words, true},
		{"unchanged with all letters", everything('a', 'z'), 0, words, true},
		{"unchanged with all letters - different index", everything('a', 'z'), 1, words, true},
		{"impossible with nothing", []rune{}, 0, MakeImpossible(3), false},
		{"impossible with nothing - different index", []rune{}, 1, MakeImpossible(3), false},
		{"basically unchanged when filter matches all", []rune{'c'}, 0, words, false},
		{"only regulars remain when we only match that", []rune{'a'}, 1, &Words{preferred: []string{"cat", "car"}, obscure: []string{}}, false},
		{"one regular and one obscure remain", []rune{'t'}, 2, &Words{preferred: []string{"cat"}, obscure: []string{"cot"}}, false},
		{"becomes a definite when only one remains - regular", []rune{'r'}, 2, &Definite{line: ConcreteLine{Line: []rune{'c', 'a', 'r'}, Words: []string{"car"}}}, false},
		{"becomes a definite when only one remains - obscure", []rune{'p'}, 2, &Definite{line: ConcreteLine{Line: []rune{'c', 'o', 'p'}, Words: []string{"cop"}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cs := DefaultCharSet()
			for _, c := range tc.filterSet {
				cs.Add(c)
			}
			got := words.FilterAny(cs, tc.index)
			if tc.wantUnchanged && got != words {
				t.Errorf("FilterAny(%v, %d) = %v, want unchanged", cs, tc.index, got)
			}
			if tc.wantUnchanged {
				return
			}
			if !reflect.DeepEqual(tc.want, got) {
				t.Errorf("FilterAny(%v, %d) = %v, want %v", cs, tc.index, got, tc.want)
			}
		})
	}

	t.Run("Filter", func(t *testing.T) {
		// Filter by char
		filtered := words.Filter('t', 2) // cat, cot
		if filtered.MaxPossibilities() != 2 {
			t.Errorf("Filter for 't' at index 2 should yield 2 possibilities, got %d", filtered.MaxPossibilities())
		}
		// Filter by kBlocked (should be impossible)
		filteredBlocked := words.Filter(kBlocked, 0)
		if !isActuallyImpossible(filteredBlocked) {
			t.Errorf("Filter for kBlocked should return Impossible, got %T", filteredBlocked)
		}
	})

	t.Run("RemoveWordOption", func(t *testing.T) {
		removedCAT := words.RemoveWordOption("cat")
		if removedCAT.MaxPossibilities() != 3 {
			t.Errorf("RemoveWordOption(\"cat\") should leave 3 possibilities, got %d", removedCAT.MaxPossibilities())
		}
		iteratedWords := []string{}
		for w := range removedCAT.Iterate() {
			iteratedWords = append(iteratedWords, string(w.Line))
		}
		expectedAfterRemove := []string{"car", "cot", "cop"} // Order matters based on preferred/obscure
		if diff := cmp.Diff(expectedAfterRemove, iteratedWords); diff != "" {
			t.Errorf("Words after removing cat (-want +got): %s", diff)
		}
	})

	t.Run("FirstOrNull", func(t *testing.T) {
		first := words.FirstOrNull()
		if first == nil || string(first.Line) != "cat" { // "cat" is the first preferred word
			t.Errorf("Expected FirstOrNull to be cat, got %v", first)
		}
	})

	t.Run("Iterate", func(t *testing.T) {
		iteratedAll := []string{}
		for w := range words.Iterate() {
			iteratedAll = append(iteratedAll, string(w.Line))
		}
		expectedAll := []string{"cat", "car", "cot", "cop"} // Order matters
		if diff := cmp.Diff(expectedAll, iteratedAll); diff != "" {
			t.Errorf("Iterate all (-want +got): %s", diff)
		}
	})

	t.Run("MakeChoice", func(t *testing.T) {
		t.Run("SimpleSplitPreferred", func(t *testing.T) {
			wordsToSplitInstance := MakeWords([]string{"aaaa", "bbbb"}, []string{})
			wordsToSplit, ok := wordsToSplitInstance.(*Words)
			if !ok {
				t.Fatalf("MakeWords for MakeChoice test did not return *Words, got %T", wordsToSplitInstance)
			}
			choice := wordsToSplit.MakeChoice()
			if choice.Choice.MaxPossibilities() != 1 || string(choice.Choice.FirstOrNull().Line) != "aaaa" {
				t.Errorf("MakeChoice.Choice expected aaaa, got %v", choice.Choice.FirstOrNull())
			}
			if choice.Remaining.MaxPossibilities() != 1 || string(choice.Remaining.FirstOrNull().Line) != "bbbb" {
				t.Errorf("MakeChoice.Remaining expected bbbb, got %v", choice.Remaining.FirstOrNull())
			}
		})

		t.Run("PreferredAndObscureSplit", func(t *testing.T) {
			wordsMixedSplitInstance := MakeWords([]string{"pref"}, []string{"obsc"})
			wordsMixedSplit, ok := wordsMixedSplitInstance.(*Words)
			if !ok {
				t.Fatalf("MakeWords for mixed MakeChoice test did not return *Words, got %T", wordsMixedSplitInstance)
			}

			choiceMixed := wordsMixedSplit.MakeChoice()
			if choiceMixed.Choice.MaxPossibilities() != 1 || string(choiceMixed.Choice.FirstOrNull().Line) != "pref" {
				t.Errorf("MakeChoice.Choice for mixed (1,1) expected pref, got %v", choiceMixed.Choice.FirstOrNull())
			}
			if choiceMixed.Remaining.MaxPossibilities() != 1 || string(choiceMixed.Remaining.FirstOrNull().Line) != "obsc" {
				t.Errorf("MakeChoice.Remaining for mixed (1,1) expected obsc, got %v", choiceMixed.Remaining.FirstOrNull())
			}
		})

		t.Run("PanicOnSingleFilteredWord", func(t *testing.T) {
			wordsSingleFilteredInstance := MakeWords([]string{"onlyone", "another"}, []string{})
			wordsSingleFiltered, ok := wordsSingleFilteredInstance.(*Words)
			if !ok {
				t.Fatalf("MakeWords for panic test did not return *Words, got %T", wordsSingleFilteredInstance)
			}

			filteredToOne := wordsSingleFiltered.Filter('o', 0).Filter('n', 1).Filter('l', 2).Filter('y', 3)
			if filteredToOne.MaxPossibilities() != 1 {
				t.Fatalf("Expected filtered to one to have 1 possibility, got %d", filteredToOne.MaxPossibilities())
			}

			panicked := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = true
					}
				}()
				filteredToOne.MakeChoice()
			}()
			if !panicked {
				t.Error("Expected MakeChoice to panic when called on a PossibleLines with only one possibility (either Definite or Words with 1 item)")
			}
		})
	})
}

func TestDefinite(t *testing.T) {
	line := ConcreteLine{Line: []rune("test"), Words: []string{"test"}}
	definite := MakeDefinite(line)

	t.Run("Properties", func(t *testing.T) {
		if definite.NumLetters() != 4 {
			t.Errorf("Expected NumLetters 4, got %d", definite.NumLetters())
		}

		if definite.MaxPossibilities() != 1 {
			t.Errorf("Expected MaxPossibilities 1, got %d", definite.MaxPossibilities())
		}
	})

	t.Run("CharsAt", func(t *testing.T) {
		cs := DefaultCharSet()
		definite.CharsAt(cs, 0) // T
		if !cs.Contains('t') {
			t.Error("CharsAt(0) should contain 't'")
		}
		if cs.Count() != 1 {
			t.Errorf("CharsAt(0) should only contain 't', count was %d", cs.Count())
		}
		cs = DefaultCharSet()
		definite.CharsAt(cs, 1) // E
		if !cs.Contains('e') {
			t.Error("CharsAt(1) should contain 'e'")
		}
		if cs.Count() != 1 {
			t.Errorf("CharsAt(1) should only contain 'e', count was %d", cs.Count())
		}
	})

	t.Run("DefinitelyBlockedAt", func(t *testing.T) {
		if definite.DefinitelyBlockedAt(0) {
			t.Error("DefinitelyBlockedAt(0) should be false for 't'")
		}
		blockedLine := ConcreteLine{Line: []rune{kBlocked, 'a'}, Words: []string{}}
		definiteBlocked := MakeDefinite(blockedLine)
		if !definiteBlocked.DefinitelyBlockedAt(0) {
			t.Error("DefinitelyBlockedAt(0) should be true for kBlocked")
		}
		if definiteBlocked.DefinitelyBlockedAt(1) {
			t.Error("DefinitelyBlockedAt(1) should be false for 'a' in {kBlocked, 'a'}")
		}
	})

	t.Run("DefiniteWords", func(t *testing.T) {
		expectedWords := []string{"test"}
		actualWords := definite.DefiniteWords()
		if diff := cmp.Diff(expectedWords, actualWords); diff != "" {
			t.Errorf("DefiniteWords mismatch (-want +got): %s", diff)
		}
	})

	t.Run("FilterAny", func(t *testing.T) {
		testCases := []struct {
			name       string
			filterSet  *CharSet
			index      int
			expectSelf bool
		}{
			{
				name: "matching char",
				filterSet: func() *CharSet {
					cs := DefaultCharSet()
					cs.Add('t')
					return cs
				}(),
				index:      0,
				expectSelf: true,
			},
			{
				name: "non-matching char",
				filterSet: func() *CharSet {
					cs := DefaultCharSet()
					cs.Add('x')
					return cs
				}(),
				index:      0,
				expectSelf: false,
			},
			{
				name: "full set",
				filterSet: func() *CharSet {
					cs := DefaultCharSet()
					for r := 'a'; r <= 'z'; r++ {
						cs.Add(r)
					}
					return cs
				}(),
				index:      0,
				expectSelf: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := definite.FilterAny(tc.filterSet, tc.index)
				if tc.expectSelf {
					if result != definite {
						t.Error("Expected FilterAny to return self")
					}
				} else {
					if !isActuallyImpossible(result) {
						t.Error("Expected FilterAny to return Impossible")
					}
				}
			})
		}
	})

	t.Run("Filter", func(t *testing.T) {
		definiteTest := MakeDefinite(ConcreteLine{Line: []rune("test"), Words: []string{"test"}})
		definiteBlocked := MakeDefinite(ConcreteLine{Line: []rune{kBlocked, 'a'}, Words: []string{}})

		testCases := []struct {
			name        string
			pl          PossibleLines // The Definite instance to test on
			filterChar  rune
			index       int
			expectSelf  bool
			expectPanic bool // Not used for Definite.Filter, but good for table structure
		}{
			{"matching char on normal", definiteTest, 't', 0, true, false},
			{"non-matching char on normal", definiteTest, 'x', 0, false, false},
			{"kBlocked on normal char", definiteTest, kBlocked, 0, false, false},
			{"kBlocked on kBlocked", definiteBlocked, kBlocked, 0, true, false},
			{"char on kBlocked", definiteBlocked, 'a', 0, false, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := tc.pl.Filter(tc.filterChar, tc.index)
				if tc.expectSelf {
					if result != tc.pl {
						t.Errorf("Expected Filter to return self, got %T, want %T", result, tc.pl)
					}
				} else {
					if !isActuallyImpossible(result) {
						t.Errorf("Expected Filter to return Impossible, got %T", result)
					}
				}
			})
		}
	})

	t.Run("RemoveWordOption", func(t *testing.T) {
		t.Run("remove definite word", func(t *testing.T) {
			if !isActuallyImpossible(definite.RemoveWordOption("test")) {
				t.Error("RemoveWordOption with the definite word should return Impossible")
			}
		})
		t.Run("remove different word", func(t *testing.T) {
			if definite.RemoveWordOption("OTHER") != definite {
				t.Error("RemoveWordOption with a different word should return self")
			}
		})
		t.Run("remove longer word", func(t *testing.T) {
			if definite.RemoveWordOption("TESTS") != definite {
				t.Error("RemoveWordOption with a longer word should return self")
			}
		})
	})

	t.Run("Iterate", func(t *testing.T) {
		count := 0
		var iteratedLine ConcreteLine
		for l := range definite.Iterate() {
			count++
			iteratedLine = l
		}
		if count != 1 {
			t.Errorf("Iterate should yield 1 item, got %d", count)
		}
		if string(iteratedLine.Line) != "test" {
			t.Errorf("Iterated line expected TEST, got %s", string(iteratedLine.Line))
		}
	})

	t.Run("FirstOrNull", func(t *testing.T) {
		first := definite.FirstOrNull()
		if first == nil || string(first.Line) != "test" {
			t.Errorf("Expected FirstOrNull to be TEST, got %v", first)
		}
	})

	t.Run("MakeChoicePanic", func(t *testing.T) {
		panicked := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicked = true
				}
			}()
			definite.MakeChoice()
		}()
		if !panicked {
			t.Error("MakeChoice on Definite should panic")
		}
	})
}

func TestBlockBefore(t *testing.T) {
	innerWord := MakeWords([]string{"hi"}, []string{})
	bb := MakeBlockBefore(innerWord)

	t.Run("Properties", func(t *testing.T) {
		if bb.NumLetters() != 3 { // kBlocked + "hi"
			t.Errorf("Expected NumLetters 3, got %d", bb.NumLetters())
		}

		if bb.MaxPossibilities() != innerWord.MaxPossibilities() {
			t.Errorf("Expected MaxPossibilities %d, got %d", innerWord.MaxPossibilities(), bb.MaxPossibilities())
		}
	})

	t.Run("CharsAt", func(t *testing.T) {
		testCases := []struct {
			name         string
			index        int
			expectChar   rune
			expectCount  int
			setupCharSet func() *CharSet // For specific setup if needed
		}{
			{"at block position", 0, kBlocked, 1, DefaultCharSet},
			{"after block", 1, 'h', 1, DefaultCharSet},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cs := tc.setupCharSet()
				bb.CharsAt(cs, tc.index)
				if !cs.Contains(tc.expectChar) {
					t.Errorf("CharsAt(%d) should contain %q, got %v", tc.index, tc.expectChar, cs)
				}
				if cs.Count() != tc.expectCount {
					t.Errorf("CharsAt(%d) expected count %d, got %d", tc.index, tc.expectCount, cs.Count())
				}
			})
		}
	})

	t.Run("DefinitelyBlockedAt", func(t *testing.T) {
		testCases := []struct {
			name          string
			index         int
			expectBlocked bool
		}{
			{"at block position", 0, true},
			{"after block", 1, false},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if bb.DefinitelyBlockedAt(tc.index) != tc.expectBlocked {
					t.Errorf("DefinitelyBlockedAt(%d) expected %t, got %t", tc.index, tc.expectBlocked, !tc.expectBlocked)
				}
			})
		}
	})

	t.Run("DefiniteWords", func(t *testing.T) {
		t.Run("with inner Definite", func(t *testing.T) {
			innerDefinite := MakeDefinite(ConcreteLine{Line: []rune("word"), Words: []string{"word"}})
			bbDefinite := MakeBlockBefore(innerDefinite)
			expectedWords := []string{"word"}
			if diff := cmp.Diff(expectedWords, bbDefinite.DefiniteWords()); diff != "" {
				t.Errorf("DefiniteWords mismatch (-want +got): %s", diff)
			}
		})
		t.Run("with inner Words", func(t *testing.T) {
			if MakeBlockBefore(MakeWords([]string{"hi", "ho"}, []string{})).DefiniteWords() != nil {
				t.Errorf("Expected DefiniteWords nil for BlockBefore(Words)")
			}
		})
	})

	t.Run("FilterAny", func(t *testing.T) {
		filterSetBlocked := DefaultCharSet()
		filterSetBlocked.Add(kBlocked)
		filterSetNotBlockedA := DefaultCharSet()
		filterSetNotBlockedA.Add('a')
		filterSetH := DefaultCharSet()
		filterSetH.Add('h')

		testCases := []struct {
			name                string
			pl                  PossibleLines
			filterSet           *CharSet
			index               int
			expectSelf          bool
			expectImpossible    bool
			expectPossibilities int64  // if not self and not impossible
			expectFirstLine     string // if not self and not impossible
		}{
			{"kBlocked at block index", bb, filterSetBlocked, 0, true, false, 0, ""},
			{"char at block index", bb, filterSetNotBlockedA, 0, false, true, 0, ""},
			{"matching char after block", bb, filterSetH, 1, false, false, 1, string([]rune{kBlocked, 'h', 'i'})},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := tc.pl.FilterAny(tc.filterSet, tc.index)
				if tc.expectSelf {
					if result != tc.pl {
						t.Error("Expected FilterAny to return self")
					}
				} else if tc.expectImpossible {
					if !isActuallyImpossible(result) {
						t.Error("Expected FilterAny to return Impossible")
					}
				} else {
					if result.MaxPossibilities() != tc.expectPossibilities {
						t.Errorf("Expected %d possibilities, got %d", tc.expectPossibilities, result.MaxPossibilities())
					}
					first := result.FirstOrNull()
					if first == nil || string(first.Line) != tc.expectFirstLine {
						t.Errorf("Expected first line %q, got %q", tc.expectFirstLine, string(first.Line))
					}
				}
			})
		}
	})

	t.Run("Filter", func(t *testing.T) {
		testCases := []struct {
			name                string
			pl                  PossibleLines
			filterChar          rune
			index               int
			expectSelf          bool
			expectImpossible    bool
			expectPossibilities int64 // if not self and not impossible
		}{
			{"kBlocked at block index", bb, kBlocked, 0, true, false, 0},
			{"char at block index", bb, 'a', 0, false, true, 0},
			{"matching char after block", bb, 'h', 1, false, false, 1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := tc.pl.Filter(tc.filterChar, tc.index)
				if tc.expectSelf {
					if result != tc.pl {
						t.Error("Expected Filter to return self")
					}
				} else if tc.expectImpossible {
					if !isActuallyImpossible(result) {
						t.Error("Expected Filter to return Impossible")
					}
				} else {
					if result.MaxPossibilities() != tc.expectPossibilities {
						t.Errorf("Expected %d possibilities, got %d", tc.expectPossibilities, result.MaxPossibilities())
					}
				}
			})
		}
	})

	t.Run("RemoveWordOption", func(t *testing.T) {
		bbWords := MakeBlockBefore(MakeWords([]string{"cat", "dog"}, []string{}))
		t.Run("remove existing inner word", func(t *testing.T) {
			removedCAT := bbWords.RemoveWordOption("cat")
			if removedCAT.MaxPossibilities() != 1 {
				t.Errorf("RemoveWordOption(\"cat\") should leave 1 possibility, got %d", removedCAT.MaxPossibilities())
			}
			expectedLine := string([]rune{kBlocked, 'd', 'o', 'g'})
			if string(removedCAT.FirstOrNull().Line) != expectedLine {
				t.Errorf("RemoveWordOption(\"cat\") expected %q, got %q", expectedLine, string(removedCAT.FirstOrNull().Line))
			}
		})
		t.Run("remove non-existent or too long word", func(t *testing.T) {
			notRemoved := bbWords.RemoveWordOption("longer")
			if notRemoved != bbWords { // Check for self, or at least equivalent content if self isn't guaranteed
				t.Error("RemoveWordOption with word not present or too long should return self or equivalent")
			}
		})
	})

	t.Run("FirstOrNull", func(t *testing.T) {
		first := bb.FirstOrNull()
		expectedLine := string([]rune{kBlocked, 'h', 'i'})
		if first == nil || string(first.Line) != expectedLine {
			t.Errorf("Expected FirstOrNull %q, got %q", expectedLine, string(first.Line))
		}
	})

	t.Run("Iterate", func(t *testing.T) {
		innerWordsIter := MakeWords([]string{"ab", "cd"}, []string{})
		bbIter := MakeBlockBefore(innerWordsIter)
		iteratedCount := 0
		expectedLines := map[string]bool{
			string([]rune{kBlocked, 'a', 'b'}): true,
			string([]rune{kBlocked, 'c', 'd'}): true,
		}
		actualLines := map[string]bool{}
		for l := range bbIter.Iterate() {
			actualLines[string(l.Line)] = true
			iteratedCount++
		}
		if !reflect.DeepEqual(actualLines, expectedLines) {
			t.Errorf("Iterate yielded unexpected lines. Got %v, want %v", actualLines, expectedLines)
		}
		if iteratedCount != len(expectedLines) {
			t.Errorf("Iterate should yield %d items, got %d", len(expectedLines), iteratedCount)
		}
	})

	t.Run("MakeChoice", func(t *testing.T) {
		innerChoice := MakeWords([]string{"one", "two"}, []string{"tri"})
		bbChoice := MakeBlockBefore(innerChoice)
		choiceStep := bbChoice.MakeChoice()

		choiceLine := choiceStep.Choice.FirstOrNull()
		remainingLine := choiceStep.Remaining.FirstOrNull()

		if choiceLine == nil || choiceLine.Line[0] != kBlocked {
			t.Errorf("MakeChoice.Choice should start with kBlocked, got %v", choiceLine)
		}
		if remainingLine == nil || remainingLine.Line[0] != kBlocked {
			t.Errorf("MakeChoice.Remaining should start with kBlocked, got %v", remainingLine)
		}
		// Further checks depend on inner MakeChoice, which should be tested for its own contract.
		// We are primarily testing the wrapping behavior of BlockBefore here.
		// Example: check that the inner part corresponds to innerChoice.MakeChoice()
		innerChoiceStep := innerChoice.MakeChoice()
		if string(choiceLine.Line[1:]) != string(innerChoiceStep.Choice.FirstOrNull().Line) {
			t.Errorf("MakeChoice.Choice inner part mismatch. Got %s, expected %s", string(choiceLine.Line[1:]), string(innerChoiceStep.Choice.FirstOrNull().Line))
		}
		if string(remainingLine.Line[1:]) != string(innerChoiceStep.Remaining.FirstOrNull().Line) {
			t.Errorf("MakeChoice.Remaining inner part mismatch. Got %s, expected %s", string(remainingLine.Line[1:]), string(innerChoiceStep.Remaining.FirstOrNull().Line))
		}
	})

	t.Run("BuildWithImpossibleInner", func(t *testing.T) {
		bbImpossible := MakeBlockBefore(MakeImpossible(2))
		// Filter is a good way to check if it behaves as Impossible
		if !isActuallyImpossible(bbImpossible.Filter('a', 1)) {
			t.Errorf("Filter on BlockBefore(Impossible) should result in Impossible, got %T", bbImpossible.Filter('a', 1))
		}
		if !isActuallyImpossible(bbImpossible) { // MakeBlockBefore(Impossible) should itself be Impossible
			t.Errorf("MakeBlockBefore(Impossible) should be Impossible, got %T", bbImpossible)
		}
	})
}

func TestBlockAfter(t *testing.T) {
	innerWord := MakeWords([]string{"hi"}, []string{})
	ba := MakeBlockAfter(innerWord)
	innerNumLetters := innerWord.NumLetters()

	t.Run("Properties", func(t *testing.T) {
		if ba.NumLetters() != innerNumLetters+1 { // "hi" + kBlocked
			t.Errorf("Expected NumLetters %d, got %d", innerNumLetters+1, ba.NumLetters())
		}

		if ba.MaxPossibilities() != innerWord.MaxPossibilities() {
			t.Errorf("Expected MaxPossibilities %d, got %d", innerWord.MaxPossibilities(), ba.MaxPossibilities())
		}
	})

	t.Run("CharsAt", func(t *testing.T) {
		testCases := []struct {
			name         string
			index        int
			expectChar   rune
			expectCount  int
			setupCharSet func() *CharSet
		}{
			{"at first char of inner", 0, 'h', 1, DefaultCharSet},
			{"at block position", innerNumLetters, kBlocked, 1, func() *CharSet {
				// Add kBlocked to make capacity checks work as expected in CharsAt, if applicable
				// For this specific test, we want to ensure only kBlocked is present after CharsAt.
				cs := DefaultCharSet()
				// cs.Add(kBlocked) // Pre-adding might obscure the actual behavior of CharsAt.
				// Let CharsAt add it. We will check it contains kBlocked and count is 1.
				return cs
			}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cs := tc.setupCharSet()
				ba.CharsAt(cs, tc.index)
				if !cs.Contains(tc.expectChar) {
					t.Errorf("CharsAt(%d) should contain %q, got %v (all chars: %v)", tc.index, tc.expectChar, cs.Contains(tc.expectChar), cs)
				}
				if cs.Count() != tc.expectCount {
					t.Errorf("CharsAt(%d) expected count %d, got %d (all chars: %v)", tc.index, tc.expectCount, cs.Count(), cs)
				}
			})
		}
	})

	t.Run("DefinitelyBlockedAt", func(t *testing.T) {
		testCases := []struct {
			name          string
			index         int
			expectBlocked bool
		}{
			{"at block position", innerNumLetters, true},
			{"before block", 0, false},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if ba.DefinitelyBlockedAt(tc.index) != tc.expectBlocked {
					t.Errorf("DefinitelyBlockedAt(%d) expected %t, got %t", tc.index, tc.expectBlocked, !tc.expectBlocked)
				}
			})
		}
	})

	t.Run("DefiniteWords", func(t *testing.T) {
		t.Run("with inner Definite", func(t *testing.T) {
			innerDefinite := MakeDefinite(ConcreteLine{Line: []rune("word"), Words: []string{"word"}})
			baDefinite := MakeBlockAfter(innerDefinite)
			expectedWords := []string{"word"}
			if diff := cmp.Diff(expectedWords, baDefinite.DefiniteWords()); diff != "" {
				t.Errorf("DefiniteWords mismatch (-want +got): %s", diff)
			}
		})
		t.Run("with inner Words", func(t *testing.T) {
			if MakeBlockAfter(MakeWords([]string{"hi", "ho"}, []string{})).DefiniteWords() != nil {
				t.Errorf("Expected DefiniteWords nil for BlockAfter(Words)")
			}
		})
	})

	t.Run("FilterAny", func(t *testing.T) {
		filterSetBlocked := DefaultCharSet()
		filterSetBlocked.Add(kBlocked)
		filterSetNotBlockedA := DefaultCharSet()
		filterSetNotBlockedA.Add('a')
		filterSetH := DefaultCharSet()
		filterSetH.Add('h')

		testCases := []struct {
			name                string
			pl                  PossibleLines
			filterSet           *CharSet
			index               int
			expectSelf          bool
			expectImpossible    bool
			expectPossibilities int64
			expectFirstLine     string
		}{
			{"kBlocked at block index", ba, filterSetBlocked, innerNumLetters, true, false, 0, ""},
			{"char at block index", ba, filterSetNotBlockedA, innerNumLetters, false, true, 0, ""},
			{"matching char before block", ba, filterSetH, 0, false, false, 1, string([]rune{'h', 'i', kBlocked})},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := tc.pl.FilterAny(tc.filterSet, tc.index)
				if tc.expectSelf {
					if result != tc.pl {
						t.Error("Expected FilterAny to return self")
					}
				} else if tc.expectImpossible {
					if !isActuallyImpossible(result) {
						t.Error("Expected FilterAny to return Impossible")
					}
				} else {
					if result.MaxPossibilities() != tc.expectPossibilities {
						t.Errorf("Expected %d possibilities, got %d", tc.expectPossibilities, result.MaxPossibilities())
					}
					first := result.FirstOrNull()
					if first == nil || string(first.Line) != tc.expectFirstLine {
						var gotLine string
						if first != nil {
							gotLine = string(first.Line)
						}
						t.Errorf("Expected first line %q, got %q", tc.expectFirstLine, gotLine)
					}
				}
			})
		}
	})

	t.Run("Filter", func(t *testing.T) {
		testCases := []struct {
			name                string
			pl                  PossibleLines
			filterChar          rune
			index               int
			expectSelf          bool
			expectImpossible    bool
			expectPossibilities int64
		}{
			{"kBlocked at block index", ba, kBlocked, innerNumLetters, true, false, 0},
			{"char at block index", ba, 'a', innerNumLetters, false, true, 0},
			{"matching char before block", ba, 'h', 0, false, false, 1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := tc.pl.Filter(tc.filterChar, tc.index)
				if tc.expectSelf {
					if result != tc.pl {
						t.Error("Expected Filter to return self")
					}
				} else if tc.expectImpossible {
					if !isActuallyImpossible(result) {
						t.Error("Expected Filter to return Impossible")
					}
				} else {
					if result.MaxPossibilities() != tc.expectPossibilities {
						t.Errorf("Expected %d possibilities, got %d", tc.expectPossibilities, result.MaxPossibilities())
					}
				}
			})
		}
	})

	t.Run("RemoveWordOption", func(t *testing.T) {
		baWords := MakeBlockAfter(MakeWords([]string{"cat", "dog"}, []string{}))
		t.Run("remove existing inner word", func(t *testing.T) {
			removedCAT := baWords.RemoveWordOption("cat")
			if removedCAT.MaxPossibilities() != 1 {
				t.Errorf("RemoveWordOption(\"cat\") should leave 1 possibility, got %d", removedCAT.MaxPossibilities())
			}
			expectedLine := string([]rune{'d', 'o', 'g', kBlocked})
			if string(removedCAT.FirstOrNull().Line) != expectedLine {
				t.Errorf("RemoveWordOption(\"cat\") expected %q, got %q", expectedLine, string(removedCAT.FirstOrNull().Line))
			}
		})
		t.Run("remove non-existent or too long word", func(t *testing.T) {
			notRemoved := baWords.RemoveWordOption("longer")
			if notRemoved != baWords {
				t.Error("RemoveWordOption with word not present or too long should return self or equivalent")
			}
		})
	})

	t.Run("FirstOrNull", func(t *testing.T) {
		first := ba.FirstOrNull()
		expectedLine := string([]rune{'h', 'i', kBlocked})
		if first == nil || string(first.Line) != expectedLine {
			t.Errorf("Expected FirstOrNull %q, got %q", expectedLine, string(first.Line))
		}
	})

	t.Run("Iterate", func(t *testing.T) {
		innerWordsIter := MakeWords([]string{"ab", "cd"}, []string{})
		baIter := MakeBlockAfter(innerWordsIter)
		iteratedCount := 0
		expectedLines := map[string]bool{
			string([]rune{'a', 'b', kBlocked}): true,
			string([]rune{'c', 'd', kBlocked}): true,
		}
		actualLines := map[string]bool{}
		for l := range baIter.Iterate() {
			actualLines[string(l.Line)] = true
			iteratedCount++
		}
		if !reflect.DeepEqual(actualLines, expectedLines) {
			t.Errorf("Iterate yielded unexpected lines. Got %v, want %v", actualLines, expectedLines)
		}
		if iteratedCount != len(expectedLines) {
			t.Errorf("Iterate should yield %d items, got %d", len(expectedLines), iteratedCount)
		}
	})

	t.Run("MakeChoice", func(t *testing.T) {
		innerChoice := MakeWords([]string{"one", "two"}, []string{"tri"})
		baChoice := MakeBlockAfter(innerChoice)
		choiceStep := baChoice.MakeChoice()

		choiceLineCL := choiceStep.Choice.FirstOrNull()
		remainingLineCL := choiceStep.Remaining.FirstOrNull()

		if choiceLineCL == nil || choiceLineCL.Line[len(choiceLineCL.Line)-1] != kBlocked {
			t.Errorf("MakeChoice.Choice should end with kBlocked, got %v", choiceLineCL)
		}
		if remainingLineCL == nil || remainingLineCL.Line[len(remainingLineCL.Line)-1] != kBlocked {
			t.Errorf("MakeChoice.Remaining should end with kBlocked, got %v", remainingLineCL)
		}

		innerChoiceStep := innerChoice.MakeChoice()
		innerChoiceFirst := innerChoiceStep.Choice.FirstOrNull()
		innerRemainingFirst := innerChoiceStep.Remaining.FirstOrNull()

		if innerChoiceFirst != nil && string(choiceLineCL.Line[:len(choiceLineCL.Line)-1]) != string(innerChoiceFirst.Line) {
			t.Errorf("MakeChoice.Choice inner part mismatch. Got %s, expected %s", string(choiceLineCL.Line[:len(choiceLineCL.Line)-1]), string(innerChoiceFirst.Line))
		}
		if innerRemainingFirst != nil && string(remainingLineCL.Line[:len(remainingLineCL.Line)-1]) != string(innerRemainingFirst.Line) {
			t.Errorf("MakeChoice.Remaining inner part mismatch. Got %s, expected %s", string(remainingLineCL.Line[:len(remainingLineCL.Line)-1]), string(innerRemainingFirst.Line))
		}
	})

	t.Run("BuildWithImpossibleInner", func(t *testing.T) {
		baImpossible := MakeBlockAfter(MakeImpossible(2))
		if !isActuallyImpossible(baImpossible.Filter('a', 0)) { // Index 0 is before the block
			t.Errorf("Filter on BlockAfter(Impossible) should result in Impossible, got %T", baImpossible.Filter('a', 0))
		}
		if !isActuallyImpossible(baImpossible) { // MakeBlockAfter(Impossible) should itself be Impossible
			t.Errorf("MakeBlockAfter(Impossible) should be Impossible, got %T", baImpossible)
		}
	})
}

func TestBlockBetween(t *testing.T) {
	firstInner := MakeWords([]string{"ab"}, []string{})
	secondInner := MakeWords([]string{"cd"}, []string{})
	bb := MakeBlockBetween(firstInner, secondInner)

	firstLen := firstInner.NumLetters()
	// secondLen := secondInner.NumLetters() // Not strictly needed for indices if using firstLen as blockStart
	expectedNumLetters := 1 + firstInner.NumLetters() + secondInner.NumLetters()

	t.Run("Properties", func(t *testing.T) {
		if bb.NumLetters() != expectedNumLetters {
			t.Errorf("Expected NumLetters %d, got %d", expectedNumLetters, bb.NumLetters())
		}

		expectedMaxPossibilities := firstInner.MaxPossibilities() * secondInner.MaxPossibilities()
		if bb.MaxPossibilities() != expectedMaxPossibilities {
			t.Errorf("Expected MaxPossibilities %d, got %d", expectedMaxPossibilities, bb.MaxPossibilities())
		}
	})

	t.Run("CharsAt", func(t *testing.T) {
		testCases := []struct {
			name        string
			index       int
			expectChar  rune
			expectCount int
		}{
			{"in first part", 0, 'a', 1},
			{"at block position", firstLen, kBlocked, 1},
			{"in second part", firstLen + 1, 'c', 1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cs := DefaultCharSet()
				// If testing kBlocked, some CharsAt implementations might need the set to have capacity or kBlocked already.
				// For BlockBetween, it seems to handle it directly.
				bb.CharsAt(cs, tc.index)
				if !cs.Contains(tc.expectChar) {
					t.Errorf("CharsAt(%d) should contain %q, got %v (all: %v)", tc.index, tc.expectChar, cs.Contains(tc.expectChar), cs)
				}
				if cs.Count() != tc.expectCount {
					t.Errorf("CharsAt(%d) expected count %d, got %d (all: %v)", tc.index, tc.expectCount, cs.Count(), cs)
				}
			})
		}
	})

	t.Run("DefinitelyBlockedAt", func(t *testing.T) {
		testCases := []struct {
			name          string
			index         int
			expectBlocked bool
		}{
			{"in first part", 0, false},
			{"at block position", firstLen, true},
			{"in second part", firstLen + 1, false},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if bb.DefinitelyBlockedAt(tc.index) != tc.expectBlocked {
					t.Errorf("DefinitelyBlockedAt(%d) expected %t, got %t", tc.index, tc.expectBlocked, !tc.expectBlocked)
				}
			})
		}
	})

	t.Run("DefiniteWords", func(t *testing.T) {
		defFirst := MakeDefinite(ConcreteLine{Line: []rune("wa"), Words: []string{"wa"}})
		defSecond := MakeDefinite(ConcreteLine{Line: []rune("wb"), Words: []string{"wb"}})
		bbDefinite := MakeBlockBetween(defFirst, defSecond)
		expectedWords := []string{"wa", "wb"}
		if diff := cmp.Diff(expectedWords, bbDefinite.DefiniteWords()); diff != "" {
			t.Errorf("DefiniteWords mismatch (-want +got): %s", diff)
		}
	})

	t.Run("FilterAny", func(t *testing.T) {
		filterSetBlocked := DefaultCharSet()
		filterSetBlocked.Add(kBlocked)
		filterSetA := DefaultCharSet()
		filterSetA.Add('a')

		testCases := []struct {
			name                string
			pl                  PossibleLines
			filterSet           *CharSet
			index               int
			expectSelf          bool
			expectImpossible    bool
			expectPossibilities int64
		}{
			{"kBlocked at block position", bb, filterSetBlocked, firstLen, true, false, 0},
			{"char at block position", bb, filterSetA, firstLen, false, true, 0},
			{"matching char in first part", bb, filterSetA, 0, false, false, secondInner.MaxPossibilities()},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := tc.pl.FilterAny(tc.filterSet, tc.index)
				if tc.expectSelf {
					if result != tc.pl {
						t.Error("Expected FilterAny to return self")
					}
				} else if tc.expectImpossible {
					if !isActuallyImpossible(result) {
						t.Error("Expected FilterAny to return Impossible")
					}
				} else {
					if result.MaxPossibilities() != tc.expectPossibilities {
						t.Errorf("Expected %d possibilities, got %d", tc.expectPossibilities, result.MaxPossibilities())
					}
					// Further checks on content if needed, e.g., FirstOrNull()
				}
			})
		}
	})

	t.Run("Filter", func(t *testing.T) {
		testCases := []struct {
			name                string
			pl                  PossibleLines
			filterChar          rune
			index               int
			expectSelf          bool
			expectImpossible    bool
			expectPossibilities int64
		}{
			{"kBlocked at block position", bb, kBlocked, firstLen, true, false, 0},
			{"char at block position", bb, 'x', firstLen, false, true, 0},
			{"matching char in first part", bb, 'a', 0, false, false, secondInner.MaxPossibilities()},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := tc.pl.Filter(tc.filterChar, tc.index)
				if tc.expectSelf {
					if result != tc.pl {
						t.Error("Expected Filter to return self")
					}
				} else if tc.expectImpossible {
					if !isActuallyImpossible(result) {
						t.Error("Expected Filter to return Impossible")
					}
				} else {
					if result.MaxPossibilities() != tc.expectPossibilities {
						t.Errorf("Expected %d possibilities, got %d", tc.expectPossibilities, result.MaxPossibilities())
					}
				}
			})
		}
	})

	t.Run("RemoveWordOption", func(t *testing.T) {
		bbComplex := MakeBlockBetween(MakeWords([]string{"one", "two"}, []string{}), MakeWords([]string{"three"}, []string{}))

		t.Run("remove from first part", func(t *testing.T) {
			removedONE := bbComplex.RemoveWordOption("one")
			if removedONE.MaxPossibilities() != 1*1 { // (two) # (three)
				t.Errorf("RemoveWordOption(\"one\") expected 1 possibility, got %d", removedONE.MaxPossibilities())
			}
			expectedLine := "two" + string(kBlocked) + "three"
			if string(removedONE.FirstOrNull().Line) != expectedLine {
				t.Errorf("RemoveWordOption(\"one\") line error, got %q, want %q", string(removedONE.FirstOrNull().Line), expectedLine)
			}
		})

		t.Run("remove from second part making it impossible", func(t *testing.T) {
			removedTHREE := bbComplex.RemoveWordOption("three") // (one or two) # ()
			if !isActuallyImpossible(removedTHREE) {            // second part becomes impossible, so whole thing is
				t.Errorf("RemoveWordOption(\"three\") should make it impossible, got %T with %d poss", removedTHREE, removedTHREE.MaxPossibilities())
			}
		})

		t.Run("remove non-existent word", func(t *testing.T) {
			notRemoved := bbComplex.RemoveWordOption("four")
			if notRemoved.MaxPossibilities() != bbComplex.MaxPossibilities() || string(notRemoved.FirstOrNull().Line) != string(bbComplex.FirstOrNull().Line) {
				// Depending on implementation, it might return self or a new equivalent object.
				// Checking MaxPossibilities and FirstOrNull content is a robust way.
				t.Error("RemoveWordOption with non-existent word should return self or equivalent")
			}
		})
	})

	t.Run("FirstOrNull", func(t *testing.T) {
		t.Run("standard case", func(t *testing.T) {
			first := bb.FirstOrNull()
			expectedLineStr := "ab" + string(kBlocked) + "cd"
			if first == nil || string(first.Line) != expectedLineStr {
				t.Errorf("Expected FirstOrNull %q, got %q", expectedLineStr, string(first.Line))
			}
		})
		t.Run("with impossible part", func(t *testing.T) {
			bbWithImpossible := MakeBlockBetween(firstInner, MakeImpossible(2))
			if bbWithImpossible.FirstOrNull() != nil {
				t.Errorf("FirstOrNull with an impossible part should be nil, got %v", bbWithImpossible.FirstOrNull())
			}
		})
	})

	t.Run("Iterate", func(t *testing.T) {
		iterFirst := MakeWords([]string{"X", "Y"}, []string{})
		iterSecond := MakeWords([]string{"Z"}, []string{})
		bbIter := MakeBlockBetween(iterFirst, iterSecond)
		iteratedCount := 0
		expectedIterLines := map[string]bool{
			"X" + string(kBlocked) + "Z": true,
			"Y" + string(kBlocked) + "Z": true,
		}
		actualLines := map[string]bool{}
		for l := range bbIter.Iterate() {
			actualLines[string(l.Line)] = true
			iteratedCount++
		}
		if !reflect.DeepEqual(actualLines, expectedIterLines) {
			t.Errorf("Iterate yielded unexpected lines. Got %v, want %v", actualLines, expectedIterLines)
		}
		if iteratedCount != len(expectedIterLines) {
			t.Errorf("Iterate should yield %d items, got %d", len(expectedIterLines), iteratedCount)
		}
	})

	t.Run("MakeChoice", func(t *testing.T) {
		// Case 1: First part has choices
		t.Run("first part choices", func(t *testing.T) {
			choiceFirstInner := MakeWords([]string{"F1", "F2"}, []string{})
			bbChoice1 := MakeBlockBetween(choiceFirstInner, secondInner) // secondInner is ("cd")
			cs1 := bbChoice1.MakeChoice()
			if cs1.Choice.FirstOrNull() == nil || string(cs1.Choice.FirstOrNull().Line) != "F1"+string(kBlocked)+"cd" {
				t.Errorf("MakeChoice case 1 Choice error, got %v", cs1.Choice.FirstOrNull())
			}
			if cs1.Remaining.FirstOrNull() == nil || string(cs1.Remaining.FirstOrNull().Line) != "F2"+string(kBlocked)+"cd" {
				t.Errorf("MakeChoice case 1 Remaining error, got %v", cs1.Remaining.FirstOrNull())
			}
		})

		// Case 2: First part no choice, second part has choices
		t.Run("second part choices", func(t *testing.T) {
			choiceSecondInner := MakeWords([]string{"S1", "S2"}, []string{})
			bbChoice2 := MakeBlockBetween(firstInner, choiceSecondInner) // firstInner is ("ab")
			cs2 := bbChoice2.MakeChoice()
			if cs2.Choice.FirstOrNull() == nil || string(cs2.Choice.FirstOrNull().Line) != "ab"+string(kBlocked)+"S1" {
				t.Errorf("MakeChoice case 2 Choice error, got %v", cs2.Choice.FirstOrNull())
			}
			if cs2.Remaining.FirstOrNull() == nil || string(cs2.Remaining.FirstOrNull().Line) != "ab"+string(kBlocked)+"S2" {
				t.Errorf("MakeChoice case 2 Remaining error, got %v", cs2.Remaining.FirstOrNull())
			}
		})
	})

	t.Run("BuildWithImpossible", func(t *testing.T) {
		t.Run("first part impossible", func(t *testing.T) {
			bbBuild1 := MakeBlockBetween(MakeImpossible(1), secondInner)
			if !isActuallyImpossible(bbBuild1) {
				t.Errorf("MakeBlockBetween(Impossible, X) should be Impossible, got %T", bbBuild1)
			}
		})
		t.Run("second part impossible", func(t *testing.T) {
			bbBuild2 := MakeBlockBetween(firstInner, MakeImpossible(1))
			if !isActuallyImpossible(bbBuild2) {
				t.Errorf("MakeBlockBetween(X, Impossible) should be Impossible, got %T", bbBuild2)
			}
		})
	})

	t.Run("BuildReturnsSelf", func(t *testing.T) {
		if blockBetweenInstance, ok := bb.(*BlockBetween); ok {
			builtSelf := blockBetweenInstance.build(firstInner, secondInner)
			if builtSelf != bb {
				t.Error("build(original, original) should return self for BlockBetween instance")
			}
		} else {
			t.Errorf("Expected bb to be *BlockBetween for build self test, got %T", bb)
		}
	})
}

func TestCompound(t *testing.T) {
	t.Run("MakeCompound", func(t *testing.T) {
		t.Run("empty possibilities", func(t *testing.T) {
			impossible := MakeCompound([]PossibleLines{})
			if !isActuallyImpossible(impossible) {
				t.Errorf("MakeCompound with empty slice should return Impossible, got %T", impossible)
			}
		})

		t.Run("single possibility returns self", func(t *testing.T) {
			definiteSingle := MakeDefinite(ConcreteLine{Line: []rune("S"), Words: []string{"S"}})
			single := MakeCompound([]PossibleLines{definiteSingle})
			if single != definiteSingle {
				t.Errorf("MakeCompound with single item should return the item itself, got %T want %T", single, definiteSingle)
			}
		})

		t.Run("flattening and impossible removal", func(t *testing.T) {
			w1 := MakeWords([]string{"wa"}, []string{})
			w2 := MakeWords([]string{"wb"}, []string{})
			w3 := MakeWords([]string{"wc"}, []string{})
			// nestedCompound will resolve to w2, as MakeCompound with {w2, Impossible} becomes w2.
			nestedCompound := MakeCompound([]PossibleLines{w2, MakeImpossible(2)})
			flat := MakeCompound([]PossibleLines{w1, MakeImpossible(1), nestedCompound, w3})

			// Expecting flat to be a *Compound containing [w1, w2, w3]
			if compoundResult, ok := flat.(*Compound); ok {
				if len(compoundResult.possibilities) != 3 { // w1, w2 (from resolved nestedCompound), w3
					t.Errorf("MakeCompound expected to flatten to 3 possibilities, got %d", len(compoundResult.possibilities))
				} else {
					if compoundResult.possibilities[0] != w1 || compoundResult.possibilities[1] != w2 || compoundResult.possibilities[2] != w3 {
						t.Error("MakeCompound did not flatten correctly or preserve order")
					}
				}
			} else {
				t.Errorf("Expected MakeCompound to return *Compound, got %T", flat)
			}
		})
	})

	// Setup a basic compound for further tests
	p1 := MakeWords([]string{"ab"}, []string{})
	p2 := MakeWords([]string{"ac"}, []string{})
	// This should result in a *Compound type because p1 and p2 are distinct non-Impossible PossibleLines.
	compoundInstance := MakeCompound([]PossibleLines{p1, p2})
	compound, ok := compoundInstance.(*Compound)
	if !ok {
		t.Fatalf("MakeCompound for basic setup did not return *Compound, got %T. Skipping remaining Compound tests.", compoundInstance)
		return
	}

	t.Run("Properties", func(t *testing.T) {
		if compound.NumLetters() != 2 {
			t.Errorf("Expected NumLetters 2, got %d", compound.NumLetters())
		}

		if compound.MaxPossibilities() != p1.MaxPossibilities()+p2.MaxPossibilities() { // 1 + 1 = 2
			t.Errorf("Expected MaxPossibilities %d, got %d", p1.MaxPossibilities()+p2.MaxPossibilities(), compound.MaxPossibilities())
		}
	})

	t.Run("CharsAt", func(t *testing.T) {
		testCases := []struct {
			name        string
			index       int
			expectChars []rune
			expectCount int
		}{
			{"at index 0 (common char)", 0, []rune{'a'}, 1},
			{"at index 1 (different chars)", 1, []rune{'b', 'c'}, 2},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cs := DefaultCharSet()
				compound.CharsAt(cs, tc.index)
				for _, char := range tc.expectChars {
					if !cs.Contains(char) {
						t.Errorf("CharsAt(%d) should contain %q, but does not. Set: %v", tc.index, char, cs)
					}
				}
				if cs.Count() != tc.expectCount {
					t.Errorf("CharsAt(%d) expected count %d, got %d. Set: %v", tc.index, tc.expectCount, cs.Count(), cs)
				}
			})
		}
	})

	t.Run("DefinitelyBlockedAt", func(t *testing.T) {
		pBlock1 := MakeBlockBefore(MakeWords([]string{"X"}, []string{}))
		pBlock2 := MakeBlockBefore(MakeWords([]string{"Y"}, []string{}))
		pNonBlock := MakeWords([]string{"ab"}, []string{})

		testCases := []struct {
			name          string
			pls           []PossibleLines
			index         int
			expectBlocked bool
		}{
			{"all inner blocked", []PossibleLines{pBlock1, pBlock2}, 0, true},
			{"mixed blocked/non-blocked", []PossibleLines{pBlock1, pNonBlock}, 0, false},
			{"all inner not blocked", []PossibleLines{pNonBlock, p1}, 0, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				compoundMixed := MakeCompound(tc.pls)
				if compoundMixed.DefinitelyBlockedAt(tc.index) != tc.expectBlocked {
					t.Errorf("DefinitelyBlockedAt(%d) for %s case: expected %t, got %t", tc.index, tc.name, tc.expectBlocked, !tc.expectBlocked)
				}
			})
		}
	})

	t.Run("DefiniteWords", func(t *testing.T) {
		t.SkipNow()
		if compound.DefiniteWords() != nil {
			t.Errorf("Expected DefiniteWords nil for a compound with multiple different words, got %v", compound.DefiniteWords())
		}
		// Case where MakeCompound might simplify to a Definite if all constituents are the same word
		sameWord1 := MakeWords([]string{"same"}, []string{})
		sameWord2 := MakeDefinite(ConcreteLine{Line: []rune("same"), Words: []string{"same"}})
		compoundSame := MakeCompound([]PossibleLines{sameWord1, sameWord2})
		if compoundSameDefinite, ok := compoundSame.(*Definite); ok {
			expected := []string{"same"}
			if diff := cmp.Diff(expected, compoundSameDefinite.DefiniteWords()); diff != "" {
				t.Errorf("DefiniteWords for compound of same words mismatch (-want +got): %s", diff)
			}
		} else {
			// This might also be a Compound that happens to have DefiniteWords if logic changes
			// For now, MakeCompound simplifies this to *Definite.
			t.Logf("MakeCompound with all same words returned %T, checking DefiniteWords on it", compoundSame)
			if compoundSame.DefiniteWords() == nil || !slices.Equal(compoundSame.DefiniteWords(), []string{"same"}) {
				t.Errorf("Expected DefiniteWords to be [\"same\"] for compound of same words, got %v", compoundSame.DefiniteWords())
			}
		}
	})

	t.Run("FilterAny", func(t *testing.T) {
		filterSetA := DefaultCharSet()
		filterSetA.Add('a')
		filterSetB := DefaultCharSet()
		filterSetB.Add('b')
		filterSetX := DefaultCharSet()
		filterSetX.Add('x')

		testCases := []struct {
			name                string
			filterSet           *CharSet
			index               int
			expectImpossible    bool
			expectPossibilities int64  // if not impossible
			expectFirstLine     string // if not impossible and has possibilities
		}{
			{"filter 'a' at 0 (matches all)", filterSetA, 0, false, 2, "ab"},
			{"filter 'b' at 1 (matches one)", filterSetB, 1, false, 1, "ab"},
			{"filter 'x' at 0 (matches none)", filterSetX, 0, true, 0, ""},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := compound.FilterAny(tc.filterSet, tc.index)
				if tc.expectImpossible {
					if !isActuallyImpossible(result) {
						t.Errorf("Expected FilterAny to return Impossible, got %T", result)
					}
				} else {
					if result.MaxPossibilities() != tc.expectPossibilities {
						t.Errorf("Expected %d possibilities, got %d", tc.expectPossibilities, result.MaxPossibilities())
					}
					if result.MaxPossibilities() > 0 && (result.FirstOrNull() == nil || string(result.FirstOrNull().Line) != tc.expectFirstLine) {
						var gotLine string
						if result.FirstOrNull() != nil {
							gotLine = string(result.FirstOrNull().Line)
						}
						t.Errorf("Expected first line %q, got %q", tc.expectFirstLine, gotLine)
					}
				}
			})
		}
	})

	t.Run("Filter", func(t *testing.T) {
		testCases := []struct {
			name                string
			filterChar          rune
			index               int
			expectImpossible    bool
			expectPossibilities int64
			expectFirstLine     string
		}{
			{"filter 'a' at 0 (matches all)", 'a', 0, false, 2, "ab"},
			{"filter 'b' at 1 (matches one)", 'b', 1, false, 1, "ab"},
			{"filter 'z' at 0 (matches none)", 'z', 0, true, 0, ""},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := compound.Filter(tc.filterChar, tc.index)
				if tc.expectImpossible {
					if !isActuallyImpossible(result) {
						t.Errorf("Expected Filter to return Impossible, got %T", result)
					}
				} else {
					if result.MaxPossibilities() != tc.expectPossibilities {
						t.Errorf("Expected %d possibilities, got %d", tc.expectPossibilities, result.MaxPossibilities())
					}
				}
			})
		}
	})

	t.Run("RemoveWordOption", func(t *testing.T) {
		compoundForRemove := MakeCompound([]PossibleLines{
			MakeWords([]string{"one"}, []string{}),
			MakeWords([]string{"two"}, []string{}),
			MakeWords([]string{"one"}, []string{}), // Duplicate to test removal from all
		}).(*Compound) // This setup should remain a Compound

		t.Run("remove existing word", func(t *testing.T) {
			removedONE := compoundForRemove.RemoveWordOption("one")
			if removedONE.MaxPossibilities() != 1 { // Only TWO should remain
				t.Errorf("RemoveWordOption(\"one\") expected 1 possibility, got %d", removedONE.MaxPossibilities())
			}
			if string(removedONE.FirstOrNull().Line) != "two" {
				t.Errorf("RemoveWordOption(\"one\") expected line TWO, got %s", string(removedONE.FirstOrNull().Line))
			}
		})
		t.Run("remove non-existent word", func(t *testing.T) {
			removedTHREE := compoundForRemove.RemoveWordOption("tri")
			if removedTHREE.MaxPossibilities() != compoundForRemove.MaxPossibilities() {
				t.Errorf("RemoveWordOption for non-existent word changed possibilities from %d to %d", compoundForRemove.MaxPossibilities(), removedTHREE.MaxPossibilities())
			}
			// Also check if content is equivalent if not self
			if !reflect.DeepEqual(collectLines(removedTHREE), collectLines(compoundForRemove)) {
				t.Errorf("RemoveWordOption for non-existent word changed content. Got %v, want %v", collectLines(removedTHREE), collectLines(compoundForRemove))
			}
		})
	})

	t.Run("FirstOrNull", func(t *testing.T) {
		t.Run("standard compound", func(t *testing.T) {
			if compound.FirstOrNull() == nil || string(compound.FirstOrNull().Line) != "ab" { // p1 is{"ab"}, p2 is {"ac"}
				t.Errorf("Expected FirstOrNull AB, got %v", compound.FirstOrNull())
			}
		})
		t.Run("compound with impossible first element", func(t *testing.T) {
			compoundWithImpossibleFirst := MakeCompound([]PossibleLines{MakeImpossible(2), p1}) // p1 is {"ab"}
			// MakeCompound should simplify this to just p1.
			if compoundWithImpossibleFirst.FirstOrNull() == nil || string(compoundWithImpossibleFirst.FirstOrNull().Line) != "ab" {
				t.Errorf("FirstOrNull should skip impossible and find AB, got %v. Result type: %T", compoundWithImpossibleFirst.FirstOrNull(), compoundWithImpossibleFirst)
			}
		})
	})

	t.Run("Iterate", func(t *testing.T) {
		iterP1 := MakeWords([]string{"X1", "X2"}, []string{})
		iterP2 := MakeWords([]string{"Y1"}, []string{})
		compoundIter := MakeCompound([]PossibleLines{iterP1, iterP2}).(*Compound)

		actualLines := collectLines(compoundIter)
		expectedIter := []string{"X1", "X2", "Y1"} // Order depends on Compound.possibilities and inner iter

		mapActual := make(map[string]int)
		for _, l := range actualLines {
			mapActual[l]++
		}
		mapExpected := make(map[string]int)
		for _, l := range expectedIter {
			mapExpected[l]++
		}
		if !reflect.DeepEqual(mapActual, mapExpected) {
			t.Errorf("Iterate lines mismatch. Expected counts %v, got counts %v (Actual lines: %v, Expected lines: %v)", mapExpected, mapActual, actualLines, expectedIter)
		}
	})

	t.Run("MakeChoice", func(t *testing.T) {
		cpChoice1 := MakeWords([]string{"C1"}, []string{})
		cpChoice2 := MakeWords([]string{"C2"}, []string{})
		cpChoice3 := MakeWords([]string{"C3"}, []string{})
		cpChoice4 := MakeWords([]string{"C4"}, []string{})
		// This setup will result in a *Compound
		compoundToChooseInstance := MakeCompound([]PossibleLines{cpChoice1, cpChoice2, cpChoice3, cpChoice4})
		compoundToChoose, ok := compoundToChooseInstance.(*Compound)
		if !ok {
			t.Fatalf("MakeCompound for MakeChoice test did not return *Compound, got %T", compoundToChooseInstance)
		}

		choiceStep := compoundToChoose.MakeChoice()
		// Default split is half, so 2 and 2
		if choiceStep.Choice.MaxPossibilities() != 2 || choiceStep.Remaining.MaxPossibilities() != 2 {
			t.Errorf("MakeChoice split error: Choice %d, Remaining %d. Expected 2, 2",
				choiceStep.Choice.MaxPossibilities(), choiceStep.Remaining.MaxPossibilities())
		}
		// Check content of choices (relies on order of possibilities in Compound and default split logic)
		choiceActualLines := collectLines(choiceStep.Choice)
		choiceExpectedLines := []string{"C1", "C2"}
		if diff := cmp.Diff(choiceExpectedLines, choiceActualLines); diff != "" {
			t.Errorf("MakeChoice.Choice lines: -want +got %s", diff)
		}

		remainingActualLines := collectLines(choiceStep.Remaining)
		remainingExpectedLines := []string{"C3", "C4"}
		if diff := cmp.Diff(remainingExpectedLines, remainingActualLines); diff != "" {
			t.Errorf("MakeChoice.Remaining lines: -want +got %s", diff)
		}

		// Test panic conditions (MakeCompound usually simplifies, making direct panic hard for Compound.MakeChoice)
		// A Compound type directly should not have <=1 possibility or <=1 item in list due to MakeCompound logic.
		// If Filter reduces a Compound to 1 possibility, it returns that single possibility, not a Compound.
		// The panic `if len(c.possibilities) <= 1` in Compound.MakeChoice is the main one to consider for direct Compound calls.
		// MakeCompound ensures len(possibilities) > 1 for a *Compound result.
		// If Filter reduces a Compound to 1 possibility, it returns that single possibility, not a Compound.
	})
}

// Helper function to collect all lines from a PossibleLines iterator
func collectLines(pl PossibleLines) []string {
	if pl == nil || isActuallyImpossible(pl) {
		return []string{}
	}
	lines := []string{}
	for item := range pl.Iterate() {
		lines = append(lines, string(item.Line))
	}
	return lines
}
