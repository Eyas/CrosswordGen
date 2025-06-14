package xw_generator

import (
	"fmt"
	"strings"
)

// Grid is a 2D grid of runes.
//
// It represents a 'definite' possible grid
type Grid struct {
	grid [][]rune
}

func NewGrid(g [][]rune) Grid {
	return Grid{
		grid: g,
	}
}

func (g Grid) Width() int {
	return len(g.grid[0])
}

func (g Grid) Height() int {
	return len(g.grid)
}

func (g Grid) Get(x, y int) rune {
	return g.grid[y][x]
}

func (g Grid) Repr() string {
	lines := make([]string, g.Height())
	for y := range g.Height() {
		lines[y] = string(g.grid[y])
	}
	return strings.Join(lines, "\n")
}

func (g Grid) DebugString() string {
	return fmt.Sprintf("Grid{width: %d, height: %d, grid: %v}", g.Width(), g.Height(), g.grid)
}
