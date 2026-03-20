package tool

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var palDefenderIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

type PalDefenderAPIResponse struct {
	Errors int `json:"Errors"`
	Error  any `json:"Error"`
}

type PalDefenderAPIError struct {
	StatusCode int
	Response   PalDefenderAPIResponse
	Body       string
}

func (e *PalDefenderAPIError) Error() string {
	detail := strings.TrimSpace(formatPalDefenderErrorDetail(e.Response.Error))
	if detail != "" {
		if e.Response.Errors > 0 {
			return fmt.Sprintf("paldefender: %d error(s): %s", e.Response.Errors, detail)
		}
		return "paldefender: " + detail
	}
	if strings.TrimSpace(e.Body) != "" {
		return fmt.Sprintf("paldefender: %s", strings.TrimSpace(e.Body))
	}
	return fmt.Sprintf("paldefender: request failed with status %d", e.StatusCode)
}

type PalDefenderGiveItem struct {
	ItemID string `json:"ItemID"`
	Count  int    `json:"Count"`
}

type PalDefenderGivePal struct {
	PalID string `json:"PalID"`
	Level int    `json:"Level,omitempty"`
}

type PalDefenderGiveEgg struct {
	ItemID string `json:"ItemID"`
	PalID  string `json:"PalID"`
	Level  int    `json:"Level,omitempty"`
}

type PalDefenderGiveRequest struct {
	UserID                  string                `json:"UserID"`
	EXP                     int                   `json:"EXP,omitempty"`
	Lifmunks                int                   `json:"Lifmunks,omitempty"`
	TechnologyPoints        int                   `json:"TechnologyPoints,omitempty"`
	AncientTechnologyPoints int                   `json:"AncientTechnologyPoints,omitempty"`
	UnlockTechnology        string                `json:"UnlockTechnology,omitempty"`
	Items                   []PalDefenderGiveItem `json:"Items,omitempty"`
	Pals                    []PalDefenderGivePal  `json:"Pals,omitempty"`
	PalTemplates            []string              `json:"PalTemplates,omitempty"`
	PalEggs                 []PalDefenderGiveEgg  `json:"PalEggs,omitempty"`
}

type PalDefenderPlanItem struct {
	ItemID string `json:"item_id"`
	Amount int    `json:"amount"`
}

type PalDefenderPlanPal struct {
	PalID  string `json:"pal_id"`
	Level  int    `json:"level"`
	Amount int    `json:"amount"`
}

type PalDefenderPlanEgg struct {
	ItemID string `json:"item_id"`
	PalID  string `json:"pal_id"`
	Level  int    `json:"level"`
	Amount int    `json:"amount"`
}

type PalDefenderPlanTemplate struct {
	TemplateName string `json:"template_name"`
	Amount       int    `json:"amount"`
}

type PalDefenderGrantPlan struct {
	EXP                     int                       `json:"exp"`
	Lifmunks                int                       `json:"lifmunks"`
	TechnologyPoints        int                       `json:"technology_points"`
	AncientTechnologyPoints int                       `json:"ancient_technology_points"`
	Items                   []PalDefenderPlanItem     `json:"items,omitempty"`
	Pals                    []PalDefenderPlanPal      `json:"pals,omitempty"`
	PalEggs                 []PalDefenderPlanEgg      `json:"pal_eggs,omitempty"`
	PalTemplates            []PalDefenderPlanTemplate `json:"pal_templates,omitempty"`
}

type PalDefenderGrantPreset struct {
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Grant       PalDefenderGrantPlan `json:"grant"`
}

type PalDefenderStatus struct {
	Enabled      bool                     `json:"enabled"`
	Configured   bool                     `json:"configured"`
	Reachable    bool                     `json:"reachable"`
	Healthy      bool                     `json:"healthy"`
	Address      string                   `json:"address,omitempty"`
	Timeout      int                      `json:"timeout"`
	CheckedAt    time.Time                `json:"checked_at"`
	Version      map[string]any           `json:"version,omitempty"`
	Error        string                   `json:"error,omitempty"`
	ErrorCode    string                   `json:"error_code,omitempty"`
	Capabilities []string                 `json:"capabilities,omitempty"`
	Presets      []PalDefenderGrantPreset `json:"presets,omitempty"`
}

func formatPalDefenderErrorDetail(detail any) string {
	if detail == nil {
		return ""
	}
	b, err := json.Marshal(detail)
	if err != nil {
		return fmt.Sprintf("%v", detail)
	}
	return string(b)
}

