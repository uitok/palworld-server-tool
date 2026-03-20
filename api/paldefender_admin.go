package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/internal/logger"
	"github.com/zaigie/palworld-server-tool/internal/tool"
	"github.com/zaigie/palworld-server-tool/service"
)

type palDefenderBatchGrantRequest struct {
	Targets     []playerActionTarget      `json:"targets"`
	PresetNames []string                  `json:"preset_names"`
	Grant       tool.PalDefenderGrantPlan `json:"grant"`
}

type palDefenderBatchRetryRequest struct {
	BatchID    string `json:"batch_id"`
	FailedOnly *bool  `json:"failed_only"`
}

type palDefenderResolvedTarget struct {
	playerActionTarget
	Nickname   string    `json:"nickname,omitempty"`
	LastOnline time.Time `json:"last_online,omitempty"`
}

type palDefenderBatchGrantResult struct {
	PlayerUID string `json:"player_uid,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	SteamID   string `json:"steam_id,omitempty"`
	Nickname  string `json:"nickname,omitempty"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
	Errors    int    `json:"errors,omitempty"`
}

type palDefenderBatchGrantResponse struct {
	Success              bool                          `json:"success"`
	BatchID              string                        `json:"batch_id"`
	SourceBatchID        string                        `json:"source_batch_id,omitempty"`
	RequestedTargetCount int                           `json:"requested_target_count"`
	TargetCount          int                           `json:"target_count"`
	SuccessCount         int                           `json:"success_count"`
	FailureCount         int                           `json:"failure_count"`
	FailureCodes         map[string]int                `json:"failure_codes,omitempty"`
	AppliedPresetNames   []string                      `json:"applied_preset_names,omitempty"`
	CompletedAt          time.Time                     `json:"completed_at"`
	DurationMs           int64                         `json:"duration_ms"`
	Results              []palDefenderBatchGrantResult `json:"results"`
}

var (
	palDefenderGiveFunc           = tool.PalDefenderGive
	palDefenderDeleteItemFunc     = tool.PalDefenderDeleteItem
	palDefenderClearInventoryFunc = tool.PalDefenderClearInventory
	palDefenderExportPalsFunc     = tool.PalDefenderExportPals
	palDefenderDeletePalsFunc     = tool.PalDefenderDeletePals
)

func writePlayerActionError(c *gin.Context, err error) {
	status := http.StatusBadRequest
	if strings.EqualFold(err.Error(), "Player not found") {
		status = http.StatusNotFound
	}
	writeError(c, status, err.Error(), tool.PalDefenderErrorCode(err), nil, 0)
}

func writePalDefenderError(c *gin.Context, err error) {
	errorsCount := 0
	var details any
	if pdErr, ok := err.(*tool.PalDefenderAPIError); ok {
		errorsCount = pdErr.Response.Errors
		details = pdErr.Response.Error
	}
	writeBadRequestDetails(c, err.Error(), tool.PalDefenderErrorCode(err), details, errorsCount)
}

func resolveBatchGrantTargets(targets []playerActionTarget) ([]palDefenderResolvedTarget, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("at least one target is required")
	}
	resolved := make([]palDefenderResolvedTarget, 0, len(targets))
	seen := map[string]struct{}{}
	for _, target := range targets {
		playerUID := strings.TrimSpace(target.PlayerUID)
		if playerUID == "" {
			return nil, fmt.Errorf("target player_uid is required")
		}
		if _, ok := seen[playerUID]; ok {
			continue
		}
		seen[playerUID] = struct{}{}
		resolvedTarget := palDefenderResolvedTarget{playerActionTarget: playerActionTarget{
			PlayerUID: playerUID,
			UserID:    strings.TrimSpace(target.UserID),
			SteamID:   strings.TrimSpace(target.SteamID),
		}}
		player, err := service.GetPlayer(getDB(), playerUID)
		if err == nil {
			if resolvedTarget.UserID == "" {
				resolvedTarget.UserID = player.UserId
			}
			if resolvedTarget.SteamID == "" {
				resolvedTarget.SteamID = player.SteamId
			}
			resolvedTarget.Nickname = player.Nickname
			resolvedTarget.LastOnline = player.LastOnline
		} else if err != service.ErrNoRecord {
			return nil, err
		}
		resolved = append(resolved, resolvedTarget)
	}
	return resolved, nil
}

