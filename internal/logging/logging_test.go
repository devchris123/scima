package logging

import "testing"

func TestDefaultLogger(_ *testing.T) {
	Default.Printf("test %d", 1) // just ensure no panic
}
