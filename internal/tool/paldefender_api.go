package tool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

func getPalDefenderConfig() (string, string, time.Duration, error) {
	if !viper.GetBool("paldefender.enabled") {
		return "", "", 0, fmt.Errorf("paldefender is disabled")
	}
	address := normalizeBaseAddress(viper.GetString("paldefender.address"))
	if address == "" {
		return "", "", 0, fmt.Errorf("paldefender.address is required")
	}
	authKey := strings.TrimSpace(viper.GetString("paldefender.auth_key"))
	if authKey == "" {
		return "", "", 0, fmt.Errorf("paldefender.auth_key is required")
	}
	timeout := time.Duration(viper.GetInt("paldefender.timeout")) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
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
