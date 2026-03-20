package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/internal/tool"
	"github.com/zaigie/palworld-server-tool/service"
)

func TestGrantPalDefenderBatchIncludesFailureSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAPITestDB(t, "players", "paldefender_audit_logs")
	now := time.Now().UTC()
	if err := service.PutPlayers(db, []database.Player{
		{TersePlayer: database.TersePlayer{PlayerUid: "uid-ok", Nickname: "Alice", SaveLastOnline: now.Format(time.RFC3339), OnlinePlayer: database.OnlinePlayer{UserId: "user-ok", SteamId: "steam-ok"}}},
		{TersePlayer: database.TersePlayer{PlayerUid: "uid-fail", Nickname: "Bob", SaveLastOnline: now.Format(time.RFC3339), OnlinePlayer: database.OnlinePlayer{UserId: "user-fail", SteamId: "steam-fail"}}},
	}); err != nil {
		t.Fatalf("seed players: %v", err)
	}

	originalGive := palDefenderGiveFunc
	palDefenderGiveFunc = func(request tool.PalDefenderGiveRequest) (tool.PalDefenderAPIResponse, error) {
		if request.UserID == "user-fail" {
			response := tool.PalDefenderAPIResponse{Errors: 1, Error: "boom"}
			return response, &tool.PalDefenderAPIError{StatusCode: http.StatusBadGateway, Response: response}
		}
		return tool.PalDefenderAPIResponse{}, nil
	}
	t.Cleanup(func() { palDefenderGiveFunc = originalGive })

	payload, _ := json.Marshal(map[string]any{
		"targets": []map[string]any{{"player_uid": "uid-ok"}, {"player_uid": "uid-fail"}},
		"grant":   map[string]any{"exp": 10},
	})
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/server/paldefender/grant-batch", bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	grantPalDefenderBatch(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp palDefenderBatchGrantResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.RequestedTargetCount != 2 || resp.TargetCount != 2 || resp.SuccessCount != 1 || resp.FailureCount != 1 {
		t.Fatalf("unexpected batch counters: %#v", resp)
	}
	if resp.FailureCodes["paldefender_service_error"] != 1 {
		t.Fatalf("expected failure code summary, got %#v", resp.FailureCodes)
	}
	if resp.DurationMs < 0 || resp.CompletedAt.IsZero() {
		t.Fatalf("expected completion metadata, got %#v", resp)
	}
	logs, err := service.ListPalDefenderAuditLogs(db, 10)
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 audit logs, got %d", len(logs))
	}
}

func TestRecordPalDefenderCommandAuditPersistsDetailsAndResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAPITestDB(t, "players", "paldefender_audit_logs")
	if err := service.PutPlayers(db, []database.Player{{TersePlayer: database.TersePlayer{PlayerUid: "uid-1", Nickname: "Alice", SaveLastOnline: time.Now().UTC().Format(time.RFC3339), OnlinePlayer: database.OnlinePlayer{UserId: "user-1", SteamId: "steam-1"}}}}); err != nil {
		t.Fatalf("seed players: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/player/uid-1/pals/export", nil)
	recordPalDefenderCommandAudit(ctx, "export-pals", playerActionTarget{PlayerUID: "uid-1"}, map[string]any{"scope": "all"}, "ok", nil)

	logs, err := service.ListPalDefenderAuditLogs(db, 1)
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].Action != "export-pals" || logs[0].UserID != "user-1" || logs[0].SteamID != "steam-1" {
		t.Fatalf("unexpected audit identifiers: %#v", logs[0])
	}
	details, ok := logs[0].Details.(map[string]any)
	if !ok || details["scope"] != "all" {
		t.Fatalf("unexpected audit details: %#v", logs[0].Details)
	}
	if result, ok := logs[0].Result.(string); !ok || result != "ok" {
		t.Fatalf("unexpected audit result: %#v", logs[0].Result)
	}

	recordPalDefenderCommandAudit(ctx, "export-pals", playerActionTarget{PlayerUID: "uid-1"}, nil, nil, errors.New("player action user id not found"))
	logs, err = service.ListPalDefenderAuditLogs(db, 2)
	if err != nil {
		t.Fatalf("list audit logs after failure: %v", err)
	}
	if logs[0].Action != "export-pals" || logs[0].ErrorCode != "player_action_user_id_not_found" {
		t.Fatalf("unexpected audit logs after failure: %#v", logs)
	}
}

func TestRecordPalDefenderCommandAuditCapturesPalDefenderAPIError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAPITestDB(t, "players", "paldefender_audit_logs")
	if err := service.PutPlayers(db, []database.Player{{
		TersePlayer: database.TersePlayer{
			PlayerUid:      "uid-err",
			Nickname:       "ErrAlice",
			SaveLastOnline: time.Now().UTC().Format(time.RFC3339),
			OnlinePlayer:   database.OnlinePlayer{UserId: "user-err", SteamId: "steam-err"},
		},
	}}); err != nil {
		t.Fatalf("seed players: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/player/uid-err/items/adjust", nil)
	pdErr := &tool.PalDefenderAPIError{
		StatusCode: http.StatusBadGateway,
		Response:   tool.PalDefenderAPIResponse{Errors: 2, Error: "boom"},
	}
	recordPalDefenderCommandAudit(ctx, "clear-inventory", playerActionTarget{PlayerUID: "uid-err"}, nil, nil, pdErr)

	logs, err := service.ListPalDefenderAuditLogs(db, 1)
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].Action != "clear-inventory" || logs[0].ErrorCode != "paldefender_service_error" || logs[0].PalDefenderErrors != 2 {
		t.Fatalf("unexpected audit log: %#v", logs[0])
	}
	if detail, ok := logs[0].Details.(string); !ok || detail != "boom" {
		t.Fatalf("unexpected audit details: %#v", logs[0].Details)
	}
}