func recordPalDefenderAuditLogWithDetails(c *gin.Context, action string, batchID string, target playerActionTarget, nickname string, presetNames []string, grant any, details any, result any, response tool.PalDefenderAPIResponse, err error) {
	playerUID := strings.TrimSpace(target.PlayerUID)
	userID := strings.TrimSpace(target.UserID)
	steamID := normalizePlayerActionSteamID(target.SteamID)
	if playerUID != "" {
		if player, lookupErr := service.GetPlayer(getDB(), playerUID); lookupErr == nil {
			if strings.TrimSpace(nickname) == "" {
				nickname = player.Nickname
			}
			if userID == "" {
				userID = player.UserId
			}
			if steamID == "" {
				steamID = normalizePlayerActionSteamID(player.SteamId)
			}
		}
	}
	log := database.PalDefenderAuditLog{
		Action:      action,
		Source:      c.FullPath(),
		Operator:    c.ClientIP(),
		BatchID:     batchID,
		PlayerUID:   playerUID,
		UserID:      userID,
		SteamID:     steamID,
		Nickname:    nickname,
		PresetNames: append([]string(nil), presetNames...),
		Grant:       grant,
		Details:     details,
		Result:      result,
		Success:     err == nil,
	}
	if err != nil {
		log.Error = err.Error()
		log.ErrorCode = tool.PalDefenderErrorCode(err)
	}
	if response.Errors > 0 {
		log.PalDefenderErrors = response.Errors
	}
	if saveErr := service.AddPalDefenderAuditLog(getDB(), log); saveErr != nil {
		logger.Errorf("save paldefender audit log failed: %v\n", saveErr)
	}
}

func recordPalDefenderAuditLog(c *gin.Context, action string, batchID string, target playerActionTarget, nickname string, presetNames []string, grant any, response tool.PalDefenderAPIResponse, err error) {
	recordPalDefenderAuditLogWithDetails(c, action, batchID, target, nickname, presetNames, grant, nil, nil, response, err)
}

func recordPalDefenderCommandAudit(c *gin.Context, action string, target playerActionTarget, details any, result any, err error) {
	response := tool.PalDefenderAPIResponse{}
	if pdErr, ok := err.(*tool.PalDefenderAPIError); ok {
		response = pdErr.Response
		if details == nil && pdErr.Response.Error != nil {
			details = pdErr.Response.Error
		}
	}
	recordPalDefenderAuditLogWithDetails(c, action, "", target, "", nil, nil, details, result, response, err)
}

func incrementFailureCode(summary map[string]int, code string) {
	code = strings.TrimSpace(code)
	if code == "" {
		code = "unknown_error"
	}
	summary[code]++
}

func getPalDefenderStatus(c *gin.Context) {
	c.JSON(http.StatusOK, tool.PalDefenderStatusSnapshot())
}

func parseAuditSuccessFilter(value string) *bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "success", "ok":
		result := true
		return &result
	case "false", "0", "fail", "failed", "error":
		result := false
		return &result
	default:
		return nil
	}
}

func buildPalDefenderAuditFilter(c *gin.Context, defaultLimit, maxLimit int) service.PalDefenderAuditLogFilter {
	limit, err := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("limit", strconv.Itoa(defaultLimit))))
	if err != nil || limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return service.PalDefenderAuditLogFilter{
		Limit:     limit,
		Action:    strings.TrimSpace(c.Query("action")),
		BatchID:   strings.TrimSpace(c.Query("batch_id")),
		PlayerUID: strings.TrimSpace(c.Query("player_uid")),
		UserID:    strings.TrimSpace(c.Query("user_id")),
		Success:   parseAuditSuccessFilter(c.Query("success")),
		ErrorCode: strings.TrimSpace(c.Query("error_code")),
	}
}

func listPalDefenderAuditLogs(c *gin.Context) {
	logs, err := service.ListPalDefenderAuditLogsByFilter(getDB(), buildPalDefenderAuditFilter(c, 20, 200))
	if err != nil {
		writeBadRequestErr(c, err)
		return
	}
	c.JSON(http.StatusOK, logs)
}

