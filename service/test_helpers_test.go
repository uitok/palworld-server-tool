package service

import (
	"path/filepath"
	"testing"

	"go.etcd.io/bbolt"
)

func newTestDB(t *testing.T, buckets ...string) *bbolt.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "service-test.db")
	db, err := bbolt.Open(dbPath, 0o600, nil)
	if err != nil {
		t.Fatalf("open test db: %v", err)
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
		t.Fatalf("create test buckets: %v", err)
	}

	return db
}
