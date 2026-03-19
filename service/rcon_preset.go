package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zaigie/palworld-server-tool/internal/database"
	"go.etcd.io/bbolt"
)

var officialRconPresetCommands = []database.RconCommand{
	{Command: "give", Remark: "给予玩家物品", Placeholder: "{steamUserID} {itemID} {amount}"},
	{Command: "givepal", Remark: "给予玩家帕鲁", Placeholder: "{steamUserID} {palID} {level}"},
	{Command: "giveegg", Remark: "给予玩家帕鲁蛋", Placeholder: "{steamUserID} {eggID} {palID} {level}"},
	{Command: "give_exp", Remark: "给予玩家经验", Placeholder: "{steamUserID} {amount}"},
	{Command: "give_relic", Remark: "给予玩家翠叶鼠雕像", Placeholder: "{steamUserID} {amount}"},
	{Command: "givetechpoints", Remark: "给予玩家科技点", Placeholder: "{steamUserID} {amount}"},
	{Command: "givebosstechpoints", Remark: "给予玩家古代科技点", Placeholder: "{steamUserID} {amount}"},
	{Command: "pgbroadcast", Remark: "游戏内广播", Placeholder: "{message}"},
	{Command: "kick", Remark: "踢出玩家", Placeholder: "{steamUserID}"},
	{Command: "ban", Remark: "封禁玩家", Placeholder: "{steamUserID}"},
	{Command: "getip", Remark: "查询玩家 IP", Placeholder: "{steamUserID}"},
	{Command: "setguildleader", Remark: "转移当前公会会长", Placeholder: "{steamUserID}"},
}

var palDefenderRconPresetCommands = []database.RconCommand{
	{Command: "giveitems", Remark: "PalDefender：批量给予玩家物品", Placeholder: "{steamUserID} {itemID}:{amount}"},
	{Command: "givepal", Remark: "PalDefender：给予玩家帕鲁（数量）", Placeholder: "{steamUserID} {palID} {amount}"},
	{Command: "giveegg", Remark: "PalDefender：给予玩家帕鲁蛋", Placeholder: "{steamUserID} {eggID} {palID} {level}"},
	{Command: "learntech", Remark: "PalDefender：解锁科技", Placeholder: "{steamUserID} {techID}"},
	{Command: "clearinv", Remark: "PalDefender：清空玩家背包", Placeholder: "{steamUserID}"},
	{Command: "deletepals", Remark: "PalDefender：删除玩家全部帕鲁", Placeholder: "{steamUserID} all"},
}

func EnsureDefaultRconCommands(db *bbolt.DB) error {
	_, err := ImportRconPresetGroup(db, "official")
	return err
}

func ImportRconPresetGroup(db *bbolt.DB, name string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "official", "default", "builtin":
		return importUniqueRconCommands(db, officialRconPresetCommands)
	case "paldefender", "pal_guard", "palguard":
		return importUniqueRconCommands(db, palDefenderRconPresetCommands)
	default:
		return 0, fmt.Errorf("unknown rcon preset group: %s", name)
	}
}

func importUniqueRconCommands(db *bbolt.DB, commands []database.RconCommand) (int, error) {
	existing := make(map[string]struct{})
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("rcons"))
		return b.ForEach(func(_, v []byte) error {
			var rcon database.RconCommand
			if err := json.Unmarshal(v, &rcon); err != nil {
				return err
			}
			existing[normalizeRconCommandKey(rcon)] = struct{}{}
			return nil
		})
	})
	if err != nil {
		return 0, err
	}

	imported := 0
	for _, command := range commands {
		key := normalizeRconCommandKey(command)
		if _, ok := existing[key]; ok {
			continue
		}
		if err := AddRconCommand(db, command); err != nil {
			return imported, err
		}
		existing[key] = struct{}{}
		imported++
	}
	return imported, nil
}

func normalizeRconCommandKey(rcon database.RconCommand) string {
	placeholder := strings.Join(strings.Fields(strings.TrimSpace(rcon.Placeholder)), " ")
	return strings.ToLower(strings.TrimSpace(rcon.Command)) + "\x00" + strings.ToLower(placeholder)
}