func exportPalDefenderAuditLogs(c *gin.Context) {
	logs, err := service.ListPalDefenderAuditLogsByFilter(getDB(), buildPalDefenderAuditFilter(c, 200, 1000))
	if err != nil {
		writeBadRequestErr(c, err)
		return
	}
	fileName := fmt.Sprintf("paldefender-audit-%s.json", time.Now().UTC().Format("20060102-150405"))
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	c.JSON(http.StatusOK, logs)
}

func executePalDefenderBatchGrant(c *gin.Context, resolvedTargets []palDefenderResolvedTarget, mergedGrant tool.PalDefenderGrantPlan, appliedPresetNames []string, action string, sourceBatchID string, requestedTargetCount int) palDefenderBatchGrantResponse {
	startedAt := time.Now().UTC()
	batchID := fmt.Sprintf("%020d", startedAt.UnixNano())
	results := make([]palDefenderBatchGrantResult, 0, len(resolvedTargets))
	successCount := 0
	failureCount := 0
	failureCodes := map[string]int{}
	for _, target := range resolvedTargets {
		result := palDefenderBatchGrantResult{
			PlayerUID: target.PlayerUID,
			UserID:    target.UserID,
			SteamID:   target.SteamID,
			Nickname:  target.Nickname,
		}
		if err := ensurePlayerOnlineForLiveAction(target.PlayerUID); err != nil {
			result.Error = err.Error()
			result.ErrorCode = tool.PalDefenderErrorCode(err)
			failureCount++
			incrementFailureCode(failureCodes, result.ErrorCode)
			recordPalDefenderAuditLog(c, action, batchID, target.playerActionTarget, target.Nickname, appliedPresetNames, mergedGrant, tool.PalDefenderAPIResponse{}, err)
			results = append(results, result)
			continue
		}
		userID, err := resolvePlayerActionUserID(target.PlayerUID, target.playerActionTarget)
		if err != nil {
			result.Error = err.Error()
			result.ErrorCode = tool.PalDefenderErrorCode(err)
			failureCount++
			incrementFailureCode(failureCodes, result.ErrorCode)
			recordPalDefenderAuditLog(c, action, batchID, target.playerActionTarget, target.Nickname, appliedPresetNames, mergedGrant, tool.PalDefenderAPIResponse{}, err)
			results = append(results, result)
			continue
		}
		result.UserID = userID
		request, err := mergedGrant.ToGiveRequest(userID)
		if err != nil {
			result.Error = err.Error()
			result.ErrorCode = tool.PalDefenderErrorCode(err)
			failureCount++
			incrementFailureCode(failureCodes, result.ErrorCode)
			recordPalDefenderAuditLog(c, action, batchID, playerActionTarget{PlayerUID: target.PlayerUID, UserID: userID, SteamID: target.SteamID}, target.Nickname, appliedPresetNames, mergedGrant, tool.PalDefenderAPIResponse{}, err)
			results = append(results, result)
			continue
		}
		response, err := palDefenderGiveFunc(request)
		if err != nil {
			result.Error = err.Error()
			result.ErrorCode = tool.PalDefenderErrorCode(err)
			if pdErr, ok := err.(*tool.PalDefenderAPIError); ok {
				result.Errors = pdErr.Response.Errors
			}
			failureCount++
			incrementFailureCode(failureCodes, result.ErrorCode)
			recordPalDefenderAuditLog(c, action, batchID, playerActionTarget{PlayerUID: target.PlayerUID, UserID: userID, SteamID: target.SteamID}, target.Nickname, appliedPresetNames, mergedGrant, response, err)
			results = append(results, result)
			continue
		}
		result.Success = true
		successCount++
		recordPalDefenderAuditLog(c, action, batchID, playerActionTarget{PlayerUID: target.PlayerUID, UserID: userID, SteamID: target.SteamID}, target.Nickname, appliedPresetNames, mergedGrant, response, nil)
		results = append(results, result)
	}
	completedAt := time.Now().UTC()
	return palDefenderBatchGrantResponse{
		Success:              failureCount == 0,
		BatchID:              batchID,
		SourceBatchID:        sourceBatchID,
		RequestedTargetCount: requestedTargetCount,
		TargetCount:          len(resolvedTargets),
		SuccessCount:         successCount,
		FailureCount:         failureCount,
		FailureCodes:         failureCodes,
		AppliedPresetNames:   appliedPresetNames,
		CompletedAt:          completedAt,
		DurationMs:           completedAt.Sub(startedAt).Milliseconds(),
		Results:              results,
	}
}

