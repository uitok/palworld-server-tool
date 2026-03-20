package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/service"
)

func TestListPlayerOverviewsMasksSensitiveFieldsWhenLoggedOut(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAPITestDB(t, "players", "guilds", "whitelist")
	if err := service.PutPlayers(db, []database.Player{{
		TersePlayer: database.TersePlayer{
			PlayerUid:      "uid-1",
			Nickname:       "Alice",
			SaveLastOnline: time.Now().UTC().Format(time.RFC3339),
			OnlinePlayer: database.OnlinePlayer{
				UserId:  "steam_user-1",
				SteamId: "steam-1",
			},
		},
		Items: &database.Items{CommonContainerId: []*database.Item{{ItemId: "Stone", StackCount: 3}}},
		Pals:  []*database.Pal{{Type: "Lamball", Level: 10}},
	}}); err != nil {
		t.Fatalf("seed players: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/player/overview", nil)
	ctx.Set("loggedIn", false)
	listPlayerOverviews(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var response PlayerOverviewListResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("expected 1 item, got %#v", response.Items)
	}
	if response.Items[0].SteamId != "" || response.Items[0].UserId != "steam_" {
		t.Fatalf("expected masked identifiers, got %#v", response.Items[0])
	}
}

func TestGetPlayerOverviewDetailAndSearches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAPITestDB(t, "players", "guilds", "whitelist")
	if err := service.PutPlayers(db, []database.Player{{
		TersePlayer: database.TersePlayer{
			PlayerUid: "uid-1",
			Nickname:  "Alice",
			OnlinePlayer: database.OnlinePlayer{
				UserId:  "steam_user-1",
				SteamId: "steam-1",
			},
		},
		Items: &database.Items{CommonContainerId: []*database.Item{{ItemId: "Stone", StackCount: 5}}},
		Pals:  []*database.Pal{{Type: "Lamball", Nickname: "Fluffy", Level: 12, Skills: []string{"WorkSpeedUp1"}}},
	}}); err != nil {
		t.Fatalf("seed players: %v", err)
	}
	if err := service.PutGuilds(db, []database.Guild{{
		Name: "Builders",
		AdminPlayerUid: "uid-1",
		BaseCampLevel: 4,
		Players: []*database.GuildPlayer{{PlayerUid: "uid-1", Nickname: "Alice"}},
	}}); err != nil {
		t.Fatalf("seed guilds: %v", err)
	}

	t.Run("detail", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/player/uid-1/overview", nil)
		ctx.Params = gin.Params{{Key: "player_uid", Value: "uid-1"}}
		ctx.Set("loggedIn", true)
		getPlayerOverviewDetail(ctx)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
		}
		var response PlayerOverviewDetailResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode detail response: %v", err)
		}
		if response.Overview.Summary.Guild == nil || response.Overview.Summary.Guild.Name != "Builders" {
			t.Fatalf("unexpected guild summary: %#v", response.Overview.Summary)
		}
		if response.Overview.Summary.PalCount != 1 || response.Overview.Summary.ItemCount != 5 {
			t.Fatalf("unexpected counts: %#v", response.Overview.Summary)
		}
	})

	t.Run("item search requires keyword", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/player/search/items", nil)
		searchPlayerItems(ctx)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("searches", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/player/search/items?keyword=stone", nil)
		searchPlayerItems(ctx)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
		}
		var itemResponse PlayerItemSearchResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &itemResponse); err != nil {
			t.Fatalf("decode item response: %v", err)
		}
		if len(itemResponse.Items) != 1 || itemResponse.Items[0].ItemId != "Stone" {
			t.Fatalf("unexpected item response: %#v", itemResponse.Items)
		}

		recorder = httptest.NewRecorder()
		ctx, _ = gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/player/search/pals?keyword=fluff", nil)
		searchPlayerPals(ctx)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
		}
		var palResponse PlayerPalSearchResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &palResponse); err != nil {
			t.Fatalf("decode pal response: %v", err)
		}
		if len(palResponse.Items) != 1 || palResponse.Items[0].PalNickname != "Fluffy" {
			t.Fatalf("unexpected pal response: %#v", palResponse.Items)
		}
	})
}
