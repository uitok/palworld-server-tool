package config

import "testing"

func TestValidateAllowsMinimalConfigWithoutSync(t *testing.T) {
	var conf Config
	conf.Web.Password = "devpass"
	conf.Web.Port = 8080
	conf.Task.SyncInterval = 0
	conf.Save.SyncInterval = 0
	conf.Save.BackupInterval = 0
	conf.Save.BackupKeepDays = 7
	conf.Rest.Timeout = 5
	conf.PalDefender.Timeout = 5

	if err := Validate(&conf); err != nil {
		t.Fatalf("expected config to pass validation, got %v", err)
	}
}

func TestValidateAggregatesKeyIssues(t *testing.T) {
	var conf Config
	conf.Web.Port = -1
	conf.Task.SyncInterval = 60
	conf.Rest.Address = "127.0.0.1:8212"
	conf.Rest.Timeout = 0
	conf.Save.Path = placeholderSavePath
	conf.Save.DecodePath = "/tmp/definitely-missing-sav-cli"
	conf.Save.SyncInterval = 120
	conf.Save.BackupInterval = 14400
	conf.PalDefender.Timeout = 5

	err := Validate(&conf)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if len(validationErr.Issues) < 5 {
		t.Fatalf("expected aggregated issues, got %d", len(validationErr.Issues))
	}
	fields := map[string]bool{}
	for _, issue := range validationErr.Issues {
		fields[issue.Field] = true
	}
	for _, field := range []string{"web.password", "web.port", "rest.address", "rest.password", "save.path", "save.decode_path"} {
		if !fields[field] {
			t.Fatalf("expected issue for %s, got %#v", field, validationErr.Issues)
		}
	}
}

func TestValidatePalDefenderRequirements(t *testing.T) {
	var conf Config
	conf.Web.Password = "devpass"
	conf.Web.Port = 8080
	conf.Save.SyncInterval = 0
	conf.Save.BackupInterval = 0
	conf.PalDefender.Enabled = true
	conf.PalDefender.Timeout = 0

	err := Validate(&conf)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	fields := map[string]bool{}
	for _, issue := range validationErr.Issues {
		fields[issue.Field] = true
	}
	for _, field := range []string{"paldefender.address", "paldefender.auth_key", "paldefender.timeout"} {
		if !fields[field] {
			t.Fatalf("expected issue for %s, got %#v", field, validationErr.Issues)
		}
	}
}

func TestValidateRconRequirementsWhenConfigured(t *testing.T) {
	var conf Config
	conf.Web.Password = "devpass"
	conf.Web.Port = 8080
	conf.Save.SyncInterval = 0
	conf.Save.BackupInterval = 0
	conf.Rcon.Address = "127.0.0.1"
	conf.Rcon.UseBase64 = true
	err := Validate(&conf)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	fields := map[string]bool{}
	for _, issue := range validationErr.Issues {
		fields[issue.Field] = true
	}
	for _, field := range []string{"rcon.address", "rcon.password", "rcon.timeout"} {
		if !fields[field] {
			t.Fatalf("expected issue for %s, got %#v", field, validationErr.Issues)
		}
	}
}

func TestValidatePlayerLoggingRequiresMessagesAndRESTUsername(t *testing.T) {
	var conf Config
	conf.Web.Password = "devpass"
	conf.Web.Port = 8080
	conf.Task.SyncInterval = 60
	conf.Task.PlayerLogging = true
	conf.Rest.Address = "http://127.0.0.1:8212"
	conf.Rest.Password = "secret"
	conf.Rest.Timeout = 5
	conf.Save.SyncInterval = 0
	conf.Save.BackupInterval = 0
	err := Validate(&conf)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	validationErr := err.(*ValidationError)
	fields := map[string]bool{}
	for _, issue := range validationErr.Issues {
		fields[issue.Field] = true
	}
	for _, field := range []string{"task.player_login_message", "task.player_logout_message", "rest.username"} {
		if !fields[field] {
			t.Fatalf("expected issue for %s, got %#v", field, validationErr.Issues)
		}
	}
}
