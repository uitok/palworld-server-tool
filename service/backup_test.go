package service

import (
	"errors"
	"testing"
	"time"

	"github.com/zaigie/palworld-server-tool/internal/database"
)

func TestBackupCRUDAndListFilters(t *testing.T) {
	db := newTestDB(t, "backups")

	base := time.Date(2026, 3, 19, 8, 0, 0, 0, time.UTC)
	backups := []database.Backup{
		{BackupId: "backup-2", SaveTime: base.Add(2 * time.Hour), Path: "/tmp/backup-2"},
		{BackupId: "backup-1", SaveTime: base.Add(1 * time.Hour), Path: "/tmp/backup-1"},
		{BackupId: "backup-3", SaveTime: base.Add(3 * time.Hour), Path: "/tmp/backup-3"},
	}

	for _, backup := range backups {
		if err := AddBackup(db, backup); err != nil {
			t.Fatalf("add backup %s: %v", backup.BackupId, err)
		}
	}

	gotBackup, err := GetBackup(db, "backup-2")
	if err != nil {
		t.Fatalf("get backup: %v", err)
	}
	if gotBackup.Path != "/tmp/backup-2" {
		t.Fatalf("unexpected backup path: %s", gotBackup.Path)
	}

	listed, err := ListBackups(db, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("list backups: %v", err)
	}
	if len(listed) != 3 {
		t.Fatalf("expected 3 backups, got %d", len(listed))
	}
	if listed[0].BackupId != "backup-1" || listed[1].BackupId != "backup-2" || listed[2].BackupId != "backup-3" {
		t.Fatalf("expected backups sorted by save time, got %#v", listed)
	}

	filtered, err := ListBackups(db, base.Add(1*time.Hour), base.Add(3*time.Hour))
	if err != nil {
		t.Fatalf("list filtered backups: %v", err)
	}
	if len(filtered) != 1 || filtered[0].BackupId != "backup-2" {
		t.Fatalf("expected only middle backup in exclusive time range, got %#v", filtered)
	}

	if err := DeleteBackup(db, "backup-2"); err != nil {
		t.Fatalf("delete backup: %v", err)
	}
	_, err = GetBackup(db, "backup-2")
	if !errors.Is(err, ErrNoRecord) {
		t.Fatalf("expected ErrNoRecord after delete, got %v", err)
	}
}
