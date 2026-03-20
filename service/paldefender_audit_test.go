package service

import (
	"testing"
	"time"

	"github.com/zaigie/palworld-server-tool/internal/database"
)

func TestListPalDefenderAuditLogsByFilter(t *testing.T) {
	db := newTestDB(t, "paldefender_audit_logs")
	baseTime := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	seed := []database.PalDefenderAuditLog{
		{ID: "1", CreatedAt: baseTime.Add(1 * time.Minute), Action: "batch-grant", BatchID: "batch-1", PlayerUID: "uid-1", UserID: "user-1", Success: true},
		{ID: "2", CreatedAt: baseTime.Add(2 * time.Minute), Action: "batch-grant", BatchID: "batch-1", PlayerUID: "uid-2", UserID: "user-2", Success: false, ErrorCode: "player_offline"},
		{ID: "3", CreatedAt: baseTime.Add(3 * time.Minute), Action: "export-pals", BatchID: "", PlayerUID: "uid-3", UserID: "user-3", Success: true},
		{ID: "4", CreatedAt: baseTime.Add(4 * time.Minute), Action: "batch-grant-retry", BatchID: "batch-2", PlayerUID: "uid-2", UserID: "user-2", Success: true},
	}
	for _, log := range seed {
		if err := AddPalDefenderAuditLog(db, log); err != nil {
			t.Fatalf("add audit log: %v", err)
		}
	}

	failure := false
	failures, err := ListPalDefenderAuditLogsByFilter(db, PalDefenderAuditLogFilter{Limit: 10, BatchID: "batch-1", Success: &failure})
	if err != nil {
		t.Fatalf("list filtered failures: %v", err)
	}
	if len(failures) != 1 || failures[0].ID != "2" {
		t.Fatalf("unexpected failures: %#v", failures)
	}

	logs, err := ListPalDefenderAuditLogsByFilter(db, PalDefenderAuditLogFilter{Limit: 10, Action: "batch-grant"})
	if err != nil {
		t.Fatalf("list action filtered logs: %v", err)
	}
	if len(logs) != 2 || logs[0].ID != "2" || logs[1].ID != "1" {
		t.Fatalf("unexpected action-filtered logs order: %#v", logs)
	}

	errorLogs, err := ListPalDefenderAuditLogsByFilter(db, PalDefenderAuditLogFilter{Limit: 10, ErrorCode: "player_offline"})
	if err != nil {
		t.Fatalf("list error-code filtered logs: %v", err)
	}
	if len(errorLogs) != 1 || errorLogs[0].ID != "2" {
		t.Fatalf("unexpected error-code logs: %#v", errorLogs)
	}
}
