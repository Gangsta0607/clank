package main

import (
	"strings"
	"unicode/utf8"
)

// layoutColumns раскладывает ячейки по колонкам как ls: column-major,
// то есть номера читаются сверху вниз по каждой колонке. Если в ширину
// не влезает даже одна ячейка — по одной в строку.
func layoutColumns(cells []string, width int) []string {
	if len(cells) == 0 {
		return nil
	}
	const gap = 2
	maxw := 0
	for _, c := range cells {
		if n := utf8.RuneCountInString(c); n > maxw {
			maxw = n
		}
	}
	colw := maxw + gap
	cols := width / colw
	if cols < 1 || colw > width {
		cols = 1
	}
	if cols > len(cells) {
		cols = len(cells)
	}
	rows := (len(cells) + cols - 1) / cols

	lines := make([]string, 0, rows)
	for r := 0; r < rows; r++ {
		var b strings.Builder
		for c := 0; c < cols; c++ {
			i := c*rows + r
			if i >= len(cells) {
				continue
			}
			b.WriteString(cells[i])
			if c != cols-1 && i+rows < len(cells) {
				pad := colw - utf8.RuneCountInString(cells[i])
				b.WriteString(strings.Repeat(" ", pad))
			}
		}
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}
	return lines
}
