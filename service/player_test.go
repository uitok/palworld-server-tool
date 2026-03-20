package service

import (
	"errors"
	"testing"
	"time"

	"github.com/zaigie/palworld-server-tool/internal/database"
)

func TestPutPlayersMergesExistingFieldsAndRemovesMissingPlayers(t *testing.T) {
	db := newTestDB(t, "players")

	existingTime := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)
	if err := PutPlayers(db, []database.Player{
		{
			TersePlayer: database.TersePlayer{
				PlayerUid:      "uid-1",
				Nickname:       "Alice",
				SaveLastOnline: existingTime.Format(time.RFC3339),
				OnlinePlayer: database.OnlinePlayer{
					SteamId:   "steam-old",
					Ip:        "10.0.0.1",
					Ping:      12.5,
					LocationX: 1.5,
					LocationY: 2.5,
				},
			},
		},
		{
			TersePlayer: database.TersePlayer{PlayerUid: "uid-stale", Nickname: "Stale"},
		},
	}); err != nil {
		t.Fatalf("seed players: %v", err)
	}

	syncTime := time.Date(2026, 3, 19, 8, 30, 0, 0, time.UTC)
	if err := PutPlayers(db, []database.Player{
		{
			TersePlayer: database.TersePlayer{
				PlayerUid:      "uid-1",
				Nickname:       "Alice Updated",
				SaveLastOnline: syncTime.Format(time.RFC3339),
			},
		},
		{
			TersePlayer: database.TersePlayer{
				PlayerUid: "uid-2",
				Nickname:  "Bob",
				OnlinePlayer: database.OnlinePlayer{
					SteamId: "steam-2",
				},
			},
		},
	}); err != nil {
		t.Fatalf("put players: %v", err)
	}

	player, err := GetPlayer(db, "uid-1")
	if err != nil {
		t.Fatalf("get merged player: %v", err)
	}
	if player.SteamId != "steam-old" {
		t.Fatalf("expected existing steam id to be preserved, got %s", player.SteamId)
	}
	if player.Ip != "10.0.0.1" || player.Ping != 12.5 || player.LocationX != 1.5 || player.LocationY != 2.5 {
		t.Fatalf("expected runtime fields to be preserved, got %#v", player.TersePlayer)
	}
	if !player.LastOnline.Equal(syncTime) {
		t.Fatalf("expected save time to be parsed into LastOnline, got %v", player.LastOnline)
	}

	if _, err := GetPlayer(db, "uid-stale"); !errors.Is(err, ErrNoRecord) {
		t.Fatalf("expected stale player to be removed, got %v", err)
	}
	if _, err := GetPlayer(db, "uid-2"); err != nil {
		t.Fatalf("expected new player to be inserted, got %v", err)
	}
}

func TestPutPlayersOnlineCreatesAndEnrichesPlayers(t *testing.T) {
	db := newTestDB(t, "players")

	if err := PutPlayers(db, []database.Player{{
		TersePlayer: database.TersePlayer{
			PlayerUid: "uid-1",
			Nickname:  "Alice",
			OnlinePlayer: database.OnlinePlayer{
				SteamId:     "000000-placeholder",
				AccountName: "",
				UserId:      "",
			},
		},
	}}); err != nil {
		t.Fatalf("seed players: %v", err)
	}

	before := time.Now()
	if err := PutPlayersOnline(db, []database.OnlinePlayer{
		{
			PlayerUid:     "uid-1",
			UserId:        "user-1",
			SteamId:       "steam-live",
			Nickname:      "Alice",
			AccountName:   "alice-account",
			Ip:            "10.0.0.9",
			Ping:          88.8,
			LocationX:     99.5,
			LocationY:     77.7,
			Level:         30,
			BuildingCount: 12,
		},
		{
			PlayerUid:   "uid-2",
			UserId:      "user-2",
			SteamId:     "steam-2",
			Nickname:    "Bob",
			AccountName: "bob-account",
		},
	}); err != nil {
		t.Fatalf("put online players: %v", err)
	}

	merged, err := GetPlayer(db, "uid-1")
	if err != nil {
		t.Fatalf("get online merged player: %v", err)
	}
	if merged.SteamId != "steam-live" || merged.UserId != "user-1" || merged.AccountName != "alice-account" {
		t.Fatalf("expected identity fields to be enriched, got %#v", merged.TersePlayer)
	}
	if merged.Ip != "10.0.0.9" || merged.Ping != 88.8 || merged.Level != 30 || merged.BuildingCount != 12 {
		t.Fatalf("expected online runtime fields to be updated, got %#v", merged.TersePlayer)
	}
	if merged.LastOnline.Before(before) {
		t.Fatalf("expected LastOnline to be refreshed, got %v", merged.LastOnline)
	}

	created, err := GetPlayer(db, "uid-2")
	if err != nil {
		t.Fatalf("get online-created player: %v", err)
	}
	if created.UserId != "user-2" || created.SteamId != "steam-2" || created.Nickname != "Bob" {
		t.Fatalf("expected online-only player to be created, got %#v", created.TersePlayer)
	}
}

func TestListPlayersSkipsPlaceholderUIDsAndGetPlayerMissing(t *testing.T) {
	db := newTestDB(t, "players")

	if err := PutPlayers(db, []database.Player{
		{TersePlayer: database.TersePlayer{PlayerUid: "uid-1", Nickname: "Alice"}},
		{TersePlayer: database.TersePlayer{PlayerUid: "000000-placeholder", Nickname: "Ghost"}},
	}); err != nil {
		t.Fatalf("put players: %v", err)
	}

	players, err := ListPlayers(db)
	if err != nil {
		t.Fatalf("list players: %v", err)
	}
	if len(players) != 1 || players[0].PlayerUid != "uid-1" {
		t.Fatalf("expected placeholder player to be filtered, got %#v", players)
	}

	if _, err := GetPlayer(db, "missing"); !errors.Is(err, ErrNoRecord) {
		t.Fatalf("expected ErrNoRecord for missing player, got %v", err)
	}
}