func normalizeBaseAddress(address string) string {
	address = strings.TrimSpace(address)
	if address == "" {
		return ""
	}
	if !strings.Contains(address, "://") {
		return "http://" + address
	}
	return address
}

func inspectPalDefenderConfig() (bool, string, string, time.Duration, error) {
	enabled := viper.GetBool("paldefender.enabled")
	address := normalizeBaseAddress(viper.GetString("paldefender.address"))
	authKey := strings.TrimSpace(viper.GetString("paldefender.auth_key"))
	timeout := time.Duration(viper.GetInt("paldefender.timeout")) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if !enabled {
		return false, address, authKey, timeout, fmt.Errorf("paldefender is disabled")
	}
	if address == "" {
		return true, address, authKey, timeout, fmt.Errorf("paldefender.address is required")
	}
	if authKey == "" {
		return true, address, authKey, timeout, fmt.Errorf("paldefender.auth_key is required")
	}
	return true, address, authKey, timeout, nil
}

func getPalDefenderConfig() (string, string, time.Duration, error) {
	_, address, authKey, timeout, err := inspectPalDefenderConfig()
	if err != nil {
		return "", "", 0, err
	}
	return address, authKey, timeout, nil
}

func callPalDefenderAPI(method, api string, payload any, result any) error {
	address, authKey, timeout, err := getPalDefenderConfig()
	if err != nil {
		return err
	}

	joinedURL, err := url.JoinPath(address, api)
	if err != nil {
		return err
	}

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, joinedURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+authKey)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		pdResp := PalDefenderAPIResponse{}
		_ = json.Unmarshal(respBody, &pdResp)
		return &PalDefenderAPIError{
			StatusCode: resp.StatusCode,
			Response:   pdResp,
			Body:       string(respBody),
		}
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return err
		}
	}
	return nil
}

func PalDefenderGive(request PalDefenderGiveRequest) (PalDefenderAPIResponse, error) {
	if strings.TrimSpace(request.UserID) == "" {
		return PalDefenderAPIResponse{}, fmt.Errorf("paldefender user id is required")
	}
	response := PalDefenderAPIResponse{}
	if err := callPalDefenderAPI(http.MethodPost, "/v1/pdapi/give", request, &response); err != nil {
		return response, err
	}
	if response.Errors > 0 {
		return response, &PalDefenderAPIError{StatusCode: http.StatusOK, Response: response}
	}
	return response, nil
}

