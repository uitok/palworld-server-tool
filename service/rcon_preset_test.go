package service

import (
	"strings"
	"testing"

	"github.com/zaigie/palworld-server-tool/internal/database"
)

func TestImportRconPresetGroupImportsAndDeduplicates(t *testing.T) {
	db := newTestDB(t, "rcons")

	if err := AddRconCommand(db, database.RconCommand{
		Command:     " GIVE ",
		Placeholder: " {steamUserID}   {itemID}   {amount} ",
		Remark:      "preseeded duplicate",
	}); err != nil {
		t.Fatalf("seed duplicate command: %v", err)
	}

	imported, err := ImportRconPresetGroup(db, "official")
	if err != nil {
		t.Fatalf("import official preset group: %v", err)
	}
	if imported != len(officialRconPresetCommands)-1 {
		t.Fatalf("expected %d imported official commands, got %d", len(officialRconPresetCommands)-1, imported)
	}

	importedAgain, err := ImportRconPresetGroup(db, "default")
	if err != nil {
		t.Fatalf("re-import official preset alias: %v", err)
	}
	if importedAgain != 0 {
		t.Fatalf("expected alias import to dedupe everything, got %d", importedAgain)
	}

	importedPalDefender, err := ImportRconPresetGroup(db, "palguard")
	if err != nil {
		t.Fatalf("import paldefender preset alias: %v", err)
	}
	if importedPalDefender != len(palDefenderRconPresetCommands)-1 {
		t.Fatalf("expected %d paldefender commands after dedupe, got %d", len(palDefenderRconPresetCommands)-1, importedPalDefender)
	}

	listed, err := ListRconCommands(db)
	if err != nil {
		t.Fatalf("list rcon commands: %v", err)
	}
	wantTotal := len(officialRconPresetCommands) + len(palDefenderRconPresetCommands) - 1
	if len(listed) != wantTotal {
		t.Fatalf("expected %d total preset commands, got %d", wantTotal, len(listed))
	}
}

func TestImportRconPresetGroupRejectsUnknownGroup(t *testing.T) {
	db := newTestDB(t, "rcons")

	_, err := ImportRconPresetGroup(db, "unknown-group")
	if err == nil || !strings.Contains(err.Error(), "unknown rcon preset group") {
		t.Fatalf("expected unknown preset group error, got %v", err)
	}
}
