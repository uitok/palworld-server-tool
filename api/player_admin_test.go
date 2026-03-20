package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/internal/tool"
	"github.com/zaigie/palworld-server-tool/service"
)

func TestResolvePlayerActionUserIDPriorityAndValidation(t *testing.T) {
	t.Run("prefers explicit user id before stored steam fallback", func(t *testing.T) {
		db := newAPITestDB(t, "players")
		if err := service.PutPlayers(db, []database.Player{{
			TersePlayer: database.TersePlayer{
				PlayerUid: "uid-1",
				OnlinePlayer: database.OnlinePlayer{
					SteamId: "steam-db",
				},
			},
		}}); err != nil {
			t.Fatalf("seed players: %v", err)
		}

		userID, err := resolvePlayerActionUserID("uid-1", playerActionTarget{UserID: "user-explicit", SteamID: "steam-db"})
		if err != nil {
			t.Fatalf("resolve player action user id: %v", err)
		}
		if userID != "user-explicit" {
			t.Fatalf("expected explicit user id, got %s", userID)
		}
	})

	t.Run("rejects conflicting user id", func(t *testing.T) {
		db := newAPITestDB(t, "players")
		if err := service.PutPlayers(db, []database.Player{{
			TersePlayer: database.TersePlayer{
				PlayerUid: "uid-2",
				OnlinePlayer: database.OnlinePlayer{
					UserId:  "user-db",
					SteamId: "steam-db",
				},
			},
		}}); err != nil {
			t.Fatalf("seed players: %v", err)
		}

		_, err := resolvePlayerActionUserID("uid-2", playerActionTarget{UserID: "user-other"})
		if err == nil || !strings.Contains(err.Error(), "does not match") {
			t.Fatalf("expected mismatch error, got %v", err)
		}
	})

	t.Run("rejects conflicting steam id", func(t *testing.T) {
		db := newAPITestDB(t, "players")
		if err := service.PutPlayers(db, []database.Player{{
			TersePlayer: database.TersePlayer{
				PlayerUid: "uid-3",
				OnlinePlayer: database.OnlinePlayer{
					SteamId: "steam-db",
				},
			},
		}}); err != nil {
			t.Fatalf("seed players: %v", err)
		}

		_, err := resolvePlayerActionUserID("uid-3", playerActionTarget{SteamID: "steam-other"})
		if err == nil || !strings.Contains(err.Error(), "does not match") {
			t.Fatalf("expected steam mismatch error, got %v", err)
		}
	})

	t.Run("falls back to steam prefix for unknown player", func(t *testing.T) {
		_ = newAPITestDB(t, "players")
		userID, err := resolvePlayerActionUserID("missing", playerActionTarget{SteamID: "steam_7656119"})
		if err != nil {
			t.Fatalf("resolve fallback steam id: %v", err)
		}
		if userID != "steam_7656119" {
			t.Fatalf("expected normalized steam user id, got %s", userID)
		}
	})
}

