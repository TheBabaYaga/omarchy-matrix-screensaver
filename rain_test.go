package main

import (
	"bufio"
	"bytes"
	"math/rand/v2"
	"testing"
)

// seq returns a fake random source that cycles through vals.
func seq(vals ...float64) func() float64 {
	i := 0
	return func() float64 {
		v := vals[i%len(vals)]
		i++
		return v
	}
}

func TestParseHex(t *testing.T) {
	tests := []struct {
		in      string
		want    rgb
		wantErr bool
	}{
		{"#00ff41", rgb{0x00, 0xff, 0x41}, false},
		{"00FF41", rgb{0x00, 0xff, 0x41}, false},
		{"#ffffff", rgb{0xff, 0xff, 0xff}, false},
		{"#000000", rgb{}, false},
		{"#0f4", rgb{}, true},
		{"#00ff4", rgb{}, true},
		{"#00ff411", rgb{}, true},
		{"#gg0000", rgb{}, true},
		{"", rgb{}, true},
	}
	for _, tt := range tests {
		got, err := parseHex(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseHex(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("parseHex(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestGlyphs(t *testing.T) {
	tests := []struct {
		name     string
		wantErr  bool
		contains []rune
		excludes []rune
		minLen   int
	}{
		{"katakana", false, []rune{0xFF66, 0xFF9D, '0', '9'}, []rune{'A', 0xFF65, 0xFF9E}, 60},
		{"ascii", false, []rune{'!', 'A', 'z', '~'}, []rune{' ', 0x7F}, 94},
		{"binary", false, []rune{'0', '1'}, []rune{'2'}, 2},
		{"emoji", true, nil, nil, 0},
		{"", true, nil, nil, 0},
	}
	for _, tt := range tests {
		got, err := glyphs(tt.name)
		if (err != nil) != tt.wantErr {
			t.Errorf("glyphs(%q) err = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if tt.wantErr {
			continue
		}
		if len(got) < tt.minLen {
			t.Errorf("glyphs(%q) len = %d, want >= %d", tt.name, len(got), tt.minLen)
		}
		set := map[rune]bool{}
		for _, r := range got {
			set[r] = true
		}
		for _, r := range tt.contains {
			if !set[r] {
				t.Errorf("glyphs(%q) missing %q", tt.name, r)
			}
		}
		for _, r := range tt.excludes {
			if set[r] {
				t.Errorf("glyphs(%q) has unwanted %q", tt.name, r)
			}
		}
	}
}

func TestFade(t *testing.T) {
	base := rgb{0, 200, 80}
	tests := []struct {
		name       string
		dist, len  int
		want       rgb
		wantBright bool // head: brighter than base in every channel
	}{
		{"head is near white", 0, 10, rgb{}, true},
		{"first trail cell is full color", 1, 10, base, false},
		{"last trail cell is dim, not black", 9, 10, rgb{0, 25, 10}, false},
		{"past the tail is black", 10, 10, rgb{}, false},
		{"above the head is black", -1, 10, rgb{}, false},
		{"mid trail is half", 5, 10, rgb{0, 100, 40}, false},
	}
	for _, tt := range tests {
		got := fade(base, tt.dist, tt.len)
		if tt.wantBright {
			if got.r < 180 || got.g < base.g || got.b < 180 {
				t.Errorf("%s: fade = %v, want near white", tt.name, got)
			}
			continue
		}
		if got != tt.want {
			t.Errorf("%s: fade(%d,%d) = %v, want %v", tt.name, tt.dist, tt.len, got, tt.want)
		}
	}
}

func TestFadeIsMonotonic(t *testing.T) {
	base := rgb{0, 255, 65}
	prev := fade(base, 1, 20)
	for d := 2; d < 20; d++ {
		got := fade(base, d, 20)
		if got.g > prev.g || got.g == 0 {
			t.Fatalf("fade(%d) = %v after %v: want dimmer and non-zero", d, got, prev)
		}
		prev = got
	}
}

func TestRespawnDelay(t *testing.T) {
	tests := []struct {
		name    string
		active  float64
		density float64
		u       float64
		want    int
	}{
		{"full density never waits", 100, 1, 0.9, 0},
		{"half density, mid draw equals active time", 100, 0.5, 0.5, 100},
		{"half density, max draw is twice", 100, 0.5, 1, 200},
		{"zero draw", 100, 0.25, 0, 0},
		{"quarter density mean", 100, 0.25, 0.5, 300},
	}
	for _, tt := range tests {
		if got := respawnDelay(tt.active, tt.density, tt.u); got != tt.want {
			t.Errorf("%s: respawnDelay = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestColumnStep(t *testing.T) {
	cfg := config{fps: 10, speed: 1, density: 0.5}

	t.Run("empty column counts down", func(t *testing.T) {
		c := column{wait: 2}
		c.step(20, cfg, seq(0.5))
		if len(c.drops) != 0 || c.wait != 1 {
			t.Fatalf("got %+v, want no drops with wait 1", c)
		}
	})

	t.Run("spawn sets head, speed, length and next wait", func(t *testing.T) {
		c := column{}
		c.step(20, cfg, seq(0, 0, 0.5))
		if len(c.drops) != 1 || c.drops[0].head != 0 {
			t.Fatalf("got %+v, want one drop at head 0", c)
		}
		d := c.drops[0]
		if want := minSpeed / 10; d.speed != want {
			t.Errorf("speed = %v, want %v", d.speed, want)
		}
		if d.length != minTrail {
			t.Errorf("length = %d, want %d", d.length, minTrail)
		}
		// trail time = 4/0.6 frames, density 0.5, u 0.5 -> 6
		if c.wait != 6 {
			t.Errorf("wait = %d, want 6", c.wait)
		}
	})

	t.Run("speed multiplier scales speed", func(t *testing.T) {
		c := column{}
		c.step(20, config{fps: 10, speed: 2, density: 0.5}, seq(1, 0, 0))
		if want := 2 * maxSpeed / 10; c.drops[0].speed != want {
			t.Errorf("speed = %v, want %v", c.drops[0].speed, want)
		}
	})

	t.Run("drop moves down", func(t *testing.T) {
		c := column{wait: 5, drops: []drop{{head: 6, speed: 0.5, length: 5}}}
		c.step(20, cfg, seq(0.5))
		if len(c.drops) != 1 || c.drops[0].head != 6.5 {
			t.Fatalf("got %+v, want head 6.5", c)
		}
	})

	t.Run("drop ends when tail leaves screen", func(t *testing.T) {
		c := column{wait: 5, drops: []drop{{head: 24.5, speed: 1, length: 5}}}
		c.step(20, cfg, seq(0.5))
		if len(c.drops) != 0 {
			t.Fatalf("got %+v, want no drops", c)
		}
	})

	t.Run("drop stays while tail is on screen", func(t *testing.T) {
		c := column{wait: 5, drops: []drop{{head: 23, speed: 1, length: 5}}}
		c.step(20, cfg, seq(0.5))
		if len(c.drops) != 1 {
			t.Fatalf("got %+v, want one drop", c)
		}
	})

	t.Run("no new drop while last tail is on row 0", func(t *testing.T) {
		c := column{drops: []drop{{head: 2, speed: 1, length: 5}}}
		c.step(20, cfg, seq(0.5))
		if len(c.drops) != 1 || c.wait != 0 {
			t.Fatalf("got %+v, want one drop and wait 0", c)
		}
	})

	t.Run("gap counts down after last tail leaves row 0", func(t *testing.T) {
		c := column{wait: 3, drops: []drop{{head: 8, speed: 1, length: 5}}}
		c.step(20, cfg, seq(0.5))
		if len(c.drops) != 1 || c.wait != 2 {
			t.Fatalf("got %+v, want one drop and wait 2", c)
		}
	})

	t.Run("new drop starts while old drop is on screen", func(t *testing.T) {
		c := column{drops: []drop{{head: 8, speed: 1, length: 5}}}
		c.step(20, cfg, seq(0, 0, 0))
		if len(c.drops) != 2 || c.drops[1].head != 0 {
			t.Fatalf("got %+v, want two drops, new one at head 0", c)
		}
	})
}

func TestColumnLit(t *testing.T) {
	c := column{drops: []drop{{head: 15, length: 10}, {head: 4, length: 3}}}
	tests := []struct {
		y              int
		wantDist, wLen int
		wantOK         bool
	}{
		{0, 0, 0, false},
		{2, 2, 3, true},
		{4, 0, 3, true},
		{5, 0, 0, false},
		{6, 9, 10, true},
		{15, 0, 10, true},
		{16, 0, 0, false},
	}
	for _, tt := range tests {
		dist, length, ok := c.lit(tt.y)
		if ok != tt.wantOK || (ok && (dist != tt.wantDist || length != tt.wLen)) {
			t.Errorf("lit(%d) = %d, %d, %v, want %d, %d, %v", tt.y, dist, length, ok, tt.wantDist, tt.wLen, tt.wantOK)
		}
	}
}

func TestFieldTickLightsTrail(t *testing.T) {
	f := newField(3, 10, []rune{'a', 'b'}, rgb{0, 255, 0}, config{fps: 10, speed: 1, density: 1}, seq(0.3))
	for i := 0; i < 40; i++ {
		f.tick()
		for x := 0; x < f.cols; x++ {
			for y := 0; y < f.rows; y++ {
				ch, col := f.cell(x, y)
				_, _, lit := f.columns[x].lit(y)
				if lit && (ch == ' ' || col == (rgb{})) {
					t.Fatalf("tick %d cell %d,%d: want lit, got %q %v", i, x, y, ch, col)
				}
				if !lit && ch != ' ' {
					t.Fatalf("tick %d cell %d,%d: want blank, got %q", i, x, y, ch)
				}
			}
		}
	}
}

// litFraction gives the mean part of lit cells over many frames.
func litFraction(density float64) float64 {
	r := rand.New(rand.NewPCG(1, 2))
	f := newField(200, 60, []rune{'a'}, rgb{0, 255, 0}, config{fps: 30, speed: 1, density: density}, r.Float64)
	for range 300 {
		f.tick()
	}
	lit, total := 0, 0
	for range 1000 {
		f.tick()
		for x := range f.cols {
			for y := range f.rows {
				if ch, _ := f.cell(x, y); ch != ' ' {
					lit++
				}
				total++
			}
		}
	}
	return float64(lit) / float64(total)
}

func TestDensityFillsScreen(t *testing.T) {
	tests := []struct {
		density, min, max float64
	}{
		{0.3, 0.2, 0.4},
		{0.7, 0.55, 0.8},
		{1.0, 0.75, 1.0},
	}
	for _, tt := range tests {
		if got := litFraction(tt.density); got < tt.min || got > tt.max {
			t.Errorf("density %v: lit fraction %.3f, want %.2f-%.2f", tt.density, got, tt.min, tt.max)
		}
	}
}

func TestRenderDiff(t *testing.T) {
	green := rgb{0, 255, 0}
	blank := cell{ch: ' '}
	mk := func(cells ...cell) []cell { return cells }

	tests := []struct {
		name      string
		prev, cur []cell
		want      string
	}{
		{"no change writes nothing", mk(blank, blank), mk(blank, blank), ""},
		{"one new glyph", mk(blank, blank), mk(blank, cell{'x', green}),
			"\x1b[1;2H\x1b[38;2;0;255;0mx"},
		{"adjacent cells skip cursor move and repeat color", mk(blank, blank, blank),
			mk(cell{'a', green}, cell{'b', green}, blank),
			"\x1b[1;1H\x1b[38;2;0;255;0mab"},
		{"cleared cell writes a space without color", mk(cell{'a', green}, blank), mk(blank, blank),
			"\x1b[1;1H "},
		{"second row", mk(blank, blank, blank, blank), mk(blank, blank, cell{'z', green}, blank),
			"\x1b[2;1H\x1b[38;2;0;255;0mz"},
	}
	for _, tt := range tests {
		var buf bytes.Buffer
		w := bufio.NewWriter(&buf)
		renderDiff(w, tt.prev, tt.cur, 2)
		w.Flush()
		if got := buf.String(); got != tt.want {
			t.Errorf("%s:\n got %q\nwant %q", tt.name, got, tt.want)
		}
	}
}
