package service

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/zaigie/palworld-server-tool/internal/database"
	"go.etcd.io/bbolt"
)

const playerOverviewOnlineWindow = 80 * time.Second

type PlayerOverviewFilter struct {
	Keyword       string
	OnlineOnly    bool
	WhitelistOnly bool
	GuildOnly     bool
}

func ListPlayerOverviews(db *bbolt.DB, filter PlayerOverviewFilter) ([]database.PlayerOverviewSummary, error) {
	players, err := listPlayerRecords(db)
	if err != nil {
		return nil, err
	}
	whitelist, guilds, err := buildPlayerOverviewRelations(db)
	if err != nil {
		return nil, err
	}
	keyword := strings.ToLower(strings.TrimSpace(filter.Keyword))
	items := make([]database.PlayerOverviewSummary, 0, len(players))
	for _, player := range players {
		summary := buildPlayerOverviewSummary(player, whitelist, guilds)
		if !matchesPlayerOverviewFilter(summary, keyword, filter) {
			continue
		}
		items = append(items, summary)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Online != items[j].Online {
			return items[i].Online
		}
		if !items[i].LastOnline.Equal(items[j].LastOnline) {
			return items[i].LastOnline.After(items[j].LastOnline)
		}
		return items[i].Nickname < items[j].Nickname
	})
	return items, nil
}

func GetPlayerOverview(db *bbolt.DB, playerUID string) (database.PlayerOverview, error) {
	player, err := GetPlayer(db, playerUID)
	if err != nil {
		return database.PlayerOverview{}, err
	}
	whitelist, guilds, err := buildPlayerOverviewRelations(db)
	if err != nil {
		return database.PlayerOverview{}, err
	}
	return database.PlayerOverview{
		Summary: buildPlayerOverviewSummary(player, whitelist, guilds),
		Player:   player,
	}, nil
}

func SearchPlayerItems(db *bbolt.DB, keyword string, playerUID string) ([]database.PlayerItemSearchHit, error) {
	players, err := loadPlayersForSearch(db, playerUID)
	if err != nil {
		return nil, err
	}
	whitelist, guilds, err := buildPlayerOverviewRelations(db)
	if err != nil {
		return nil, err
	}
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	hits := make([]database.PlayerItemSearchHit, 0)
	for _, player := range players {
		summary := buildPlayerOverviewSummary(player, whitelist, guilds)
		if player.Items == nil {
			continue
		}
		for container, items := range inventoryContainers(player.Items) {
			for _, item := range items {
				if item == nil {
					continue
				}
				if keyword != "" && !strings.Contains(strings.ToLower(item.ItemId), keyword) {
					continue
				}
				hit := database.PlayerItemSearchHit{
					PlayerUid:   player.PlayerUid,
					Nickname:    player.Nickname,
					UserId:      player.UserId,
					SteamId:     player.SteamId,
					Online:      summary.Online,
					Whitelisted: summary.Whitelisted,
					ItemId:      item.ItemId,
					Container:   container,
					StackCount:  item.StackCount,
					PlayerLevel: player.Level,
				}
				if summary.Guild != nil {
					hit.GuildName = summary.Guild.Name
				}
				hits = append(hits, hit)
			}
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].StackCount != hits[j].StackCount {
			return hits[i].StackCount > hits[j].StackCount
		}
		if hits[i].Nickname != hits[j].Nickname {
			return hits[i].Nickname < hits[j].Nickname
		}
		if hits[i].ItemId != hits[j].ItemId {
			return hits[i].ItemId < hits[j].ItemId
		}
		return hits[i].Container < hits[j].Container
	})
	return hits, nil
}

func SearchPlayerPals(db *bbolt.DB, keyword string, playerUID string) ([]database.PlayerPalSearchHit, error) {
	players, err := loadPlayersForSearch(db, playerUID)
	if err != nil {
		return nil, err
	}
	whitelist, guilds, err := buildPlayerOverviewRelations(db)
	if err != nil {
		return nil, err
	}
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	hits := make([]database.PlayerPalSearchHit, 0)
	for _, player := range players {
		summary := buildPlayerOverviewSummary(player, whitelist, guilds)
		for _, pal := range player.Pals {
			if pal == nil {
				continue
			}
			if keyword != "" && !matchesPalKeyword(pal, keyword) {
				continue
			}
			hit := database.PlayerPalSearchHit{
				PlayerUid:   player.PlayerUid,
				Nickname:    player.Nickname,
				UserId:      player.UserId,
				SteamId:     player.SteamId,
				Online:      summary.Online,
				Whitelisted: summary.Whitelisted,
				PalId:       pal.Type,
				PalNickname: pal.Nickname,
				Level:       pal.Level,
				Gender:      pal.Gender,
				IsLucky:     pal.IsLucky,
				IsBoss:      pal.IsBoss,
				Skills:      pal.Skills,
			}
			if summary.Guild != nil {
				hit.GuildName = summary.Guild.Name
			}
			hits = append(hits, hit)
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Level != hits[j].Level {
			return hits[i].Level > hits[j].Level
		}
		if hits[i].Nickname != hits[j].Nickname {
			return hits[i].Nickname < hits[j].Nickname
		}
		return hits[i].PalId < hits[j].PalId
	})
	return hits, nil
}