func TestAdjustPlayerItemsRemoveCreatesAuditLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAPITestDB(t, "players", "paldefender_audit_logs")
	if err := service.PutPlayers(db, []database.Player{{
		TersePlayer: database.TersePlayer{
			PlayerUid:      "uid-9",
			Nickname:       "Alice",
			SaveLastOnline: time.Now().UTC().Format(time.RFC3339),
			OnlinePlayer: database.OnlinePlayer{
				UserId:  "user-9",
				SteamId: "steam-9",
			},
		},
	}}); err != nil {
		t.Fatalf("seed players: %v", err)
	}

	originalDelete := palDefenderDeleteItemFunc
	palDefenderDeleteItemFunc = func(userID, itemID, amount string) (string, error) {
		if userID != "user-9" || itemID != "Stone" || amount != "2" {
			t.Fatalf("unexpected delete args: %s %s %s", userID, itemID, amount)
		}
		return "removed", nil
	}
	t.Cleanup(func() { palDefenderDeleteItemFunc = originalDelete })

	body := bytes.NewBufferString(`{"item_id":"Stone","operation":"remove","amount":2}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/player/uid-9/items/adjust", body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "player_uid", Value: "uid-9"}}
	adjustPlayerItems(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	logs, err := service.ListPalDefenderAuditLogs(db, 1)
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(logs) != 1 || logs[0].Action != "adjust-item-remove" || !logs[0].Success {
		t.Fatalf("unexpected audit log: %#v", logs)
	}
	details, ok := logs[0].Details.(map[string]any)
	if !ok || details["item_id"] != "Stone" {
		t.Fatalf("unexpected audit details: %#v", logs[0].Details)
	}
	if result, ok := logs[0].Result.(string); !ok || result != "removed" {
		t.Fatalf("unexpected audit result: %#v", logs[0].Result)
	}
}

func TestGrantPlayerItemsOfflineCreatesAuditLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAPITestDB(t, "players", "paldefender_audit_logs")
	if err := service.PutPlayers(db, []database.Player{{
		TersePlayer: database.TersePlayer{
			PlayerUid: "uid-offline",
			Nickname:  "OfflineAlice",
			OnlinePlayer: database.OnlinePlayer{
				UserId:     "user-offline",
				SteamId:    "steam-offline",
				LastOnline: time.Now().UTC().Add(-2 * livePlayerActionOnlineWindow),
			},
		},
	}}); err != nil {
		t.Fatalf("seed players: %v", err)
	}

	body := bytes.NewBufferString(`{"item_id":"Stone","amount":2}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/player/uid-offline/items/grant", body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "player_uid", Value: "uid-offline"}}
	grantPlayerItems(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
	logs, err := service.ListPalDefenderAuditLogs(db, 1)
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(logs) != 1 || logs[0].Action != "grant-item" || logs[0].Success {
		t.Fatalf("unexpected audit logs: %#v", logs)
	}
	if logs[0].ErrorCode != "player_offline" || logs[0].UserID != "user-offline" {
		t.Fatalf("unexpected audit failure info: %#v", logs[0])
	}
}

func TestGrantPlayerPalPersistsAuditDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAPITestDB(t, "players", "paldefender_audit_logs")
	if err := service.PutPlayers(db, []database.Player{{
		TersePlayer: database.TersePlayer{
			PlayerUid:      "uid-pal",
			Nickname:       "PalAlice",
			SaveLastOnline: time.Now().UTC().Format(time.RFC3339),
			OnlinePlayer: database.OnlinePlayer{
				UserId:     "user-pal",
				SteamId:    "steam-pal",
				LastOnline: time.Now().UTC(),
			},
		},
	}}); err != nil {
		t.Fatalf("seed players: %v", err)
	}

	originalGive := palDefenderGiveFunc
	palDefenderGiveFunc = func(request tool.PalDefenderGiveRequest) (tool.PalDefenderAPIResponse, error) {
		if request.UserID != "user-pal" || len(request.Pals) != 2 || request.Pals[0].PalID != "Lamball" || request.Pals[0].Level != 5 {
			t.Fatalf("unexpected give request: %#v", request)
		}
		return tool.PalDefenderAPIResponse{Errors: 0}, nil
	}
	t.Cleanup(func() { palDefenderGiveFunc = originalGive })

	body := bytes.NewBufferString(`{"pal_id":"Lamball","level":5,"amount":2}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/player/uid-pal/pals/grant", body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "player_uid", Value: "uid-pal"}}
	grantPlayerPal(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	logs, err := service.ListPalDefenderAuditLogs(db, 1)
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(logs) != 1 || logs[0].Action != "grant-pal" || !logs[0].Success {
		t.Fatalf("unexpected audit logs: %#v", logs)
	}
	details, ok := logs[0].Details.(map[string]any)
	if !ok {
		t.Fatalf("unexpected audit details: %#v", logs[0].Details)
	}
	if details["pal_id"] != "Lamball" || int(details["level"].(float64)) != 5 || int(details["amount"].(float64)) != 2 {
		t.Fatalf("unexpected audit details content: %#v", details)
	}
}
