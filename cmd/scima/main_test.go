package main

import "testing"

func TestRootCmdRequiresDSN(t *testing.T) {
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err == nil {
		t.Fatalf("expected error due to missing dsn")
	}
}
