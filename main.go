// Command matrix-rain draws Matrix digital rain in the terminal until it gets a signal.
// It does not read stdin.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unsafe"
)

const (
	enterScreen = "\x1b[?1049h\x1b[?25l\x1b[0m\x1b[2J"
	clearScreen = "\x1b[0m\x1b[2J"
	leaveScreen = "\x1b[0m\x1b[?25h\x1b[?1049l"
)

func main() {
	fps := flag.Int("fps", 30, "frames per second (1-240)")
	colorHex := flag.String("color", "#00ff41", "main color as #rrggbb")
	speed := flag.Float64("speed", 1.0, "speed multiplier (> 0)")
	density := flag.Float64("density", 0.7, "fraction of each column that the rain covers (0-1]")
	charset := flag.String("charset", "katakana", "glyph set: katakana, ascii or binary")
	flag.Parse()

	base, err := parseHex(*colorHex)
	if err != nil {
		fail(err)
	}
	set, err := glyphs(*charset)
	if err != nil {
		fail(err)
	}
	if *fps < 1 || *fps > 240 {
		fail(fmt.Errorf("fps %d: want 1-240", *fps))
	}
	if *speed <= 0 {
		fail(fmt.Errorf("speed %v: want > 0", *speed))
	}
	if *density <= 0 || *density > 1 {
		fail(fmt.Errorf("density %v: want 0 < density <= 1", *density))
	}
	cfg := config{fps: *fps, speed: *speed, density: *density}

	// Catch signals before the first size read, so an early SIGWINCH is not lost.
	sigs := make(chan os.Signal, 4)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGWINCH)

	cols, rows, err := termSize()
	if err != nil {
		fail(err)
	}

	out := bufio.NewWriterSize(os.Stdout, 256*1024)
	out.WriteString(enterScreen)

	var f *field
	var prev, cur []cell
	reset := func() {
		f = newField(cols, rows, set, base, cfg, rand.Float64)
		prev = blank(cols * rows)
		cur = make([]cell, cols*rows)
	}
	reset()

	ticker := time.NewTicker(time.Second / time.Duration(cfg.fps))
	defer ticker.Stop()
	for {
		select {
		case s := <-sigs:
			if s == syscall.SIGWINCH {
				if c, r, err := termSize(); err == nil {
					cols, rows = c, r
					reset()
					out.WriteString(clearScreen)
				}
				continue
			}
			out.WriteString(leaveScreen)
			out.Flush()
			os.Exit(0)
		case <-ticker.C:
			f.tick()
			for y := range rows {
				for x := range cols {
					ch, col := f.cell(x, y)
					cur[y*cols+x] = cell{ch, col}
				}
			}
			renderDiff(out, prev, cur, cols)
			if err := out.Flush(); err != nil {
				os.Exit(1) // The terminal is gone.
			}
		}
	}
}

func blank(n int) []cell {
	b := make([]cell, n)
	for i := range b {
		b[i].ch = ' '
	}
	return b
}

// termSize reads the terminal size of stdout with TIOCGWINSZ.
func termSize() (cols, rows int, err error) {
	var ws struct{ row, col, xpixel, ypixel uint16 }
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, os.Stdout.Fd(), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws)))
	if errno != 0 {
		return 0, 0, fmt.Errorf("stdout is not a terminal: %w", errno)
	}
	if ws.col == 0 || ws.row == 0 {
		return 0, 0, fmt.Errorf("terminal size is %dx%d", ws.col, ws.row)
	}
	return int(ws.col), int(ws.row), nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "matrix-rain:", err)
	os.Exit(2)
}
