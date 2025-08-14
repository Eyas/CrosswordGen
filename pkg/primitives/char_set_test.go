package primitives

import (
	"testing"
)

func TestCharSet_Add(t *testing.T) {
	cs := NewCharSet()

	tests := []struct {
		name      string
		char      rune
		wantErr   bool
		wantCount int
	}{
		{"add 'a'", 'a', false, 1},
		{"add 'b'", 'b', false, 2},
		{"add 'c'", 'c', false, 3},
		{"add 'a' again", 'a', false, 3}, // should not increase count
		{"add out of range low", 'A', true, 3},
		{"add out of range high", '~', true, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cs.Add(tt.char)
			if (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
			}
			if cs.count != tt.wantCount {
				t.Errorf("count = %d, want %d", cs.count, tt.wantCount)
			}
		})
	}
}

func TestCharSet_AddAll(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (*CharSet, *CharSet)
		expected int
	}{
		{
			name: "add to empty set",
			setup: func() (*CharSet, *CharSet) {
				cs1 := NewCharSet()
				cs2 := NewCharSet()
				cs2.Add('a')
				cs2.Add('b')
				return cs1, cs2
			},
			expected: 2,
		},
		{
			name: "add overlapping sets",
			setup: func() (*CharSet, *CharSet) {
				cs1 := NewCharSet()
				cs1.Add('a')
				cs2 := NewCharSet()
				cs2.Add('b')
				cs2.Add('c')
				return cs1, cs2
			},
			expected: 3,
		},
		{
			name: "add to partially overlapping set",
			setup: func() (*CharSet, *CharSet) {
				cs1 := NewCharSet()
				cs1.Add('a')
				cs1.Add('b')
				cs1.Add('c')
				cs2 := NewCharSet()
				cs2.Add('a')
				cs2.Add('d')
				return cs1, cs2
			},
			expected: 4,
		},
		{
			name: "add to full set",
			setup: func() (*CharSet, *CharSet) {
				cs1 := NewCharSet()
				for i := '`'; i <= 'z'; i++ {
					cs1.Add(i)
				}
				cs2 := NewCharSet()
				cs2.Add('a')
				cs2.Add('b')
				cs2.Add('c')
				return cs1, cs2
			},
			expected: 27,
		},
		{
			name: "add full set to empty",
			setup: func() (*CharSet, *CharSet) {
				cs1 := NewCharSet()
				cs1.Add('a')

				cs2 := NewCharSet()
				for i := '`'; i <= 'z'; i++ {
					cs2.Add(i)
				}
				return cs1, cs2
			},
			expected: 27,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs1, cs2 := tt.setup()
			cs1.AddAll(cs2)
			if cs1.count != tt.expected {
				t.Errorf("count = %d, want %d", cs1.count, tt.expected)
			}
		})
	}
}

func TestCharSet_Contains(t *testing.T) {
	cs := NewCharSet()
	cs.Add('a')
	cs.Add('c')

	tests := []struct {
		name string
		char rune
		want bool
	}{
		{"contains 'a'", 'a', true},
		{"contains 'b'", 'b', false},
		{"contains 'c'", 'c', true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cs.Contains(tt.char); got != tt.want {
				t.Errorf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("full always returns true", func(t *testing.T) {
		cs := NewCharSet()
		for i := '`'; i <= 'z'; i++ {
			cs.Add(i)
		}
		if !cs.IsFull() {
			t.Errorf("IsFull() = false, want true")
		}

		if !cs.Contains('a') {
			t.Errorf("Contains() = false, want true")
		}
		if !cs.Contains('b') {
			t.Errorf("Contains() = false, want true")
		}
		if !cs.Contains('c') {
			t.Errorf("Contains() = false, want true")
		}
	})
}

func TestCharSet_IsFull(t *testing.T) {
	cs := NewCharSet()

	if cs.IsFull() {
		t.Error("IsFull() = true, want false for empty set")
	}

	cs.Add('a')
	cs.Add('b')
	if cs.IsFull() {
		t.Error("IsFull() = true, want false for partially filled set")
	}

	for i := '`'; i <= 'z'; i++ {
		cs.Add(i)
	}

	if !cs.IsFull() {
		t.Error("IsFull() = false, want true for full set")
	}
}

func TestCharSet_Capacity(t *testing.T) {
	cs := NewCharSet()
	if cs.Capacity() != 27 {
		t.Errorf("Capacity() = %d, want 3", cs.Capacity())
	}
}

func TestCharSet_Count(t *testing.T) {
	cs := NewCharSet()
	if cs.Count() != 0 {
		t.Errorf("Count() = %d, want 0", cs.Count())
	}

	cs.Add('a')
	if cs.Count() != 1 {
		t.Errorf("Count() = %d, want 1", cs.Count())
	}

	cs.Add('b')
	if cs.Count() != 2 {
		t.Errorf("Count() = %d, want 2", cs.Count())
	}

	for i := '`'; i <= 'z'; i++ {
		cs.Add(i)
	}
	if cs.Count() != 27 {
		t.Errorf("Count() = %d, want 27", cs.Count())
	}
}
