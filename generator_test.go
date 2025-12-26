package xwgen

import (
	"bufio"
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"
	"time"
)

func loadWords(t testing.TB) []string {
	file, err := os.Open("testdata/words.txt")
	if err != nil {
		t.Fatalf("failed to open words file: %v", err)
	}
	defer file.Close()

	var words []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		words = append(words, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("failed to scan words file: %v", err)
	}
	return words
}

func TestPossibleGrids_5x5(t *testing.T) {
	words := loadWords(t)
	// Use a fixed seed for reproducibility.
	rng := rand.New(rand.NewPCG(42, 1024))

	gen := CreateGenerator(5, words, nil, nil, rng, GeneratorParams{
		MinWordLength: 3,
	})

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	count := 0
	maxCount := 5

	fmt.Println("Generated Grids:")
	for grid := range gen.PossibleGrids(ctx) {
		count++
		fmt.Printf("Grid #%d:\n", count)

		// Print the grid representation.
		fmt.Printf("Grid #%d:\n%s\n", count, grid.Repr())

		if count >= maxCount {
			break
		}
	}

	if count != maxCount {
		t.Errorf("expected %d grids, got %d", maxCount, count)
	}
}

func BenchmarkPossibleGrids(b *testing.B) {
	words := loadWords(b)
	b.ReportAllocs()

	for _, tc := range []struct {
		name              string
		sideLength        int
		numBoardsToReturn int
	}{
		{name: "5x5", sideLength: 5, numBoardsToReturn: 5},
		{name: "6x6", sideLength: 6, numBoardsToReturn: 5},
		{name: "7x7", sideLength: 7, numBoardsToReturn: 5},
		{name: "8x8", sideLength: 8, numBoardsToReturn: 5},
	} {
		b.Run(tc.name, func(b *testing.B) {
			rng := rand.New(rand.NewPCG(42, 1024))
			for b.Loop() {
				gen := CreateGenerator(tc.sideLength, words, nil, nil, rng, GeneratorParams{
					MinWordLength: 3,
				})

				numReturned := 0
				for range gen.PossibleGrids(b.Context()) {
					numReturned++
					if numReturned >= tc.numBoardsToReturn {
						break
					}
				}
				b.ReportMetric(float64(numReturned), "boards_returned")
			}
		})
	}
}
