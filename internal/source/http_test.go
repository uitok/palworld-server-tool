package source

import (
	"archive/zip"
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeSaveZip(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"Level.sav":           "level",
		"Players/player1.sav": "player",
	}
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buf.Bytes()
}

func countTempDirsWithPrefix(t *testing.T, prefix string) int {
	t.Helper()
	entries, err := os.ReadDir(os.TempDir())
	if err != nil {
		t.Fatalf("read temp dir: %v", err)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			count++
		}
	}
	return count
}

func sanitizeWayForTest(name string) string {
	replacer := strings.NewReplacer("/", "-", " ", "-", "_", "-")
	return strings.ToLower(replacer.Replace(name))
}

func TestDownloadFromHttp(t *testing.T) {
	t.Run("downloads and extracts level save", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(makeSaveZip(t))
		}))
		defer server.Close()

		levelPath, err := DownloadFromHttp(server.URL, "p3-http-success")
		if err != nil {
			t.Fatalf("download from http: %v", err)
		}
		if filepath.Base(levelPath) != "Level.sav" {
			t.Fatalf("expected Level.sav path, got %s", levelPath)
		}
		if _, err := os.Stat(levelPath); err != nil {
			t.Fatalf("expected extracted level save to exist, got %v", err)
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(levelPath), "Players", "player1.sav")); err != nil {
			t.Fatalf("expected player save to exist, got %v", err)
		}
		_ = os.RemoveAll(filepath.Dir(levelPath))
	})

	t.Run("maps 404 to not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer server.Close()

		_, err := DownloadFromHttp(server.URL, "p3-http-404")
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected os.ErrNotExist wrapped, got %v", err)
		}
	})

	t.Run("cleans temp dir on unzip failure", func(t *testing.T) {
		way := sanitizeWayForTest(t.Name())
		prefix := "palworldsav-http-" + way + "-"
		if got := countTempDirsWithPrefix(t, prefix); got != 0 {
			t.Fatalf("expected clean temp namespace before test, got %d", got)
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not-a-zip"))
		}))
		defer server.Close()

		_, err := DownloadFromHttp(server.URL, way)
		if err == nil {
			t.Fatal("expected unzip failure, got nil")
		}
		if got := countTempDirsWithPrefix(t, prefix); got != 0 {
			t.Fatalf("expected temp dir cleanup after failure, got %d leftover dirs", got)
		}
	})
}
