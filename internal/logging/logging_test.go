package logging

import "testing"

func TestDefaultLogger(t *testing.T) {
	Default.Printf("test %d", 1) // just ensure no panic
}