func PalDefenderVersion() (map[string]any, error) {
	var response map[string]any
	if err := callPalDefenderAPI(http.MethodGet, "/v1/pdapi/version", nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func PalDefenderErrorCode(err error) string {
	if err == nil {
		return ""
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(message, "paldefender is disabled"):
		return "paldefender_disabled"
	case strings.Contains(message, "paldefender.address is required"), strings.Contains(message, "paldefender.auth_key is required"):
		return "paldefender_unconfigured"
	case strings.Contains(message, "player not found"):
		return "player_not_found"
	case strings.Contains(message, "player must be online"):
		return "player_offline"
	case strings.Contains(message, "player action user id not found"):
		return "player_action_user_id_not_found"
	case strings.Contains(message, "player action user id does not match player record"), strings.Contains(message, "player action steam id does not match player record"):
		return "player_action_target_mismatch"
	}
	var pdErr *PalDefenderAPIError
	if errors.As(err, &pdErr) {
		switch pdErr.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return "paldefender_auth_failed"
		case http.StatusNotFound:
			return "paldefender_endpoint_not_found"
		default:
			if pdErr.StatusCode >= http.StatusInternalServerError {
				return "paldefender_service_error"
			}
			return "paldefender_request_failed"
		}
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return "paldefender_unreachable"
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return "paldefender_unreachable"
	}
	if strings.Contains(message, "connection refused") || strings.Contains(message, "no such host") || strings.Contains(message, "timeout") {
		return "paldefender_unreachable"
	}
	return "paldefender_error"
}

func normalizePositive(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func (plan PalDefenderGrantPlan) HasAnyGrant() bool {
	return plan.EXP > 0 ||
		plan.Lifmunks > 0 ||
		plan.TechnologyPoints > 0 ||
		plan.AncientTechnologyPoints > 0 ||
		len(plan.Items) > 0 ||
		len(plan.Pals) > 0 ||
		len(plan.PalEggs) > 0 ||
		len(plan.PalTemplates) > 0
}

func NormalizePalDefenderGrantPlan(plan PalDefenderGrantPlan) (PalDefenderGrantPlan, error) {
	normalized := PalDefenderGrantPlan{
		EXP:                     plan.EXP,
		Lifmunks:                plan.Lifmunks,
		TechnologyPoints:        plan.TechnologyPoints,
		AncientTechnologyPoints: plan.AncientTechnologyPoints,
	}
	if normalized.EXP < 0 || normalized.Lifmunks < 0 || normalized.TechnologyPoints < 0 || normalized.AncientTechnologyPoints < 0 {
		return normalized, fmt.Errorf("support grant amount must not be negative")
	}

	for _, item := range plan.Items {
		itemID := strings.TrimSpace(item.ItemID)
		if itemID == "" && item.Amount == 0 {
			continue
		}
		if err := validatePalDefenderIdentifier("item id", itemID); err != nil {
			return normalized, err
		}
		if item.Amount <= 0 {
			return normalized, fmt.Errorf("item amount must be greater than 0")
		}
		normalized.Items = append(normalized.Items, PalDefenderPlanItem{ItemID: itemID, Amount: item.Amount})
	}

	for _, pal := range plan.Pals {
		palID := strings.TrimSpace(pal.PalID)
		if palID == "" && pal.Amount == 0 && pal.Level == 0 {
			continue
		}
		validatedPalID, err := ValidatePalDefenderPalID(palID)
		if err != nil {
			return normalized, err
		}
		if pal.Amount <= 0 {
			return normalized, fmt.Errorf("pal amount must be greater than 0")
		}
		normalized.Pals = append(normalized.Pals, PalDefenderPlanPal{PalID: validatedPalID, Level: normalizePositive(pal.Level, 1), Amount: pal.Amount})
	}

	for _, egg := range plan.PalEggs {
		eggID := strings.TrimSpace(egg.ItemID)
		palID := strings.TrimSpace(egg.PalID)
		if eggID == "" && palID == "" && egg.Amount == 0 && egg.Level == 0 {
			continue
		}
		validatedEggID, err := ValidatePalDefenderEggItemID(eggID)
		if err != nil {
			return normalized, err
		}
		validatedPalID, err := ValidatePalDefenderPalID(palID)
		if err != nil {
			return normalized, err
		}
		if egg.Amount <= 0 {
			return normalized, fmt.Errorf("egg amount must be greater than 0")
		}
		normalized.PalEggs = append(normalized.PalEggs, PalDefenderPlanEgg{ItemID: validatedEggID, PalID: validatedPalID, Level: normalizePositive(egg.Level, 1), Amount: egg.Amount})
	}

	for _, template := range plan.PalTemplates {
		templateName := strings.TrimSpace(template.TemplateName)
		if templateName == "" && template.Amount == 0 {
			continue
		}
		validatedTemplateName, err := ValidatePalDefenderTemplateName(templateName)
		if err != nil {
			return normalized, err
		}
		if template.Amount <= 0 {
			return normalized, fmt.Errorf("template amount must be greater than 0")
		}
		normalized.PalTemplates = append(normalized.PalTemplates, PalDefenderPlanTemplate{TemplateName: validatedTemplateName, Amount: template.Amount})
	}

	if !normalized.HasAnyGrant() {
		return normalized, fmt.Errorf("at least one PalDefender grant operation is required")
	}
	return normalized, nil
}

func (plan PalDefenderGrantPlan) Merge(other PalDefenderGrantPlan) PalDefenderGrantPlan {
	merged := plan
	merged.EXP += other.EXP
	merged.Lifmunks += other.Lifmunks
	merged.TechnologyPoints += other.TechnologyPoints
	merged.AncientTechnologyPoints += other.AncientTechnologyPoints
	merged.Items = append(merged.Items, other.Items...)
	merged.Pals = append(merged.Pals, other.Pals...)
	merged.PalEggs = append(merged.PalEggs, other.PalEggs...)
	merged.PalTemplates = append(merged.PalTemplates, other.PalTemplates...)
	return merged
}

func (plan PalDefenderGrantPlan) ToGiveRequest(userID string) (PalDefenderGiveRequest, error) {
	validatedUserID, err := validatePalDefenderUserID(userID)
	if err != nil {
		return PalDefenderGiveRequest{}, err
	}
	normalized, err := NormalizePalDefenderGrantPlan(plan)
	if err != nil {
		return PalDefenderGiveRequest{}, err
	}
	request := PalDefenderGiveRequest{
		UserID:                  validatedUserID,
		EXP:                     normalized.EXP,
		Lifmunks:                normalized.Lifmunks,
		TechnologyPoints:        normalized.TechnologyPoints,
		AncientTechnologyPoints: normalized.AncientTechnologyPoints,
	}
	for _, item := range normalized.Items {
		request.Items = append(request.Items, PalDefenderGiveItem{ItemID: item.ItemID, Count: item.Amount})
	}
	for _, pal := range normalized.Pals {
		for i := 0; i < pal.Amount; i++ {
			request.Pals = append(request.Pals, PalDefenderGivePal{PalID: pal.PalID, Level: pal.Level})
		}
	}
	for _, egg := range normalized.PalEggs {
		for i := 0; i < egg.Amount; i++ {
			request.PalEggs = append(request.PalEggs, PalDefenderGiveEgg{ItemID: egg.ItemID, PalID: egg.PalID, Level: egg.Level})
		}
	}
	for _, template := range normalized.PalTemplates {
		for i := 0; i < template.Amount; i++ {
			request.PalTemplates = append(request.PalTemplates, template.TemplateName)
		}
	}
	return request, nil
}

func LoadPalDefenderGrantPresets() ([]PalDefenderGrantPreset, error) {
	raw := viper.Get("paldefender.presets")
	if raw == nil {
		return []PalDefenderGrantPreset{}, nil
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid paldefender.presets: %w", err)
	}
	var presets []PalDefenderGrantPreset
	if err := json.Unmarshal(encoded, &presets); err != nil {
		return nil, fmt.Errorf("invalid paldefender.presets: %w", err)
	}
	seen := map[string]struct{}{}
	for index, preset := range presets {
		preset.Name = strings.TrimSpace(preset.Name)
		preset.Description = strings.TrimSpace(preset.Description)
		presets[index].Name = preset.Name
		presets[index].Description = preset.Description
		if err := validatePalDefenderIdentifier("preset name", preset.Name); err != nil {
			return nil, err
		}
		lookupKey := strings.ToLower(preset.Name)
		if _, ok := seen[lookupKey]; ok {
			return nil, fmt.Errorf("duplicate preset name: %s", preset.Name)
		}
		seen[lookupKey] = struct{}{}
		normalizedGrant, err := NormalizePalDefenderGrantPlan(preset.Grant)
		if err != nil {
			return nil, fmt.Errorf("invalid preset %s: %w", preset.Name, err)
		}
		presets[index].Grant = normalizedGrant
	}
	return presets, nil
}

func ResolvePalDefenderGrantPresets(names []string) ([]PalDefenderGrantPreset, PalDefenderGrantPlan, error) {
	presets, err := LoadPalDefenderGrantPresets()
	if err != nil {
		return nil, PalDefenderGrantPlan{}, err
	}
	if len(names) == 0 {
		return []PalDefenderGrantPreset{}, PalDefenderGrantPlan{}, nil
	}
	lookup := make(map[string]PalDefenderGrantPreset, len(presets))
	for _, preset := range presets {
		lookup[strings.ToLower(preset.Name)] = preset
	}
	selected := make([]PalDefenderGrantPreset, 0, len(names))
	merged := PalDefenderGrantPlan{}
	seen := map[string]struct{}{}
	for _, name := range names {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			continue
		}
		key := strings.ToLower(trimmedName)
		if _, ok := seen[key]; ok {
			continue
		}
		preset, ok := lookup[key]
		if !ok {
			return nil, PalDefenderGrantPlan{}, fmt.Errorf("preset not found: %s", trimmedName)
		}
		seen[key] = struct{}{}
		selected = append(selected, preset)
		merged = merged.Merge(preset.Grant)
	}
	return selected, merged, nil
}

func PalDefenderStatusSnapshot() PalDefenderStatus {
	enabled, address, _, timeout, cfgErr := inspectPalDefenderConfig()
	status := PalDefenderStatus{
		Enabled:      enabled,
		Configured:   enabled && cfgErr == nil,
		Address:      address,
		Timeout:      int(timeout / time.Second),
		CheckedAt:    time.Now().UTC(),
		Capabilities: []string{"version", "live-grant", "batch-grant", "audit-log", "presets"},
	}
	presets, presetErr := LoadPalDefenderGrantPresets()
	if presetErr == nil {
		status.Presets = presets
	}
	if cfgErr != nil {
		status.Error = cfgErr.Error()
		status.ErrorCode = PalDefenderErrorCode(cfgErr)
		return status
	}
	if presetErr != nil {
		status.Error = presetErr.Error()
		status.ErrorCode = "paldefender_preset_invalid"
		return status
	}
	version, err := PalDefenderVersion()
	if err != nil {
		status.Error = err.Error()
		status.ErrorCode = PalDefenderErrorCode(err)
		status.Reachable = false
		return status
	}
	status.Reachable = true
	status.Healthy = true
	status.Version = version
	return status
}

func validatePalDefenderIdentifier(label, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s is required", label)
	}
	if !palDefenderIdentifierPattern.MatchString(value) {
		return fmt.Errorf("invalid %s: %s", label, value)
	}
	return nil
}

