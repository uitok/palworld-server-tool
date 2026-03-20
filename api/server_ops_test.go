package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.etcd.io/bbolt"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/internal/task"
	"github.com/zaigie/palworld-server-tool/internal/tool"
	"github.com/zaigie/palworld-server-tool/service"
)

func TestGetServerOverviewReturnsSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAPITestDB(t, "backups")
	workspace := t.TempDir()
	chdirForTest(t, workspace)

	backupDir := filepath.Join(workspace, "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatalf("mkdir backup dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "latest.zip"), []byte("backup"), 0o644); err != nil {
		t.Fatalf("write backup file: %v", err)
	}
	if err := service.AddBackup(db, database.Backup{BackupId: "backup-1", Path: "latest.zip", SaveTime: time.Now().UTC()}); err != nil {
		t.Fatalf("seed backup: %v", err)
	}

	originalInfo := serverInfoFunc
	originalMetrics := serverMetricsFunc
	originalTaskStatus := taskStatusSnapshotFunc
	originalPalStatus := palDefenderStatusSnapshotFunc
	serverInfoFunc = func() (map[string]string, error) {
		return map[string]string{"version": "0.1.2", "name": "Test Server"}, nil
	}
	serverMetricsFunc = func() (map[string]interface{}, error) {
		return map[string]interface{}{
			"server_fps":         60,
			"current_player_num": 5,
			"server_frame_time":  16.7,
			"max_player_num":     32,
			"uptime":             7200,
			"days":               42,
		}, nil
	}
	taskStatusSnapshotFunc = func() task.TaskStatusSnapshot {
		return task.TaskStatusSnapshot{
			CheckedAt:  time.Now().UTC(),
			PlayerSync: task.TaskRunStatus{Name: task.TaskPlayerSync, Enabled: true, SuccessCount: 3},
			SaveSync:   task.TaskRunStatus{Name: task.TaskSaveSync, Enabled: true, SuccessCount: 2},
			Backup:     task.TaskRunStatus{Name: task.TaskBackup, Enabled: true, SuccessCount: 1},
		}
	}
	palDefenderStatusSnapshotFunc = func() tool.PalDefenderStatus {
		return tool.PalDefenderStatus{Enabled: true, Configured: true, Reachable: true, Healthy: true}
	}
	t.Cleanup(func() {
		serverInfoFunc = originalInfo
		serverMetricsFunc = originalMetrics
		taskStatusSnapshotFunc = originalTaskStatus
		palDefenderStatusSnapshotFunc = originalPalStatus
		viper.Reset()
	})

	viper.Set("rest.address", "http://127.0.0.1:8212")
	viper.Set("rest.username", "admin")
	viper.Set("rest.password", "secret")
	viper.Set("task.sync_interval", 60)
	viper.Set("save.sync_interval", 300)
	viper.Set("save.backup_interval", 600)
	viper.Set("save.path", workspace)
	viper.Set("rcon.address", "127.0.0.1:25575")
	viper.Set("rcon.password", "secret")
	viper.Set("paldefender.enabled", true)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/server/overview", nil)
	ctx.Set("version", "vtest")
	getServerOverview(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp ServerOverviewResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode overview response: %v", err)
	}
	if !resp.Success || resp.PanelVersion != "vtest" {
		t.Fatalf("unexpected overview header: %#v", resp)
	}
	if resp.Server == nil || resp.Server.Name != "Test Server" || resp.Metrics == nil || resp.Metrics.CurrentPlayerNum != 5 {
		t.Fatalf("unexpected server overview payload: %#v", resp)
	}
	if !resp.Capabilities.RestEnabled || !resp.Capabilities.BackupEnabled || !resp.Dependencies.REST.Reachable {
		t.Fatalf("unexpected capabilities/dependencies: %#v", resp)
	}
	if resp.LatestBackup == nil || !resp.LatestBackup.FileExists || resp.LatestBackup.Path != "latest.zip" {
		t.Fatalf("unexpected latest backup summary: %#v", resp.LatestBackup)
	}
}

func TestSyncDataReturnsOperationResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_ = newAPITestDB(t, "players")

	originalRunPlayerSync := runPlayerSyncNowFunc
	runPlayerSyncNowFunc = func(_ *bbolt.DB) (int, int64, string, error) { return 4, 87, "", nil }
	t.Cleanup(func() { runPlayerSyncNowFunc = originalRunPlayerSync })

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/server/sync", strings.NewReader(`{"from":"rest"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	syncData(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp ServerOperationResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode sync response: %v", err)
	}
	if !resp.Success || resp.Action != "sync" || resp.Task != task.TaskPlayerSync || resp.DurationMs != 87 {
		t.Fatalf("unexpected sync response: %#v", resp)
	}
}

func TestCreateBackupNowReturnsOperationResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_ = newAPITestDB(t, "backups")

	originalRunBackup := runBackupNowFunc
	runBackupNowFunc = func(_ *bbolt.DB) (database.Backup, int64, string, error) {
		return database.Backup{BackupId: "backup-9", Path: "backup-9.zip", SaveTime: time.Now().UTC()}, 125, "", nil
	}
	t.Cleanup(func() { runBackupNowFunc = originalRunBackup })

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/server/backup", nil)
	createBackupNow(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp ServerOperationResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode backup response: %v", err)
	}
	if !resp.Success || resp.Action != "backup" || resp.Task != task.TaskBackup || resp.DurationMs != 125 {
		t.Fatalf("unexpected backup response: %#v", resp)
	}
}
