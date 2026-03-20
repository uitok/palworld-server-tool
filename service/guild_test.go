package service

import (
	"errors"
	"testing"

	"github.com/zaigie/palworld-server-tool/internal/database"
)

func TestGuildPutListAndGetByPlayerUID(t *testing.T) {
	db := newTestDB(t, "guilds")

	guilds := []database.Guild{
		{
			Name:           "Alpha",
			BaseCampLevel:  10,
			AdminPlayerUid: "admin-alpha",
			Players: []*database.GuildPlayer{
				{PlayerUid: "player-a1", Nickname: "Alice"},
				{PlayerUid: "player-a2", Nickname: "Ares"},
			},
		},
		{
			Name:           "Beta",
			BaseCampLevel:  8,
			AdminPlayerUid: "admin-beta",
			Players: []*database.GuildPlayer{
				{PlayerUid: "player-b1", Nickname: "Bob"},
			},
		},
	}

	if err := PutGuilds(db, guilds); err != nil {
		t.Fatalf("put guilds: %v", err)
	}

	listed, err := ListGuilds(db)
	if err != nil {
		t.Fatalf("list guilds: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected 2 guilds, got %d", len(listed))
	}

	names := map[string]bool{}
	for _, guild := range listed {
		names[guild.Name] = true
	}
	if !names["Alpha"] || !names["Beta"] {
		t.Fatalf("unexpected listed guilds: %#v", listed)
	}

	got, err := GetGuild(db, "player-a2")
	if err != nil {
		t.Fatalf("get guild by player uid: %v", err)
	}
	if got.Name != "Alpha" || got.AdminPlayerUid != "admin-alpha" {
		t.Fatalf("unexpected guild result: %#v", got)
	}

	_, err = GetGuild(db, "missing-player")
	if !errors.Is(err, ErrNoRecord) {
		t.Fatalf("expected ErrNoRecord for missing player, got %v", err)
	}
}
