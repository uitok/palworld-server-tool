package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/internal/tool"
	"github.com/zaigie/palworld-server-tool/service"
)

var (
	batchKickPlayerFunc   = tool.KickPlayer
	batchBanPlayerFunc    = tool.BanPlayer
	batchUnbanPlayerFunc  = tool.UnBanPlayer
	batchAddWhitelistFunc = service.AddWhitelist
	batchRemoveWhiteFunc  = service.RemoveWhitelist
)

type BatchPlayerActionRequest struct {
	Action     string   `json:"action"`
	PlayerUIDs []string `json:"player_uids"`
}

type BatchPlayerActionResult struct {
	PlayerUID string `json:"player_uid"`
	Nickname  string `json:"nickname,omitempty"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
}

type BatchPlayerActionResponse struct {
	Success   bool                      `json:"success"`
	Action    string                    `json:"action"`
	Requested int                       `json:"requested"`
	Succeeded int                       `json:"succeeded"`
	Failed    int                       `json:"failed"`
	Results   []BatchPlayerActionResult `json:"results"`
}

func batchPlayerAction(c *gin.Context) {
	var req BatchPlayerActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if !isSupportedBatchPlayerAction(action) {
		writeBadRequestCode(c, "invalid action", "invalid_player_batch_action")
		return
	}
	playerUIDs := uniquePlayerUIDs(req.PlayerUIDs)
	if len(playerUIDs) == 0 {
		writeBadRequestCode(c, "player_uids is required", "player_batch_targets_required")
		return
	}

	response := BatchPlayerActionResponse{
		Success:   true,
		Action:    action,
		Requested: len(playerUIDs),
		Results:   make([]BatchPlayerActionResult, 0, len(playerUIDs)),
	}

	for _, playerUID := range playerUIDs {
		player, err := service.GetPlayer(getDB(), playerUID)
		if err != nil {
			result := BatchPlayerActionResult{
				PlayerUID: playerUID,
				Success:   false,
				Error:     err.Error(),
				ErrorCode: "player_not_found",
			}
			if err != service.ErrNoRecord {
				result.ErrorCode = "player_action_failed"
			}
			response.Results = append(response.Results, result)
			response.Success = false
			response.Failed++
			continue
		}
		result := BatchPlayerActionResult{
			PlayerUID: player.PlayerUid,
			Nickname:  player.Nickname,
		}
		if err := executeBatchPlayerAction(action, player); err != nil {
			result.Success = false
			result.Error = err.Error()
			result.ErrorCode = batchPlayerActionErrorCode(action, err)
			response.Success = false
			response.Failed++
		} else {
			result.Success = true
			response.Succeeded++
		}
		response.Results = append(response.Results, result)
	}

	c.JSON(http.StatusOK, response)
}

func isSupportedBatchPlayerAction(action string) bool {
	switch action {
	case "whitelist_add", "whitelist_remove", "kick", "ban", "unban":
		return true
	default:
		return false
	}
}

func uniquePlayerUIDs(playerUIDs []string) []string {
	seen := map[string]struct{}{}
	items := make([]string, 0, len(playerUIDs))
	for _, playerUID := range playerUIDs {
		trimmed := strings.TrimSpace(playerUID)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		items = append(items, trimmed)
	}
	return items
}

func executeBatchPlayerAction(action string, player database.Player) error {
	switch action {
	case "whitelist_add":
		return batchAddWhitelistFunc(getDB(), whitelistEntryFromPlayer(player))
	case "whitelist_remove":
		return batchRemoveWhiteFunc(getDB(), whitelistEntryFromPlayer(player))
	case "kick":
		return executeBatchLivePlayerAction(player, batchKickPlayerFunc)
	case "ban":
		return executeBatchLivePlayerAction(player, batchBanPlayerFunc)
	case "unban":
		return executeBatchLivePlayerAction(player, batchUnbanPlayerFunc)
	default:
		return fmt.Errorf("unsupported player batch action: %s", action)
	}
}

func executeBatchLivePlayerAction(player database.Player, actionFunc func(string) error) error {
	userID := getPlayerActionUserId(player)
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("player action user id not found")
	}
	return actionFunc(userID)
}

func whitelistEntryFromPlayer(player database.Player) database.PlayerW {
	return database.PlayerW{
		Name:      player.Nickname,
		SteamID:   player.SteamId,
		PlayerUID: player.PlayerUid,
	}
}

func batchPlayerActionErrorCode(action string, err error) string {
	if err == nil {
		return ""
	}
	if err == service.ErrNoRecord {
		return "player_not_found"
	}
	if strings.Contains(strings.ToLower(err.Error()), "not found") {
		if strings.HasPrefix(action, "whitelist") {
			return "whitelist_player_not_found"
		}
		return "player_not_found"
	}
	if strings.HasPrefix(action, "whitelist") {
		return "whitelist_action_failed"
	}
	return "player_action_failed"
}
