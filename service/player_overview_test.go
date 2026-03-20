package service

import (
	"testing"
	"time"

	"github.com/zaigie/palworld-server-tool/internal/database"
)

func TestListPlayerOverviewsFiltersAndCounts(t *testing.T) {
	db := newTestDB(t, "players", "guilds", "whitelist")
	now := time.Now().UTC()
	if err := PutPlayers(db, []database.Player{
		{
			TersePlayer: database.TersePlayer{
				PlayerUid:      "uid-1",
				Nickname:       "Alice",
				SaveLastOnline: now.Format(time.RFC3339),
				OnlinePlayer: database.OnlinePlayer{
					UserId:      "steam_user-1",
					SteamId:     "steam-1",
					AccountName: "alice-account",
					Level:       40,
				},
			},
			Pals: []*database.Pal{{Type: "Lamball", Level: 10}, {Type: "Foxparks", Level: 12}},
			Items: &database.Items{
				CommonContainerId: []*database.Item{{ItemId: "Stone", StackCount: 12}, {ItemId: "Wood", StackCount: 5}},
			},
		},
		{
			TersePlayer: database.TersePlayer{
				PlayerUid:      "uid-2",
				Nickname:       "Bob",
				SaveLastOnline: now.Add(-2 * time.Hour).Format(time.RFC3339),
				OnlinePlayer: database.OnlinePlayer{
					UserId:  "steam_user-2",
					SteamId: "steam-2",
					Level:   20,
				},
			},
			Pals: []*database.Pal{{Type: "Pengullet", Level: 8}},
			Items: &database.Items{
				CommonContainerId: []*database.Item{{ItemId: "StoneAxe", StackCount: 1}},
			},
		},
	}); err != nil {
		t.Fatalf("seed players: %v", err)
	}
	if err := PutGuilds(db, []database.Guild{{
		Name: "Builders",
		AdminPlayerUid: "uid-1",
		BaseCampLevel: 5,
		Players: []*database.GuildPlayer{{PlayerUid: "uid-1", Nickname: "Alice"}},
	}}); err != nil {
		t.Fatalf("seed guilds: %v", err)
	}
	if err := AddWhitelist(db, database.PlayerW{Name: "Alice", SteamID: "steam-1", PlayerUID: "uid-1"}); err != nil {
		t.Fatalf("seed whitelist: %v", err)
	}

	items, err := ListPlayerOverviews(db, PlayerOverviewFilter{WhitelistOnly: true})
	if err != nil {
		t.Fatalf("list player overviews: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 whitelisted player, got %d", len(items))
	}
	if items[0].PlayerUid != "uid-1" || !items[0].Online || !items[0].Whitelisted {
		t.Fatalf("unexpected summary: %#v", items[0])
	}
	if items[0].ItemCount != 17 || items[0].UniqueItemCount != 2 || items[0].PalCount != 2 {
		t.Fatalf("unexpected counts: %#v", items[0])
	}
	if items[0].Guild == nil || items[0].Guild.Name != "Builders" {
		t.Fatalf("expected guild summary, got %#v", items[0].Guild)
	}

	filtered, err := ListPlayerOverviews(db, PlayerOverviewFilter{Keyword: "bob"})
	if err != nil {
		t.Fatalf("filter by keyword: %v", err)
	}
	if len(filtered) != 1 || filtered[0].PlayerUid != "uid-2" {
		t.Fatalf("unexpected keyword filter result: %#v", filtered)
	}
}

func TestSearchPlayerItemsAndPals(t *testing.T) {
	db := newTestDB(t, "players", "guilds", "whitelist")
	if err := PutPlayers(db, []database.Player{{
		TersePlayer: database.TersePlayer{
			PlayerUid: "uid-1",
			Nickname:  "Alice",
			OnlinePlayer: database.OnlinePlayer{
				UserId:  "steam_user-1",
				SteamId: "steam-1",
			},
		},
		Pals: []*database.Pal{{Type: "Lamball", Nickname: "Fluffy", Level: 10, Skills: []string{"WorkSpeedUp1"}}},
		Items: &database.Items{
			CommonContainerId: []*database.Item{{ItemId: "Stone", StackCount: 8}},
			WeaponLoadOutContainerId: []*database.Item{{ItemId: "StoneBow", StackCount: 1}},
		},
	}}); err != nil {
		t.Fatalf("seed players: %v", err)
	}

	itemHits, err := SearchPlayerItems(db, "stone", "")
	if err != nil {
		t.Fatalf("search player items: %v", err)
	}
	if len(itemHits) != 2 {
		t.Fatalf("expected 2 item hits, got %#v", itemHits)
	}
	if itemHits[0].ItemId != "Stone" {
		t.Fatalf("expected Stone first because count is higher, got %#v", itemHits)
	}

	palHits, err := SearchPlayerPals(db, "lamb", "")
	if err != nil {
		t.Fatalf("search player pals: %v", err)
	}
	if len(palHits) != 1 || palHits[0].PalId != "Lamball" {
		t.Fatalf("unexpected pal hits: %#v", palHits)
	}

	palHits, err = SearchPlayerPals(db, "workspeed", "uid-1")
	if err != nil {
		t.Fatalf("search player pals by skill: %v", err)
	}
	if len(palHits) != 1 || palHits[0].PlayerUid != "uid-1" {
		t.Fatalf("unexpected skill pal hits: %#v", palHits)
	}
}
