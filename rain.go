package main

import (
	"bufio"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Rain speed in rows per second, before the --speed multiplier.
const (
	minSpeed = 6.0
	maxSpeed = 30.0
)

const (
	minTrail   = 4
	fadeLevels = 8   // Quantized trail colors. Fewer levels give fewer changed cells.
	headWhite  = 0.8 // How far the head color moves from the main color to white.
	mutateRate = 0.7 // Glyph changes per trail cell per second.
)

type rgb struct{ r, g, b uint8 }

type config struct {
	fps     int
	speed   float64 // Speed multiplier.
	density float64 // Fraction of a column that trails cover, 0 < density <= 1.
}

// parseHex reads a color in the form "#rrggbb" or "rrggbb".
func parseHex(s string) (rgb, error) {
	h := strings.TrimPrefix(s, "#")
	if len(h) != 6 {
		return rgb{}, fmt.Errorf("color %q: want #rrggbb", s)
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return rgb{}, fmt.Errorf("color %q: want #rrggbb", s)
	}
	return rgb{uint8(v >> 16), uint8(v >> 8), uint8(v)}, nil
}

// glyphs returns the character set for a --charset name.
func glyphs(name string) ([]rune, error) {
	var set []rune
	switch name {
	case "katakana":
		for r := rune(0xFF66); r <= 0xFF9D; r++ {
			set = append(set, r)
		}
		set = append(set, []rune("0123456789:.=*+-<>|\"")...)
	case "ascii":
		for r := rune('!'); r <= '~'; r++ {
			set = append(set, r)
		}
	case "binary":
		set = []rune("01")
	default:
		return nil, fmt.Errorf("charset %q: want katakana, ascii or binary", name)
	}
	return set, nil
}

// fade gives the color of a cell dist rows above the head of a trail.
// The head (dist 0) is near white. The trail fades from base to dark.
// Cells outside the trail are black.
func fade(base rgb, dist, length int) rgb {
	if dist < 0 || dist >= length {
		return rgb{}
	}
	if dist == 0 {
		mix := func(c uint8) uint8 { return c + uint8(float64(255-c)*headWhite) }
		return rgb{mix(base.r), mix(base.g), mix(base.b)}
	}
	level := (fadeLevels*(length-dist) + length - 1) / length // ceil, 1..fadeLevels
	scale := func(c uint8) uint8 { return uint8(int(c) * level / fadeLevels) }
	return rgb{scale(base.r), scale(base.g), scale(base.b)}
}

// respawnDelay gives the frames between two drops in a column.
// activeFrames is the time a trail takes to pass one cell. The mean gap makes
// the trails cover a fraction density of the column.
// u is a uniform random number in [0, 1).
func respawnDelay(activeFrames, density, u float64) int {
	mean := activeFrames * (1 - density) / density
	return int(2 * mean * u)
}

// drop is one falling trail.
type drop struct {
	head   float64 // Row of the head. It can go below the screen while the tail is visible.
	speed  float64 // Rows per frame.
	length int     // Trail length in rows, head included.
}

// column holds the drops in one terminal column, oldest first.
type column struct {
	drops []drop
	wait  int // Frames to wait before the next drop, after the last tail leaves row 0.
}

// step moves the drops forward one frame, removes drops that left the screen,
// and starts a new drop when the gap after the last drop is over.
func (c *column) step(rows int, cfg config, rnd func() float64) {
	kept := c.drops[:0]
	for _, d := range c.drops {
		d.head += d.speed
		if int(d.head)-d.length < rows {
			kept = append(kept, d)
		}
	}
	c.drops = kept
	if n := len(c.drops); n > 0 && int(c.drops[n-1].head) < c.drops[n-1].length {
		return // The tail of the last drop is still on row 0.
	}
	if c.wait > 0 {
		c.wait--
		return
	}
	d := drop{
		speed:  (minSpeed + rnd()*(maxSpeed-minSpeed)) * cfg.speed / float64(cfg.fps),
		length: minTrail + int(rnd()*float64(max(rows*2/3, 1))),
	}
	c.drops = append(c.drops, d)
	c.wait = respawnDelay(float64(d.length)/d.speed, cfg.density, rnd())
}

// lit gives the distance from row y up to the nearest head whose trail covers y,
// and the length of that trail. ok is false when no trail covers y.
func (c *column) lit(y int) (dist, length int, ok bool) {
	for _, d := range c.drops {
		n := int(d.head) - y
		if n >= 0 && n < d.length && (!ok || n < dist) {
			dist, length, ok = n, d.length, true
		}
	}
	return dist, length, ok
}

// field is the full grid of columns and glyphs.
type field struct {
	cols, rows int
	columns    []column
	glyphs     []rune // rows*cols, row-major.
	set        []rune
	base       rgb
	cfg        config
	rnd        func() float64
}

func newField(cols, rows int, set []rune, base rgb, cfg config, rnd func() float64) *field {
	f := &field{cols: cols, rows: rows, columns: make([]column, cols), glyphs: make([]rune, cols*rows),
		set: set, base: base, cfg: cfg, rnd: rnd}
	for i := range f.glyphs {
		f.glyphs[i] = f.pick()
	}
	// Columns start at random times so the rain does not start as one line.
	for i := range f.columns {
		f.columns[i].wait = int(rnd() * float64(cfg.fps) * 3)
	}
	return f
}

func (f *field) pick() rune {
	return f.set[min(int(f.rnd()*float64(len(f.set))), len(f.set)-1)]
}

// tick moves every column one frame, writes new glyphs under the heads,
// and changes some glyphs in the trails.
func (f *field) tick() {
	mutateP := mutateRate / float64(f.cfg.fps)
	for x := range f.columns {
		c := &f.columns[x]
		c.step(f.rows, f.cfg, f.rnd)
		for _, d := range c.drops {
			head := int(d.head)
			for y := max(int(math.Floor(d.head-d.speed))+1, 0); y <= head && y < f.rows; y++ {
				f.glyphs[y*f.cols+x] = f.pick()
			}
			for y := max(head-d.length+1, 0); y < head && y < f.rows; y++ {
				if f.rnd() < mutateP {
					f.glyphs[y*f.cols+x] = f.pick()
				}
			}
		}
	}
}

// cell gives the glyph and color at x, y. An unlit cell is a space.
func (f *field) cell(x, y int) (rune, rgb) {
	dist, length, ok := f.columns[x].lit(y)
	if !ok {
		return ' ', rgb{}
	}
	return f.glyphs[y*f.cols+x], fade(f.base, dist, length)
}

type cell struct {
	ch  rune
	col rgb
}

// renderDiff writes the cells that differ between prev and cur, then copies cur into prev.
// It skips a cursor move when the cursor is already in place, and skips a color
// change when the color is already set.
func renderDiff(w *bufio.Writer, prev, cur []cell, cols int) {
	pos := -1 // Known cursor index, or -1.
	var color rgb
	colorSet := false
	for i, c := range cur {
		if c == prev[i] {
			continue
		}
		if pos != i {
			fmt.Fprintf(w, "\x1b[%d;%dH", i/cols+1, i%cols+1)
		}
		if c.ch != ' ' && (!colorSet || c.col != color) {
			fmt.Fprintf(w, "\x1b[38;2;%d;%d;%dm", c.col.r, c.col.g, c.col.b)
			color, colorSet = c.col, true
		}
		w.WriteRune(c.ch)
		pos = i + 1
		if pos%cols == 0 {
			pos = -1 // The terminal holds a pending wrap at the end of a row.
		}
	}
	copy(prev, cur)
}
