// Package logging provides a minimal logging interface and stdlib wrapper for migration operations.
package logging

import "log"

// Logger minimal interface for substituting structured loggers later.
// Logger is a minimal interface for substituting structured loggers later.
type Logger interface {
	Printf(format string, v ...any)
}

// Std wraps stdlib log.
// Std wraps the standard library log for Logger interface.
type Std struct{}

// Printf logs a formatted message using the standard library log.
func (s Std) Printf(format string, v ...any) { log.Printf(format, v...) }

// Default is the default Logger implementation using Std.
var Default Logger = Std{}
