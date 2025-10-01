package tui

import rw "github.com/mattn/go-runewidth"

func init() {
	// Force box drawing characters to use narrow widths so borders align
	// across terminals that render them as single-cell glyphs.
	rw.EastAsianWidth = false
	rw.DefaultCondition = rw.NewCondition()
}