func ValidatePalDefenderTemplateName(templateName string) (string, error) {
	templateName = strings.TrimSpace(templateName)
	if templateName == "" {
		return "", fmt.Errorf("template name is required")
	}
	if strings.ContainsAny(templateName, `/\\`) {
		return "", fmt.Errorf("template name must not contain path separators")
	}
	if filepath.Base(templateName) != templateName {
		return "", fmt.Errorf("template name must be a file name")
	}
	if err := validatePalDefenderIdentifier("template name", templateName); err != nil {
		return "", err
	}
	return templateName, nil
}

func ValidatePalDefenderPalID(palID string) (string, error) {
	palID = strings.TrimSpace(palID)
	if err := validatePalDefenderIdentifier("pal id", palID); err != nil {
		return "", err
	}
	return palID, nil
}

func ValidatePalDefenderEggItemID(itemID string) (string, error) {
	itemID = strings.TrimSpace(itemID)
	if err := validatePalDefenderIdentifier("egg item id", itemID); err != nil {
		return "", err
	}
	if !strings.HasPrefix(strings.ToLower(itemID), "palegg_") {
		return "", fmt.Errorf("egg item id must start with palegg_: %s", itemID)
	}
	return itemID, nil
}

func validatePalDefenderUserID(userID string) (string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", fmt.Errorf("user id is required")
	}
	if strings.ContainsAny(userID, " \t\r\n") {
		return "", fmt.Errorf("user id must not contain whitespace")
	}
	return userID, nil
}

