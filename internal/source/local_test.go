package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyFromLocalCleansTempDirOnCopyFailure(t *testing.T) {
	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "Level.sav"), []byte("level"), 0o644); err != nil {
		t.Fatalf("write level.sav: %v", err)
	}

	way := "p3-local-cleanup"
	prefix := "palworldsav-" + way + "-"
	entriesBefore, err := os.ReadDir(os.TempDir())
	if err != nil {
		t.Fatalf("read temp dir: %v", err)
	}
	for _, entry := range entriesBefore {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			t.Fatalf("expected clean temp namespace before test, found %s", entry.Name())
		}
	}

	_, err = CopyFromLocal(sourceDir, way)
	if err == nil {
		t.Fatal("expected Players copy failure, got nil")
	}

	entriesAfter, err := os.ReadDir(os.TempDir())
	if err != nil {
		t.Fatalf("read temp dir after copy: %v", err)
	}
	for _, entry := range entriesAfter {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			t.Fatalf("expected temp dir to be cleaned up, found %s", entry.Name())
		}
	}
}
