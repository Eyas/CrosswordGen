package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"time"

	"crosswarped.com/ggg"
	xw_generator "crosswarped.com/ggg/xw_generator/generator"
)

func main() {

	firstOnly := flag.Bool("first", false, "Only generate the first grid")
	doAll := flag.Bool("all", false, "Generate all grids")
	sideLength := flag.Int("width", 4, "The width of the grid")
	loadWordsFromCloud := flag.Bool("cloud", false, "Load words from cloud")
	obscure := flag.Bool("obscure", false, "Include obscure words")
	scope := flag.String("scope", "regular", "The scope of the words to load")
	timeout := flag.Duration("timeout", 1*time.Minute, "The timeout for the generator")

	flag.Parse()

	if *firstOnly && *doAll {
		fmt.Println("Cannot use both -first and -all")
		os.Exit(1)
	}

	ctx := context.Background()

	randSource := rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().Nanosecond()))

	var preferredWords, obscureWords, excludedWords []string
	if *loadWordsFromCloud {
		fmt.Println("Loading words from cloud...")
		p, o, err := ggg.LoadWordsFromCloud(ctx, *scope, *obscure, *sideLength)
		if err != nil {
			fmt.Println("Error loading words from cloud:", err)
			os.Exit(1)
		}
		preferredWords = p
		obscureWords = o
	}
	fmt.Println("Preferred words:", len(preferredWords))
	fmt.Println("Obscure words:", len(obscureWords))
	fmt.Println("Excluded words:", len(excludedWords))

	grid := xw_generator.CreateGenerator(
		*sideLength,
		preferredWords,
		obscureWords,
		excludedWords,
		rand.New(randSource),
	)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	for grid := range grid.PossibleGrids(ctx) {
		if err := ctx.Err(); err != nil {
			fmt.Println("Context error:", err)
			break
		}

		fmt.Println("--------------------------------")
		fmt.Println(grid.Repr())

		if *firstOnly {
			break
		}

		if *doAll {
			continue
		}

		// Wait for user input and determine if they want to continue.
		// Continue (any key), or stop (n)
		fmt.Print("Continue? [Y/n]: ")
		var input string
		fmt.Scanln(&input)
		if input == "s" || input == "S" {
			fmt.Println(grid.DebugString())
		}
		if input == "n" || input == "N" {
			break
		}
	}

	fmt.Println("--------------------------------")
	fmt.Println("Done")

	if ctx.Err() != nil {
		fmt.Println("Context error:", ctx.Err())
	}
}
