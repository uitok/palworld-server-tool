package tool

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/service"
	"go.etcd.io/bbolt"
)

func newToolTestDB(t *testing.T, buckets ...string) *bbolt.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "tool-test.db")
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

func chdirToolTest(t *testing.T, dir string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previous)
	})
}

func TestCleanOldBackupsRemovesExpiredFilesAndStaleRecords(t *testing.T) {
	db := newToolTestDB(t, "backups")
	workspace := t.TempDir()
	chdirToolTest(t, workspace)

	backupDir, err := GetBackupDir()
	if err != nil {
		t.Fatalf("get backup dir: %v", err)
	}

	now := time.Now().UTC()
	records := []database.Backup{
		{BackupId: "old-existing", Path: "old-existing.zip", SaveTime: now.AddDate(0, 0, -10)},
		{BackupId: "old-missing", Path: "old-missing.zip", SaveTime: now.AddDate(0, 0, -9)},
		{BackupId: "recent-missing", Path: "recent-missing.zip", SaveTime: now.Add(-2 * time.Hour)},
		{BackupId: "recent-existing", Path: "recent-existing.zip", SaveTime: now.Add(-1 * time.Hour)},
		{BackupId: "invalid-dir", Path: "invalid-dir", SaveTime: now.Add(-30 * time.Minute)},
	}
	for _, backup := range records {
		if err := service.AddBackup(db, backup); err != nil {
			t.Fatalf("seed backup %s: %v", backup.BackupId, err)
		}
	}
	if err := os.WriteFile(filepath.Join(backupDir, "old-existing.zip"), []byte("old"), 0o644); err != nil {
		t.Fatalf("write old existing backup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "recent-existing.zip"), []byte("recent"), 0o644); err != nil {
		t.Fatalf("write recent existing backup: %v", err)
	}
	if err := os.Mkdir(filepath.Join(backupDir, "invalid-dir"), 0o755); err != nil {
		t.Fatalf("create invalid backup dir: %v", err)
	}

	if err := CleanOldBackups(db, 7); err != nil {
		t.Fatalf("clean old backups: %v", err)
	}

	if _, err := os.Stat(filepath.Join(backupDir, "old-existing.zip")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected old existing backup file removed, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(backupDir, "recent-existing.zip")); err != nil {
		t.Fatalf("expected recent existing backup file to remain, got %v", err)
	}
	if info, err := os.Stat(filepath.Join(backupDir, "invalid-dir")); err != nil {
		t.Fatalf("expected invalid directory entry to remain on disk, got %v", err)
	} else if !info.IsDir() {
		t.Fatalf("expected invalid backup path to still be a directory, got %#v", info.Mode())
	}

	for _, removedID := range []string{"old-existing", "old-missing", "recent-missing", "invalid-dir"} {
		if _, err := service.GetBackup(db, removedID); !errors.Is(err, service.ErrNoRecord) {
			t.Fatalf("expected backup record %s removed, got %v", removedID, err)
		}
	}
	if _, err := service.GetBackup(db, "recent-existing"); err != nil {
		t.Fatalf("expected recent existing backup record to remain, got %v", err)
	}
}
