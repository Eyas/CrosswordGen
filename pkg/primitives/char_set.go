package primitives

import "fmt"

// CharSet efficiently represents a set of characters.
type CharSet struct {
	available []bool
	min       rune
	count     int
}

func NewCharSet(min, max rune) *CharSet {
	return &CharSet{
		available: make([]bool, max-min+1),
		min:       min,
		count:     0,
	}
}

// DefaultCharSet is the default character set for the generator.
// It includes all ASCII characters from a to z, plus '`' (backtick), representing blocked cells.
func DefaultCharSet() *CharSet {
	return NewCharSet('`', 'z')
}

// Add adds a character to the set.
func (c *CharSet) Add(r rune) error {
	if r < c.min || r > c.min+rune(len(c.available)-1) {
		return fmt.Errorf("character %c is out of range", r)
	}

	if c.available[r-c.min] {
		return nil
	}

	c.count++
	c.available[r-c.min] = true
	return nil
}

// AddAll adds all characters from another set to this set.
func (c *CharSet) AddAll(other *CharSet) {
	// In Debug mode only, we assert that the two sets have the same min and max.
	if c.min != other.min {
		panic(fmt.Sprintf("cannot add all: char sets have different min, %c != %c", c.min, other.min))
	}
	if len(c.available) != len(other.available) {
		panic(fmt.Sprintf("cannot add all: char sets have different lengths, %d != %d", len(c.available), len(other.available)))
	}

	if c.IsFull() {
		return
	}

	if other.IsFull() && !c.IsFull() {
		// Fill c.available with true.
		for i := range c.available {
			c.available[i] = true
		}
		c.count = len(c.available)
		return
	}

	for oi, oa := range other.available {
		if !oa || c.available[oi] {
			continue
		}
		c.available[oi] = true
		c.count++
	}
}

// Contains checks if a character is in the set.
func (c *CharSet) Contains(r rune) bool {
	return c.available[r-c.min]
}

// IsFull checks if the set is full.
func (c *CharSet) IsFull() bool {
	return c.count == len(c.available)
}
