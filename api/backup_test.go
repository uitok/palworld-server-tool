package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/service"
)

func TestDownloadBackupRemovesStaleRecordWhenFileMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAPITestDB(t, "backups")
	chdirForTest(t, t.TempDir())

	if err := service.AddBackup(db, database.Backup{BackupId: "backup-1", Path: "missing.zip", SaveTime: time.Now()}); err != nil {
		t.Fatalf("seed backup: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "backup_id", Value: "backup-1"}}
	downloadBackup(ctx)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
	var resp ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ErrorCode != "backup_file_missing" {
		t.Fatalf("expected backup_file_missing, got %#v", resp)
	}
	if _, err := service.GetBackup(db, "backup-1"); !errors.Is(err, service.ErrNoRecord) {
		t.Fatalf("expected stale record to be removed, got %v", err)
	}
}

func TestDeleteBackupRemovesRecordWhenFileAlreadyMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAPITestDB(t, "backups")
	chdirForTest(t, t.TempDir())

	if err := service.AddBackup(db, database.Backup{BackupId: "backup-2", Path: "missing.zip", SaveTime: time.Now()}); err != nil {
		t.Fatalf("seed backup: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "backup_id", Value: "backup-2"}}
	deleteBackup(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if success, ok := resp["success"].(bool); !ok || !success {
		t.Fatalf("expected success response, got %#v", resp)
	}
	if _, err := service.GetBackup(db, "backup-2"); !errors.Is(err, service.ErrNoRecord) {
		t.Fatalf("expected backup record to be removed, got %v", err)
	}
}
