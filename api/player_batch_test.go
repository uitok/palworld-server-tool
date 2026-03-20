package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/service"
)

func TestBatchPlayerActionWhitelistAddPartialFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAPITestDB(t, "players", "whitelist")
	if err := service.PutPlayers(db, []database.Player{{
		TersePlayer: database.TersePlayer{
			PlayerUid: "uid-1",
			Nickname:  "Alice",
			OnlinePlayer: database.OnlinePlayer{
				SteamId: "steam-1",
			},
		},
	}}); err != nil {
		t.Fatalf("seed players: %v", err)
	}
	body := bytes.NewBufferString(`{"action":"whitelist_add","player_uids":["uid-1","missing"]}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/player/batch", body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	batchPlayerAction(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var response BatchPlayerActionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Success || response.Succeeded != 1 || response.Failed != 1 {
		t.Fatalf("unexpected response summary: %#v", response)
	}
	whitelist, err := service.ListWhitelist(db)
	if err != nil {
		t.Fatalf("list whitelist: %v", err)
	}
	if len(whitelist) != 1 || whitelist[0].PlayerUID != "uid-1" {
		t.Fatalf("expected uid-1 in whitelist, got %#v", whitelist)
	}
}

func TestBatchPlayerActionKickUsesResolvedUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAPITestDB(t, "players")
	if err := service.PutPlayers(db, []database.Player{{
		TersePlayer: database.TersePlayer{
			PlayerUid: "uid-2",
			Nickname:  "Bob",
			OnlinePlayer: database.OnlinePlayer{
				UserId:  "steam_user-2",
				SteamId: "steam-2",
			},
		},
	}}); err != nil {
		t.Fatalf("seed players: %v", err)
	}
	originalKick := batchKickPlayerFunc
	batchKickPlayerFunc = func(userID string) error {
		if userID != "steam_user-2" {
			t.Fatalf("unexpected user id: %s", userID)
		}
		return nil
	}
	t.Cleanup(func() { batchKickPlayerFunc = originalKick })
	body := bytes.NewBufferString(`{"action":"kick","player_uids":["uid-2","uid-2"]}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/player/batch", body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	batchPlayerAction(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var response BatchPlayerActionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.Requested != 1 || response.Succeeded != 1 {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestBatchPlayerActionRejectsInvalidActionAndEmptyTargets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_ = newAPITestDB(t, "players")
	t.Run("invalid action", func(t *testing.T) {
		body := bytes.NewBufferString(`{"action":"noop","player_uids":["uid-1"]}`)
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/player/batch", body)
		ctx.Request.Header.Set("Content-Type", "application/json")
		batchPlayerAction(ctx)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})
	t.Run("empty targets", func(t *testing.T) {
		body := bytes.NewBufferString(`{"action":"kick","player_uids":[]}`)
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/player/batch", body)
		ctx.Request.Header.Set("Content-Type", "application/json")
		batchPlayerAction(ctx)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})
}
