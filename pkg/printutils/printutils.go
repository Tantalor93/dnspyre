// Package printutils provides utility functions for printing colored output to the console.
package printutils

import (
	"github.com/fatih/color"
)

var (
	// ErrFprintf is a wrapper for printing colored errors.
	ErrFprintf = color.New(color.FgRed).FprintfFunc()
	// SuccessFprintf is a wrapper for printing colored successes.
	SuccessFprintf = color.New(color.FgGreen).FprintfFunc()
	// NeutralFprintf is a wrapper for printing neutral information.
	NeutralFprintf = color.New().FprintfFunc()

	highlightColor = color.New(color.FgYellow)
	// HighlightSprintf is a wrapper for highlighting formatted strings with color.
	HighlightSprintf = highlightColor.SprintfFunc()
	// HighlightSprint is a wrapper for highlighting strings with color.
	HighlightSprint = highlightColor.SprintFunc()
)
