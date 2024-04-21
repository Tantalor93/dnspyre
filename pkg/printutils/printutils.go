package printutils

import "github.com/fatih/color"

var (
	// ErrPrint is a wrapper for printing colored errors.
	ErrPrint = color.New(color.FgRed).FprintfFunc()
	// SuccessPrint is a wrapper for printing colored successes.
	SuccessPrint = color.New(color.FgGreen).FprintfFunc()
	// HighlightStr is a wrapper for highlighting strings with color.
	HighlightStr = color.New(color.FgYellow).SprintFunc()
)