func listPlayerRecords(db *bbolt.DB) ([]database.Player, error) {
	players := make([]database.Player, 0)
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("players"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			if strings.Contains(string(k), "000000") {
				return nil
			}
			var player database.Player
			if err := json.Unmarshal(v, &player); err != nil {
				return err
			}
			players = append(players, player)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return players, nil
}

func loadPlayersForSearch(db *bbolt.DB, playerUID string) ([]database.Player, error) {
	if strings.TrimSpace(playerUID) == "" {
		return listPlayerRecords(db)
	}
	player, err := GetPlayer(db, strings.TrimSpace(playerUID))
	if err != nil {
		return nil, err
	}
	return []database.Player{player}, nil
}

func buildPlayerOverviewRelations(db *bbolt.DB) (map[string]struct{}, map[string]database.PlayerGuildSummary, error) {
	whitelist := map[string]struct{}{}
	items, err := ListWhitelist(db)
	if err != nil {
		return nil, nil, err
	}
	for _, item := range items {
		if item.PlayerUID != "" {
			whitelist["uid:"+item.PlayerUID] = struct{}{}
		}
		if item.SteamID != "" {
			whitelist["steam:"+item.SteamID] = struct{}{}
		}
	}
	guildMap := map[string]database.PlayerGuildSummary{}
	guilds, err := ListGuilds(db)
	if err != nil {
		return nil, nil, err
	}
	for _, guild := range guilds {
		summary := database.PlayerGuildSummary{
			Name:           guild.Name,
			AdminPlayerUid: guild.AdminPlayerUid,
			BaseCampLevel:  guild.BaseCampLevel,
			MemberCount:    len(guild.Players),
		}
		for _, player := range guild.Players {
			if player == nil || strings.TrimSpace(player.PlayerUid) == "" {
				continue
			}
			guildMap[player.PlayerUid] = summary
		}
	}
	return whitelist, guildMap, nil
}

func buildPlayerOverviewSummary(player database.Player, whitelist map[string]struct{}, guilds map[string]database.PlayerGuildSummary) database.PlayerOverviewSummary {
	summary := database.PlayerOverviewSummary{
		PlayerUid:      player.PlayerUid,
		Nickname:       player.Nickname,
		UserId:         player.UserId,
		SteamId:        player.SteamId,
		AccountName:    player.AccountName,
		Level:          player.Level,
		LastOnline:     player.LastOnline,
		SaveLastOnline: player.SaveLastOnline,
		Online:         !player.LastOnline.IsZero() && time.Since(player.LastOnline) <= playerOverviewOnlineWindow,
		BuildingCount:  player.BuildingCount,
		LocationX:      player.LocationX,
		LocationY:      player.LocationY,
		PalCount:       len(player.Pals),
	}
	if _, ok := whitelist["uid:"+player.PlayerUid]; ok {
		summary.Whitelisted = true
	} else if player.SteamId != "" {
		_, summary.Whitelisted = whitelist["steam:"+player.SteamId]
	}
	if guild, ok := guilds[player.PlayerUid]; ok {
		g := guild
		summary.Guild = &g
	}
	itemCount := 0
	uniqueItems := map[string]struct{}{}
	for _, items := range inventoryContainers(player.Items) {
		for _, item := range items {
			if item == nil {
				continue
			}
			itemCount += int(item.StackCount)
			if item.ItemId != "" {
				uniqueItems[item.ItemId] = struct{}{}
			}
		}
	}
	summary.ItemCount = itemCount
	summary.UniqueItemCount = len(uniqueItems)
	return summary
}

func inventoryContainers(items *database.Items) map[string][]*database.Item {
	if items == nil {
		return map[string][]*database.Item{}
	}
	return map[string][]*database.Item{
		"common":    items.CommonContainerId,
		"dropslot":  items.DropSlotContainerId,
		"essential": items.EssentialContainerId,
		"food":      items.FoodEquipContainerId,
		"armor":     items.PlayerEquipArmorContainerId,
		"weapons":   items.WeaponLoadOutContainerId,
	}
}

func matchesPlayerOverviewFilter(summary database.PlayerOverviewSummary, keyword string, filter PlayerOverviewFilter) bool {
	if filter.OnlineOnly && !summary.Online {
		return false
	}
	if filter.WhitelistOnly && !summary.Whitelisted {
		return false
	}
	if filter.GuildOnly && summary.Guild == nil {
		return false
	}
	if keyword == "" {
		return true
	}
	fields := []string{summary.PlayerUid, summary.Nickname, summary.UserId, summary.SteamId, summary.AccountName}
	if summary.Guild != nil {
		fields = append(fields, summary.Guild.Name)
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), keyword) {
			return true
		}
	}
	return false
}

func matchesPalKeyword(pal *database.Pal, keyword string) bool {
	if strings.Contains(strings.ToLower(pal.Type), keyword) || strings.Contains(strings.ToLower(pal.Nickname), keyword) {
		return true
	}
	for _, skill := range pal.Skills {
		if strings.Contains(strings.ToLower(skill), keyword) {
			return true
		}
	}
	return false
}