func decodePalDefenderGrantPlan(raw any) (tool.PalDefenderGrantPlan, error) {
	var plan tool.PalDefenderGrantPlan
	encoded, err := json.Marshal(raw)
	if err != nil {
		return plan, err
	}
	if err := json.Unmarshal(encoded, &plan); err != nil {
		return plan, err
	}
	return tool.NormalizePalDefenderGrantPlan(plan)
}

func retryPalDefenderBatch(c *gin.Context) {
	var req palDefenderBatchRetryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	req.BatchID = strings.TrimSpace(req.BatchID)
	if req.BatchID == "" {
		writeBadRequestCode(c, "batch_id is required", "paldefender_batch_id_required")
		return
	}
	failedOnly := true
	if req.FailedOnly != nil {
		failedOnly = *req.FailedOnly
	}
	logs, err := service.ListPalDefenderAuditLogsByFilter(getDB(), service.PalDefenderAuditLogFilter{Limit: 500, BatchID: req.BatchID})
	if err != nil {
		writeBadRequestErr(c, err)
		return
	}
	selectedLogs := make([]database.PalDefenderAuditLog, 0, len(logs))
	for _, log := range logs {
		if log.Action != "batch-grant" && log.Action != "batch-grant-retry" {
			continue
		}
		if failedOnly && log.Success {
			continue
		}
		selectedLogs = append(selectedLogs, log)
	}
	if len(selectedLogs) == 0 {
		writeBadRequestCode(c, "no retryable batch logs found", "paldefender_retry_targets_empty")
		return
	}
	mergedGrant, err := decodePalDefenderGrantPlan(selectedLogs[0].Grant)
	if err != nil {
		writeBadRequestCode(c, err.Error(), "paldefender_retry_grant_invalid")
		return
	}
	targets := make([]playerActionTarget, 0, len(selectedLogs))
	for _, log := range selectedLogs {
		targets = append(targets, playerActionTarget{
			PlayerUID: strings.TrimSpace(log.PlayerUID),
			UserID:    strings.TrimSpace(log.UserID),
			SteamID:   strings.TrimSpace(log.SteamID),
		})
	}
	resolvedTargets, err := resolveBatchGrantTargets(targets)
	if err != nil {
		writeBadRequestCode(c, err.Error(), tool.PalDefenderErrorCode(err))
		return
	}
	c.JSON(http.StatusOK, executePalDefenderBatchGrant(c, resolvedTargets, mergedGrant, selectedLogs[0].PresetNames, "batch-grant-retry", req.BatchID, len(targets)))
}

func grantPalDefenderBatch(c *gin.Context) {
	var req palDefenderBatchGrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	resolvedTargets, err := resolveBatchGrantTargets(req.Targets)
	if err != nil {
		writeBadRequestCode(c, err.Error(), tool.PalDefenderErrorCode(err))
		return
	}
	selectedPresets, presetGrant, err := tool.ResolvePalDefenderGrantPresets(req.PresetNames)
	if err != nil {
		writePalDefenderError(c, err)
		return
	}
	mergedGrant, err := tool.NormalizePalDefenderGrantPlan(req.Grant.Merge(presetGrant))
	if err != nil {
		writeBadRequestErr(c, err)
		return
	}
	appliedPresetNames := make([]string, 0, len(selectedPresets))
	for _, preset := range selectedPresets {
		appliedPresetNames = append(appliedPresetNames, preset.Name)
	}
	c.JSON(http.StatusOK, executePalDefenderBatchGrant(c, resolvedTargets, mergedGrant, appliedPresetNames, "batch-grant", "", len(req.Targets)))
}
