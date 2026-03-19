package api

import (
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
	Success            bool                          `json:"success"`
	BatchID            string                        `json:"batch_id"`
	TargetCount        int                           `json:"target_count"`
	SuccessCount       int                           `json:"success_count"`
	FailureCount       int                           `json:"failure_count"`
	AppliedPresetNames []string                      `json:"applied_preset_names,omitempty"`
	Results            []palDefenderBatchGrantResult `json:"results"`
}

func writePlayerActionError(c *gin.Context, err error) {
	status := http.StatusBadRequest
	if strings.EqualFold(err.Error(), "Player not found") {
		status = http.StatusNotFound
	}
	c.JSON(status, gin.H{
		"error":      err.Error(),
		"error_code": tool.PalDefenderErrorCode(err),
	})
}

func writePalDefenderError(c *gin.Context, err error) {
	payload := gin.H{
		"error":      err.Error(),
		"error_code": tool.PalDefenderErrorCode(err),
	}
	if pdErr, ok := err.(*tool.PalDefenderAPIError); ok {
		payload["errors"] = pdErr.Response.Errors
		payload["detail"] = pdErr.Response.Error
	}
	c.JSON(http.StatusBadRequest, payload)
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
		player, err := service.GetPlayer(database.GetDB(), playerUID)
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

func recordPalDefenderAuditLog(c *gin.Context, action string, batchID string, target playerActionTarget, nickname string, presetNames []string, grant any, response tool.PalDefenderAPIResponse, err error) {
	playerUID := strings.TrimSpace(target.PlayerUID)
	userID := strings.TrimSpace(target.UserID)
	steamID := strings.TrimSpace(target.SteamID)
	if playerUID != "" {
		if player, lookupErr := service.GetPlayer(database.GetDB(), playerUID); lookupErr == nil {
			if strings.TrimSpace(nickname) == "" {
				nickname = player.Nickname
			}
			if userID == "" {
				userID = player.UserId
			}
			if steamID == "" {
				steamID = player.SteamId
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
		Success:     err == nil,
	}
	if err != nil {
		log.Error = err.Error()
		log.ErrorCode = tool.PalDefenderErrorCode(err)
	}
	if response.Errors > 0 {
		log.PalDefenderErrors = response.Errors
	}
	if saveErr := service.AddPalDefenderAuditLog(database.GetDB(), log); saveErr != nil {
		logger.Errorf("save paldefender audit log failed: %v\n", saveErr)
	}
}

func getPalDefenderStatus(c *gin.Context) {
	c.JSON(http.StatusOK, tool.PalDefenderStatusSnapshot())
}

func listPalDefenderAuditLogs(c *gin.Context) {
	limit, err := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("limit", "20")))
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	logs, err := service.ListPalDefenderAuditLogs(database.GetDB(), limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}

func grantPalDefenderBatch(c *gin.Context) {
	var req palDefenderBatchGrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resolvedTargets, err := resolveBatchGrantTargets(req.Targets)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      err.Error(),
			"error_code": tool.PalDefenderErrorCode(err),
		})
		return
	}
	selectedPresets, presetGrant, err := tool.ResolvePalDefenderGrantPresets(req.PresetNames)
	if err != nil {
		writePalDefenderError(c, err)
		return
	}
	mergedGrant, err := tool.NormalizePalDefenderGrantPlan(req.Grant.Merge(presetGrant))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	appliedPresetNames := make([]string, 0, len(selectedPresets))
	for _, preset := range selectedPresets {
		appliedPresetNames = append(appliedPresetNames, preset.Name)
	}
	batchID := fmt.Sprintf("%020d", time.Now().UTC().UnixNano())
	results := make([]palDefenderBatchGrantResult, 0, len(resolvedTargets))
	successCount := 0
	failureCount := 0
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
			recordPalDefenderAuditLog(c, "batch-grant", batchID, target.playerActionTarget, target.Nickname, appliedPresetNames, mergedGrant, tool.PalDefenderAPIResponse{}, err)
			results = append(results, result)
			continue
		}
		userID, err := resolvePlayerActionUserID(target.PlayerUID, target.playerActionTarget)
		if err != nil {
			result.Error = err.Error()
			result.ErrorCode = tool.PalDefenderErrorCode(err)
			failureCount++
			recordPalDefenderAuditLog(c, "batch-grant", batchID, target.playerActionTarget, target.Nickname, appliedPresetNames, mergedGrant, tool.PalDefenderAPIResponse{}, err)
			results = append(results, result)
			continue
		}
		result.UserID = userID
		request, err := mergedGrant.ToGiveRequest(userID)
		if err != nil {
			result.Error = err.Error()
			result.ErrorCode = tool.PalDefenderErrorCode(err)
			failureCount++
			recordPalDefenderAuditLog(c, "batch-grant", batchID, palDefenderResolvedTarget{playerActionTarget: playerActionTarget{PlayerUID: target.PlayerUID, UserID: userID, SteamID: target.SteamID}}.playerActionTarget, target.Nickname, appliedPresetNames, mergedGrant, tool.PalDefenderAPIResponse{}, err)
			results = append(results, result)
			continue
		}
		response, err := tool.PalDefenderGive(request)
		if err != nil {
			result.Error = err.Error()
			result.ErrorCode = tool.PalDefenderErrorCode(err)
			if pdErr, ok := err.(*tool.PalDefenderAPIError); ok {
				result.Errors = pdErr.Response.Errors
			}
			failureCount++
			recordPalDefenderAuditLog(c, "batch-grant", batchID, playerActionTarget{PlayerUID: target.PlayerUID, UserID: userID, SteamID: target.SteamID}, target.Nickname, appliedPresetNames, mergedGrant, response, err)
			results = append(results, result)
			continue
		}
		result.Success = true
		successCount++
		recordPalDefenderAuditLog(c, "batch-grant", batchID, playerActionTarget{PlayerUID: target.PlayerUID, UserID: userID, SteamID: target.SteamID}, target.Nickname, appliedPresetNames, mergedGrant, response, nil)
		results = append(results, result)
	}
	c.JSON(http.StatusOK, palDefenderBatchGrantResponse{
		Success:            failureCount == 0,
		BatchID:            batchID,
		TargetCount:        len(resolvedTargets),
		SuccessCount:       successCount,
		FailureCount:       failureCount,
		AppliedPresetNames: appliedPresetNames,
		Results:            results,
	})
}
