package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/internal/tool"
	"github.com/zaigie/palworld-server-tool/service"
)

func TestListAndExportPalDefenderAuditLogsWithFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAPITestDB(t, "paldefender_audit_logs")
	base := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	seed := []database.PalDefenderAuditLog{
		{ID: "1", CreatedAt: base.Add(1 * time.Minute), Action: "batch-grant", BatchID: "batch-1", PlayerUID: "uid-1", Success: true},
		{ID: "2", CreatedAt: base.Add(2 * time.Minute), Action: "batch-grant", BatchID: "batch-1", PlayerUID: "uid-2", Success: false, ErrorCode: "player_offline"},
		{ID: "3", CreatedAt: base.Add(3 * time.Minute), Action: "export-pals", PlayerUID: "uid-3", Success: true},
	}
	for _, log := range seed {
		if err := service.AddPalDefenderAuditLog(db, log); err != nil {
			t.Fatalf("seed audit logs: %v", err)
		}
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/server/paldefender/audit?batch_id=batch-1&success=false", nil)
	listPalDefenderAuditLogs(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var logs []database.PalDefenderAuditLog
	if err := json.Unmarshal(recorder.Body.Bytes(), &logs); err != nil {
		t.Fatalf("decode logs: %v", err)
	}
	if len(logs) != 1 || logs[0].ID != "2" {
		t.Fatalf("unexpected filtered logs: %#v", logs)
	}

	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/server/paldefender/audit/export?action=batch-grant", nil)
	exportPalDefenderAuditLogs(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Content-Disposition") == "" {
		t.Fatalf("expected export content disposition header")
	}
}

func TestRetryPalDefenderBatchRetriesFailedTargetsOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAPITestDB(t, "players", "paldefender_audit_logs")
	now := time.Now().UTC()
	if err := service.PutPlayers(db, []database.Player{
		{TersePlayer: database.TersePlayer{PlayerUid: "uid-ok", Nickname: "Alice", SaveLastOnline: now.Format(time.RFC3339), OnlinePlayer: database.OnlinePlayer{UserId: "user-ok", SteamId: "steam-ok"}}},
		{TersePlayer: database.TersePlayer{PlayerUid: "uid-fail", Nickname: "Bob", SaveLastOnline: now.Format(time.RFC3339), OnlinePlayer: database.OnlinePlayer{UserId: "user-fail", SteamId: "steam-fail"}}},
	}); err != nil {
		t.Fatalf("seed players: %v", err)
	}
	seed := []database.PalDefenderAuditLog{
		{ID: "1", CreatedAt: now.Add(-2 * time.Minute), Action: "batch-grant", BatchID: "batch-1", PlayerUID: "uid-ok", UserID: "user-ok", SteamID: "steam-ok", Nickname: "Alice", PresetNames: []string{"starter"}, Grant: map[string]any{"exp": 10}, Success: true},
		{ID: "2", CreatedAt: now.Add(-1 * time.Minute), Action: "batch-grant", BatchID: "batch-1", PlayerUID: "uid-fail", UserID: "user-fail", SteamID: "steam-fail", Nickname: "Bob", PresetNames: []string{"starter"}, Grant: map[string]any{"exp": 10}, Success: false, ErrorCode: "player_offline"},
	}
	for _, log := range seed {
		if err := service.AddPalDefenderAuditLog(db, log); err != nil {
			t.Fatalf("seed audit logs: %v", err)
		}
	}

	called := []string{}
	originalGive := palDefenderGiveFunc
	palDefenderGiveFunc = func(request tool.PalDefenderGiveRequest) (tool.PalDefenderAPIResponse, error) {
		called = append(called, request.UserID)
		return tool.PalDefenderAPIResponse{}, nil
	}
	t.Cleanup(func() { palDefenderGiveFunc = originalGive })

	body := bytes.NewBufferString(`{"batch_id":"batch-1","failed_only":true}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/server/paldefender/grant-batch/retry", body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	retryPalDefenderBatch(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(called) != 1 || called[0] != "user-fail" {
		t.Fatalf("unexpected retry targets: %#v", called)
	}
	var response palDefenderBatchGrantResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode retry response: %v", err)
	}
	if response.SourceBatchID != "batch-1" || response.RequestedTargetCount != 1 || response.SuccessCount != 1 || response.FailureCount != 0 {
		t.Fatalf("unexpected retry response: %#v", response)
	}
	logs, err := service.ListPalDefenderAuditLogsByFilter(db, service.PalDefenderAuditLogFilter{Limit: 10, Action: "batch-grant-retry"})
	if err != nil {
		t.Fatalf("list retry logs: %v", err)
	}
	if len(logs) != 1 || logs[0].PlayerUID != "uid-fail" {
		t.Fatalf("unexpected retry logs: %#v", logs)
	}
}