func PalDefenderDeleteItem(userID, itemID, amount string) (string, error) {
	validatedUserID, err := validatePalDefenderUserID(userID)
	if err != nil {
		return "", err
	}
	if err := validatePalDefenderIdentifier("item id", itemID); err != nil {
		return "", err
	}
	amount = strings.TrimSpace(amount)
	if amount == "" {
		amount = "1"
	}
	if amount != "all" {
		if _, err := strconv.Atoi(amount); err != nil {
			return "", fmt.Errorf("invalid amount: %s", amount)
		}
	}
	return CustomCommand(fmt.Sprintf("delitem %s %s %s", validatedUserID, strings.TrimSpace(itemID), amount))
}

func PalDefenderClearInventory(userID string, containers []string) (string, error) {
	validatedUserID, err := validatePalDefenderUserID(userID)
	if err != nil {
		return "", err
	}
	if len(containers) == 0 {
		containers = []string{"items"}
	}
	for _, container := range containers {
		if err := validatePalDefenderIdentifier("container", container); err != nil {
			return "", err
		}
	}
	return CustomCommand(fmt.Sprintf("clearinv %s %s", validatedUserID, strings.Join(containers, " ")))
}

func PalDefenderExportPals(userID string) (string, error) {
	validatedUserID, err := validatePalDefenderUserID(userID)
	if err != nil {
		return "", err
	}
	return CustomCommand(fmt.Sprintf("exportpals %s", validatedUserID))
}

func PalDefenderDeletePals(userID, filter string) (string, error) {
	validatedUserID, err := validatePalDefenderUserID(userID)
	if err != nil {
		return "", err
	}
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return "", fmt.Errorf("pal filter is required")
	}
	if strings.ContainsAny(filter, "\r\n") {
		return "", fmt.Errorf("pal filter must not contain newlines")
	}
	return CustomCommand(fmt.Sprintf("deletepals %s %s", validatedUserID, filter))
}
