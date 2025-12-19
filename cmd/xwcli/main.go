package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/Eyas/xwgen"
)

func main() {

	firstOnly := flag.Bool("first", false, "Only generate the first grid")
	doAll := flag.Bool("all", false, "Generate all grids")
	sideLength := flag.Int("width", 4, "The width of the grid")
	minWordLength := flag.Int("min_length", 3, "The minimum word length")
	file := flag.String("file", "", "The file to load words from")
	obscureFile := flag.String("obscure", "", "The file to load obscure words from")
	excludedFile := flag.String("excluded", "", "The file to load excluded words from")

	timeout := flag.Duration("timeout", 1*time.Minute, "The timeout for the generator")

	profile := flag.Bool("profile", false, "Profile the generator")
	profileFile := flag.String("profile-file", "cpu.pprof", "The file to write the CPU profile to")
	memoryProfileFile := flag.String("memory-profile-file", "mem.pprof", "The file to write the memory profile to")

	flag.Parse()

	if *firstOnly && *doAll {
		fmt.Println("Cannot use both -first and -all")
		os.Exit(1)
	}

	ctx := context.Background()

	randSource := rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().Nanosecond()))

	var preferredWords, obscureWords, excludedWords []string
	if *file != "" {
		fmt.Println("Loading words from file...")
		var err error
		if preferredWords, err = loadFromFile(ctx, *file, *minWordLength, *sideLength); err != nil {
			fmt.Println("Error loading words from file:", err)
			os.Exit(1)
		}
	}
	if *obscureFile != "" {
		fmt.Println("Loading obscure words from file...")
		var err error
		if obscureWords, err = loadFromFile(ctx, *obscureFile, *minWordLength, *sideLength); err != nil {
			fmt.Println("Error loading obscure words from file:", err)
			os.Exit(1)
		}
	}
	if *excludedFile != "" {
		fmt.Println("Loading excluded words from file...")
		var err error
		if excludedWords, err = loadFromFile(ctx, *excludedFile, *minWordLength, *sideLength); err != nil {
			fmt.Println("Error loading excluded words from file:", err)
			os.Exit(1)
		}
	}

	fmt.Println("Preferred words:", len(preferredWords))
	fmt.Println("Obscure words:", len(obscureWords))
	fmt.Println("Excluded words:", len(excludedWords))

	var mf *os.File
	if *profile {
		f, err := os.Create(*profileFile)
		if err != nil {
			fmt.Println("Error creating profile file:", err)
			os.Exit(1)
		}
		defer f.Close()

		mf, err = os.Create(*memoryProfileFile)
		if err != nil {
			fmt.Println("Error creating memory profile file:", err)
			os.Exit(1)
		}
		defer mf.Close()

		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Println("Error starting CPU profile:", err)
			os.Exit(1)
		}
		defer pprof.StopCPUProfile()
	}

	grid := xwgen.CreateGenerator(
		*sideLength,
		preferredWords,
		obscureWords,
		excludedWords,
		rand.New(randSource),
		xwgen.GeneratorParams{
			MinWordLength: 3,
			MaxWordLength: *sideLength,
		},
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

	if mf != nil {
		pprof.WriteHeapProfile(mf)
	}

	if ctx.Err() != nil {
		fmt.Println("Context error:", ctx.Err())
	}
}

func loadFromFile(ctx context.Context, path string, minWordLength int, maxWordLength int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var words []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		word := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if strings.HasPrefix(word, "#") {
			continue
		}
		if len(word) < minWordLength || len(word) > maxWordLength {
			continue
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		for _, r := range word {
			if r < 'a' || r > 'z' {
				return nil, fmt.Errorf("word %s contains non-lowercase letter %q", word, r)
			}
		}
		words = append(words, word)
	}
	return words, scanner.Err()
}
