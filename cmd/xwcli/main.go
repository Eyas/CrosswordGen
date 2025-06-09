package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	xw_generator "crosswarped.com/ggg/xw_generator/generator"
)

func main() {

	firstOnly := flag.Bool("first", false, "Only generate the first grid")
	doAll := flag.Bool("all", false, "Generate all grids")
	flag.Parse()

	if *firstOnly && *doAll {
		fmt.Println("Cannot use both -first and -all")
		os.Exit(1)
	}

	grid := xw_generator.Generator{
		LineLength:     4,
		PreferredWords: []string{"tart", "aria", "pest", "saks", "taps", "area", "risk", "tats", "rat", "rats", "rate", "rote", "row", "rows", "rowy", "show", "shat", "fact", "fat", "fate", "fast"},
		ObscureWords:   []string{"ffff", "aba", "tat", "abbs", "abb", "baba", "papa", "yes", "yep", "nop", "pon", "pom", "stat", "tate", "tete", "tata", "tsts"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for grid := range grid.PossibleGrids(ctx) {
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
		if input == "n" || input == "N" {
			break
		}
	}

	fmt.Println("--------------------------------")
	fmt.Println("Done")
}
