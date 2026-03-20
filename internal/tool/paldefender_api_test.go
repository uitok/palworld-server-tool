package tool

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func setPalDefenderPresetsForTest(t *testing.T, presets any) {
	t.Helper()

	previous := viper.Get("paldefender.presets")
	viper.Set("paldefender.presets", presets)
	t.Cleanup(func() {
		viper.Set("paldefender.presets", previous)
	})
}

func TestNormalizePalDefenderGrantPlan(t *testing.T) {
	t.Run("normalizes valid plan", func(t *testing.T) {
		normalized, err := NormalizePalDefenderGrantPlan(PalDefenderGrantPlan{
			EXP:              100,
			TechnologyPoints: 2,
			Items: []PalDefenderPlanItem{
				{ItemID: "  Stone  ", Amount: 5},
				{},
			},
			Pals: []PalDefenderPlanPal{
				{PalID: "  SheepBall  ", Level: 0, Amount: 2},
			},
			PalEggs: []PalDefenderPlanEgg{
				{ItemID: "  palegg_common  ", PalID: "  SheepBall  ", Level: 0, Amount: 1},
			},
			PalTemplates: []PalDefenderPlanTemplate{
				{TemplateName: "  starter-pack  ", Amount: 1},
			},
		})
		if err != nil {
			t.Fatalf("expected valid plan, got %v", err)
		}
		if normalized.Items[0].ItemID != "Stone" {
			t.Fatalf("expected trimmed item id, got %#v", normalized.Items)
		}
		if normalized.Pals[0].PalID != "SheepBall" || normalized.Pals[0].Level != 1 {
			t.Fatalf("expected pal to be normalized, got %#v", normalized.Pals)
		}
		if normalized.PalEggs[0].ItemID != "palegg_common" || normalized.PalEggs[0].Level != 1 {
			t.Fatalf("expected egg to be normalized, got %#v", normalized.PalEggs)
		}
		if normalized.PalTemplates[0].TemplateName != "starter-pack" {
			t.Fatalf("expected template to be normalized, got %#v", normalized.PalTemplates)
		}
	})

	testCases := []struct {
		name    string
		plan    PalDefenderGrantPlan
		wantErr string
	}{
		{
			name:    "requires at least one grant",
			plan:    PalDefenderGrantPlan{},
			wantErr: "at least one PalDefender grant operation is required",
		},
		{
			name:    "rejects invalid item amount",
			plan:    PalDefenderGrantPlan{Items: []PalDefenderPlanItem{{ItemID: "Stone", Amount: 0}}},
			wantErr: "item amount must be greater than 0",
		},
		{
			name:    "rejects invalid pal id",
			plan:    PalDefenderGrantPlan{Pals: []PalDefenderPlanPal{{PalID: "bad pal", Amount: 1, Level: 1}}},
			wantErr: "invalid pal id",
		},
		{
			name:    "rejects invalid egg item",
			plan:    PalDefenderGrantPlan{PalEggs: []PalDefenderPlanEgg{{ItemID: "egg_common", PalID: "SheepBall", Amount: 1, Level: 1}}},
			wantErr: "egg item id must start with palegg_",
		},
		{
			name:    "rejects invalid template amount",
			plan:    PalDefenderGrantPlan{PalTemplates: []PalDefenderPlanTemplate{{TemplateName: "starter-pack", Amount: 0}}},
			wantErr: "template amount must be greater than 0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NormalizePalDefenderGrantPlan(tc.plan)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestLoadPalDefenderGrantPresets(t *testing.T) {
	t.Run("loads and normalizes presets", func(t *testing.T) {
		setPalDefenderPresetsForTest(t, []map[string]any{
			{
				"name":        " starter ",
				"description": " basic support ",
				"grant": map[string]any{
					"exp":   10,
					"items": []map[string]any{{"item_id": " Stone ", "amount": 3}},
				},
			},
		})

		presets, err := LoadPalDefenderGrantPresets()
		if err != nil {
			t.Fatalf("expected presets to load, got %v", err)
		}
		if len(presets) != 1 {
			t.Fatalf("expected 1 preset, got %d", len(presets))
		}
		if presets[0].Name != "starter" || presets[0].Description != "basic support" {
			t.Fatalf("expected preset metadata to be trimmed, got %#v", presets[0])
		}
		if presets[0].Grant.Items[0].ItemID != "Stone" {
			t.Fatalf("expected grant to be normalized, got %#v", presets[0].Grant)
		}
	})

	t.Run("rejects duplicate names case insensitively", func(t *testing.T) {
		setPalDefenderPresetsForTest(t, []map[string]any{
			{"name": "starter", "grant": map[string]any{"exp": 10}},
			{"name": "Starter", "grant": map[string]any{"exp": 20}},
		})

		_, err := LoadPalDefenderGrantPresets()
		if err == nil || !strings.Contains(err.Error(), "duplicate preset name") {
			t.Fatalf("expected duplicate preset error, got %v", err)
		}
	})

	t.Run("rejects invalid preset name", func(t *testing.T) {
		setPalDefenderPresetsForTest(t, []map[string]any{
			{"name": "bad name", "grant": map[string]any{"exp": 10}},
		})

		_, err := LoadPalDefenderGrantPresets()
		if err == nil || !strings.Contains(err.Error(), "invalid preset name") {
			t.Fatalf("expected invalid preset name error, got %v", err)
		}
	})

	t.Run("rejects invalid grant", func(t *testing.T) {
		setPalDefenderPresetsForTest(t, []map[string]any{
			{
				"name": "starter",
				"grant": map[string]any{
					"items": []map[string]any{{"item_id": "Stone", "amount": 0}},
				},
			},
		})

		_, err := LoadPalDefenderGrantPresets()
		if err == nil || !strings.Contains(err.Error(), "invalid preset starter") {
			t.Fatalf("expected invalid preset grant error, got %v", err)
		}
	})
}

func TestResolvePalDefenderGrantPresets(t *testing.T) {
	setPalDefenderPresetsForTest(t, []map[string]any{
		{
			"name": "starter",
			"grant": map[string]any{
				"exp":   10,
				"items": []map[string]any{{"item_id": "Stone", "amount": 2}},
			},
		},
		{
			"name": "builder",
			"grant": map[string]any{
				"lifmunks":      5,
				"pal_templates": []map[string]any{{"template_name": "builder-pack", "amount": 1}},
			},
		},
	})

	selected, merged, err := ResolvePalDefenderGrantPresets([]string{" STARTER ", "starter", "builder"})
	if err != nil {
		t.Fatalf("expected presets to resolve, got %v", err)
	}
	if len(selected) != 2 {
		t.Fatalf("expected deduplicated selected presets, got %d", len(selected))
	}
	if selected[0].Name != "starter" || selected[1].Name != "builder" {
		t.Fatalf("unexpected preset order: %#v", selected)
	}
	if merged.EXP != 10 || merged.Lifmunks != 5 {
		t.Fatalf("unexpected merged support values: %#v", merged)
	}
	if len(merged.Items) != 1 || merged.Items[0].ItemID != "Stone" {
		t.Fatalf("unexpected merged items: %#v", merged.Items)
	}
	if len(merged.PalTemplates) != 1 || merged.PalTemplates[0].TemplateName != "builder-pack" {
		t.Fatalf("unexpected merged templates: %#v", merged.PalTemplates)
	}

	_, _, err = ResolvePalDefenderGrantPresets([]string{"missing"})
	if err == nil || !strings.Contains(err.Error(), "preset not found") {
		t.Fatalf("expected preset not found error, got %v", err)
	}
}

type stubNetError struct{}

func (stubNetError) Error() string   { return "network failed" }
func (stubNetError) Timeout() bool   { return true }
func (stubNetError) Temporary() bool { return false }

func TestPalDefenderErrorCode(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", err: nil, want: ""},
		{name: "disabled", err: errors.New("paldefender is disabled"), want: "paldefender_disabled"},
		{name: "unconfigured address", err: errors.New("paldefender.address is required"), want: "paldefender_unconfigured"},
		{name: "player not found", err: errors.New("player not found"), want: "player_not_found"},
		{name: "player offline", err: errors.New("player must be online"), want: "player_offline"},
		{name: "user id missing", err: errors.New("player action user id not found"), want: "player_action_user_id_not_found"},
		{name: "target mismatch", err: errors.New("player action user id does not match player record"), want: "player_action_target_mismatch"},
		{name: "api unauthorized", err: &PalDefenderAPIError{StatusCode: http.StatusUnauthorized}, want: "paldefender_auth_failed"},
		{name: "api forbidden", err: &PalDefenderAPIError{StatusCode: http.StatusForbidden}, want: "paldefender_auth_failed"},
		{name: "api not found", err: &PalDefenderAPIError{StatusCode: http.StatusNotFound}, want: "paldefender_endpoint_not_found"},
		{name: "api server error", err: &PalDefenderAPIError{StatusCode: http.StatusBadGateway}, want: "paldefender_service_error"},
		{name: "api request failed", err: &PalDefenderAPIError{StatusCode: http.StatusBadRequest}, want: "paldefender_request_failed"},
		{name: "url error", err: &url.Error{Op: "Get", URL: "http://127.0.0.1", Err: errors.New("connection refused")}, want: "paldefender_unreachable"},
		{name: "net error", err: stubNetError{}, want: "paldefender_unreachable"},
		{name: "plain timeout text", err: errors.New("timeout"), want: "paldefender_unreachable"},
		{name: "fallback", err: errors.New("something else"), want: "paldefender_error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := PalDefenderErrorCode(tc.err); got != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, got)
			}
		})
	}
}
