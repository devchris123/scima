package logging

import "log"

// Logger minimal interface for substituting structured loggers later.
type Logger interface {
	Printf(format string, v ...any)
}

// Std wraps stdlib log.
type Std struct{}

func (s Std) Printf(format string, v ...any) { log.Printf(format, v...) }

var Default Logger = Std{}
