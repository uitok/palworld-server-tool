package service

import (
	"errors"
	"testing"

	"github.com/zaigie/palworld-server-tool/internal/database"
)

func TestRconCommandCRUD(t *testing.T) {
	db := newTestDB(t, "rcons")

	original := database.RconCommand{
		Command:     "Broadcast hello",
		Placeholder: "message",
		Remark:      "test command",
	}
	if err := AddRconCommand(db, original); err != nil {
		t.Fatalf("add rcon command: %v", err)
	}

	listed, err := ListRconCommands(db)
	if err != nil {
		t.Fatalf("list rcon commands: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 command, got %d", len(listed))
	}
	if listed[0].Command != original.Command || listed[0].UUID == "" {
		t.Fatalf("unexpected listed command: %#v", listed[0])
	}

	got, err := GetRconCommand(db, listed[0].UUID)
	if err != nil {
		t.Fatalf("get rcon command: %v", err)
	}
	if got.Remark != original.Remark {
		t.Fatalf("unexpected rcon remark: %s", got.Remark)
	}

	updated := database.RconCommand{
		Command:     "Broadcast updated",
		Placeholder: "body",
		Remark:      "updated command",
	}
	if err := PutRconCommand(db, listed[0].UUID, updated); err != nil {
		t.Fatalf("put rcon command: %v", err)
	}

	got, err = GetRconCommand(db, listed[0].UUID)
	if err != nil {
		t.Fatalf("get updated rcon command: %v", err)
	}
	if got.Command != updated.Command || got.Placeholder != updated.Placeholder || got.Remark != updated.Remark {
		t.Fatalf("unexpected updated command: %#v", got)
	}

	if err := RemoveRconCommand(db, listed[0].UUID); err != nil {
		t.Fatalf("remove rcon command: %v", err)
	}
	_, err = GetRconCommand(db, listed[0].UUID)
	if !errors.Is(err, ErrNoRecord) {
		t.Fatalf("expected ErrNoRecord after remove, got %v", err)
	}
}
