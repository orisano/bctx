package bctx

import (
	"os"
	"strings"
	"testing"
)

func TestReadIgnoreNonExistentFile(t *testing.T) {
	if _, err := ReadIgnore("testdata/non_existent_file"); err != nil {
		t.Error("Should have returned nil for non-existent-file")
	}
}

func TestReadIgnoreBadPermissions(t *testing.T) {
	file := "testdata/bad_permissions"

	defer func () { os.Chmod(file, 0644) }()
	if err := os.Chmod(file, 0000); err != nil {
		t.Errorf("Could not set permissions on %s", file)
	}

	if _, err := ReadIgnore("testdata/bad_permissions"); ! strings.Contains(err.Error(), "failed to open") {
		t.Errorf("Error string did not contain 'failed to open' the file %s", file)
	}
}

func TestReadIgnore(t *testing.T) {
	content, err := ReadIgnore("testdata/dockerignore")
	if err != nil{
		t.Error("Failed to read testdata/dockerignore")
	}

	for _, expected := range []string{".git", "*.swp"} {
		missing := true
		for _, v := range content {
			if v == expected {
				missing = false
				break
			}
		}
		if missing {
			t.Errorf("Did not find '%s' in testdata/dockerignore", expected)
		}
	}
}
