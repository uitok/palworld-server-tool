package api

import (
	"os"
	"path/filepath"
	"testing"

	"go.etcd.io/bbolt"
)

func newAPITestDB(t *testing.T, buckets ...string) *bbolt.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "api-test.db")
	db, err := bbolt.Open(dbPath, 0o600, nil)
	if err != nil {
		t.Fatalf("open api test db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	err = db.Update(func(tx *bbolt.Tx) error {
		for _, bucket := range buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("create api test buckets: %v", err)
	}

	setDB(db)
	return db
}

func chdirForTest(t *testing.T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
}
