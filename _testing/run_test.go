package main

import (
	"os"
	"path/filepath"
	"testing"
)

func testBinary(t *testing.T) string {
	t.Helper()
	for _, name := range []string{"gocode", "gocode.exe"} {
		path := filepath.Join(".", name)
		if _, err := os.Stat(path); err == nil {
			absolute, err := filepath.Abs(path)
			if err == nil {
				return absolute
			}
			return path
		}
	}
	t.Fatalf("gocode binary not found")
	return ""
}

func TestAutocomplete(t *testing.T) {
	if !runAutocomplete(".", testBinary(t)) {
		t.Fail()
	}
}
func TestTypesInfo(t *testing.T) {
	if !runTypesInfo(".", testBinary(t)) {
		t.Fail()
	}
}
