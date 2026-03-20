package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/internal/tool"
	"github.com/zaigie/palworld-server-tool/service"
)

type playerActionTarget struct {
	PlayerUID string `json:"player_uid"`
	UserID    string `json:"user_id"`
	SteamID   string `json:"steam_id"`
}

type grantPlayerItemsRequest struct {
	playerActionTarget
	ItemID string `json:"item_id"`
	Amount int    `json:"amount"`
}

type adjustPlayerItemsRequest struct {
	playerActionTarget
	ItemID      string `json:"item_id"`
	Operation   string `json:"operation"`
	Amount      int    `json:"amount"`
	TargetCount int    `json:"target_count"`
}

type clearPlayerInventoryRequest struct {
	playerActionTarget
	Containers []string `json:"containers"`
}

type grantPlayerPalRequest struct {
	playerActionTarget
	PalID  string `json:"pal_id"`
	Level  int    `json:"level"`
	Amount int    `json:"amount"`
}

type grantPlayerPalTemplateRequest struct {
	playerActionTarget
	TemplateName string `json:"template_name"`
	Amount       int    `json:"amount"`
}

type grantPlayerPalEggRequest struct {
	playerActionTarget
	EggID  string `json:"egg_id"`
	PalID  string `json:"pal_id"`
	Level  int    `json:"level"`
	Amount int    `json:"amount"`
}

type grantPlayerSupportRequest struct {
	playerActionTarget
	Kind   string `json:"kind"`
	Amount int    `json:"amount"`
}

type exportPlayerPalsRequest struct {
	playerActionTarget
}

type deletePlayerPalsFilters struct {
	PalID           string   `json:"pal_id"`
	Nickname        string   `json:"nickname"`
	Gender          string   `json:"gender"`
	IsLucky         *bool    `json:"is_lucky"`
	LevelCompare    string   `json:"level_compare"`
	Level           int      `json:"level"`
	RankCompare     string   `json:"rank_compare"`
	Rank            int      `json:"rank"`
	PassiveKeywords []string `json:"passive_keywords"`
}

type deletePlayerPalsRequest struct {
	playerActionTarget
	Filters deletePlayerPalsFilters `json:"filters"`
	Limit   int                     `json:"limit"`
}

const livePlayerActionOnlineWindow = 80 * time.Second

func ensurePlayerOnlineForLiveAction(playerUID string) error {
	player, err := service.GetPlayer(getDB(), playerUID)
	if err != nil {
		if err == service.ErrNoRecord {
			return fmt.Errorf("Player not found")
		}
		return err
	}
	if player.LastOnline.IsZero() || time.Since(player.LastOnline) > livePlayerActionOnlineWindow {
		return fmt.Errorf("player must be online for live PalDefender operations")
	}
	return nil
}

func normalizePlayerActionSteamID(steamID string) string {
	steamID = strings.TrimSpace(steamID)
	if strings.HasPrefix(strings.ToLower(steamID), "steam_") {
		return steamID[6:]
	}
	return steamID
}

func userIDFromSteamID(steamID string) string {
	normalized := normalizePlayerActionSteamID(steamID)
	if normalized == "" {
		return ""
	}
	return fmt.Sprintf("steam_%s", normalized)
}

func resolvePlayerActionUserID(playerUID string, target playerActionTarget) (string, error) {
	target.UserID = strings.TrimSpace(target.UserID)
	target.SteamID = normalizePlayerActionSteamID(target.SteamID)

	var player database.Player
	hasPlayer := false
	if playerUID != "" {
		loadedPlayer, err := service.GetPlayer(getDB(), playerUID)
		if err == nil {
			player = loadedPlayer
			hasPlayer = true
		} else if err != service.ErrNoRecord {
			return "", err
		}
	}

	if hasPlayer {
		if target.UserID != "" && strings.TrimSpace(player.UserId) != "" && target.UserID != strings.TrimSpace(player.UserId) {
			return "", fmt.Errorf("player action user id does not match player record")
		}
		playerSteamID := normalizePlayerActionSteamID(player.SteamId)
		if target.SteamID != "" && playerSteamID != "" && target.SteamID != playerSteamID {
			return "", fmt.Errorf("player action steam id does not match player record")
		}
	}

	switch {
	case target.UserID != "":
		return target.UserID, nil
	case hasPlayer && strings.TrimSpace(player.UserId) != "":
		return strings.TrimSpace(player.UserId), nil
	case target.SteamID != "":
		return userIDFromSteamID(target.SteamID), nil
	case hasPlayer:
		if userID := getPlayerActionUserId(player); userID != "" {
			return userID, nil
		}
	}
	return "", fmt.Errorf("player action user id not found")
}

func getPlayerInventoryItemCount(player database.Player, itemID string) int {
	if player.Items == nil {
		return 0
	}
	containers := [][]*database.Item{
		player.Items.CommonContainerId,
		player.Items.DropSlotContainerId,
		player.Items.EssentialContainerId,
		player.Items.FoodEquipContainerId,
		player.Items.PlayerEquipArmorContainerId,
		player.Items.WeaponLoadOutContainerId,
	}
	count := 0
	for _, items := range containers {
		for _, item := range items {
			if item != nil && item.ItemId == itemID {
				count += int(item.StackCount)
			}
		}
	}
	return count
}

func normalizeClearInventoryContainers(containers []string) ([]string, error) {
	mapped := make([]string, 0, len(containers))
	seen := map[string]struct{}{}
	lookup := map[string]string{
		"commoncontainerid":           "items",
		"items":                       "items",
		"essentialcontainerid":        "keyitems",
		"keyitems":                    "keyitems",
		"playerequiparmorcontainerid": "armor",
		"armor":                       "armor",
		"weaponloadoutcontainerid":    "weapons",
		"weapons":                     "weapons",
		"foodequipcontainerid":        "food",
		"food":                        "food",
		"dropslotcontainerid":         "dropslot",
		"dropslot":                    "dropslot",
		"all":                         "all",
	}
	for _, container := range containers {
		normalized := lookup[strings.ToLower(strings.TrimSpace(container))]
		if normalized == "" {
			return nil, fmt.Errorf("unsupported inventory container: %s", container)
		}
		if normalized == "all" {
			return []string{"all"}, nil
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		mapped = append(mapped, normalized)
	}
	if len(mapped) == 0 {
		mapped = append(mapped, "items")
	}
	return mapped, nil
}

func compareSymbol(operator string) string {
	switch strings.ToLower(strings.TrimSpace(operator)) {
	case "gt":
		return ">"
	case "gte":
		return ">="
	case "lt":
		return "<"
	case "lte":
		return "<="
	case "ne":
		return "!="
	default:
		return "="
	}
}

func buildDeletePalsFilter(filters deletePlayerPalsFilters, limit int) (string, error) {
	parts := make([]string, 0)
	if strings.TrimSpace(filters.PalID) != "" {
		parts = append(parts, "ID", strings.TrimSpace(filters.PalID))
	}
	if strings.TrimSpace(filters.Nickname) != "" {
		nickname := strings.TrimSpace(filters.Nickname)
		if strings.ContainsAny(nickname, " \t\r\n") {
			return "", fmt.Errorf("nickname filter must not contain whitespace")
		}
		parts = append(parts, "Nick", nickname)
	}
	if strings.TrimSpace(filters.Gender) != "" {
		parts = append(parts, "Gender", strings.ToLower(strings.TrimSpace(filters.Gender)))
	}
	if filters.Level > 0 {
		parts = append(parts, "Level"+compareSymbol(filters.LevelCompare)+strconv.Itoa(filters.Level))
	}
	if filters.Rank > 0 {
		parts = append(parts, "Rank"+compareSymbol(filters.RankCompare)+strconv.Itoa(filters.Rank))
	}
	if filters.IsLucky != nil {
		parts = append(parts, "Lucky", strconv.FormatBool(*filters.IsLucky))
	}
	if len(filters.PassiveKeywords) > 0 {
		keywords := make([]string, 0, len(filters.PassiveKeywords))
		for _, keyword := range filters.PassiveKeywords {
			keyword = strings.TrimSpace(keyword)
			if keyword == "" {
				continue
			}
			if strings.ContainsAny(keyword, " \t\r\n") {
				return "", fmt.Errorf("passive keyword must not contain whitespace: %s", keyword)
			}
			keywords = append(keywords, keyword)
		}
		if len(keywords) > 0 {
			parts = append(parts, "Passives", strings.Join(keywords, ","))
		}
	}
	if limit > 0 {
		parts = append(parts, "Limit", strconv.Itoa(limit))
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("at least one pal filter is required")
	}
	return strings.Join(parts, " "), nil
}

func recordPlayerGrantAudit(c *gin.Context, action string, target playerActionTarget, grant any, details any, response tool.PalDefenderAPIResponse, err error) {
	var result any
	if err == nil {
		result = response
	}
	recordPalDefenderAuditLogWithDetails(c, action, "", target, "", nil, grant, details, result, response, err)
}

func grantPlayerItems(c *gin.Context) {
	baseTarget := playerActionTarget{PlayerUID: c.Param("player_uid")}
	if err := ensurePlayerOnlineForLiveAction(c.Param("player_uid")); err != nil {
		recordPalDefenderCommandAudit(c, "grant-item", baseTarget, nil, nil, err)
		writePlayerActionError(c, err)
		return
	}
	var req grantPlayerItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	itemID := strings.TrimSpace(req.ItemID)
	if itemID == "" {
		writeBadRequestCode(c, "item_id is required", "item_required")
		return
	}
	if req.Amount <= 0 {
		writeBadRequestCode(c, "amount must be greater than 0", "invalid_amount")
		return
	}
	details := map[string]any{"item_id": itemID, "amount": req.Amount}
	userID, err := resolvePlayerActionUserID(c.Param("player_uid"), req.playerActionTarget)
	auditTarget := playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}
	if err != nil {
		recordPalDefenderCommandAudit(c, "grant-item", auditTarget, details, nil, err)
		writePlayerActionError(c, err)
		return
	}
	grantPlan := tool.PalDefenderGrantPlan{Items: []tool.PalDefenderPlanItem{{
		ItemID: itemID,
		Amount: req.Amount,
	}}}
	response, err := palDefenderGiveFunc(tool.PalDefenderGiveRequest{
		UserID: userID,
		Items: []tool.PalDefenderGiveItem{{
			ItemID: itemID,
			Count:  req.Amount,
		}},
	})
	if err != nil {
		recordPlayerGrantAudit(c, "grant-item", auditTarget, grantPlan, details, response, err)
		writePalDefenderError(c, err)
		return
	}
	recordPlayerGrantAudit(c, "grant-item", auditTarget, grantPlan, details, response, nil)
	c.JSON(http.StatusOK, gin.H{"success": true, "result": response})
}

func adjustPlayerItems(c *gin.Context) {
	baseTarget := playerActionTarget{PlayerUID: c.Param("player_uid")}
	if err := ensurePlayerOnlineForLiveAction(c.Param("player_uid")); err != nil {
		recordPalDefenderCommandAudit(c, "adjust-item", baseTarget, nil, nil, err)
		writePlayerActionError(c, err)
		return
	}
	var req adjustPlayerItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	itemID := strings.TrimSpace(req.ItemID)
	if itemID == "" {
		writeBadRequestCode(c, "item_id is required", "item_required")
		return
	}
	userID, err := resolvePlayerActionUserID(c.Param("player_uid"), req.playerActionTarget)
	auditTarget := playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}
	if err != nil {
		recordPalDefenderCommandAudit(c, "adjust-item", auditTarget, map[string]any{"item_id": itemID, "operation": strings.ToLower(strings.TrimSpace(req.Operation)), "amount": req.Amount, "target_count": req.TargetCount}, nil, err)
		writePlayerActionError(c, err)
		return
	}
	switch strings.ToLower(strings.TrimSpace(req.Operation)) {
	case "grant":
		if req.Amount <= 0 {
			writeBadRequestCode(c, "amount must be greater than 0", "invalid_amount")
			return
		}
		grantPlan := tool.PalDefenderGrantPlan{Items: []tool.PalDefenderPlanItem{{ItemID: itemID, Amount: req.Amount}}}
		details := map[string]any{"operation": "grant", "item_id": itemID, "amount": req.Amount}
		response, err := palDefenderGiveFunc(tool.PalDefenderGiveRequest{
			UserID: userID,
			Items:  []tool.PalDefenderGiveItem{{ItemID: itemID, Count: req.Amount}},
		})
		if err != nil {
			recordPalDefenderAuditLogWithDetails(c, "adjust-item-grant", "", auditTarget, "", nil, grantPlan, details, nil, response, err)
			writePalDefenderError(c, err)
			return
		}
		recordPalDefenderAuditLogWithDetails(c, "adjust-item-grant", "", auditTarget, "", nil, grantPlan, details, response, response, nil)
		c.JSON(http.StatusOK, gin.H{"success": true, "result": response})
	case "remove":
		if req.Amount <= 0 {
			writeBadRequestCode(c, "amount must be greater than 0", "invalid_amount")
			return
		}
		amount := strconv.Itoa(req.Amount)
		if player, err := service.GetPlayer(getDB(), c.Param("player_uid")); err == nil {
			currentCount := getPlayerInventoryItemCount(player, itemID)
			if currentCount > 0 && req.Amount >= currentCount {
				amount = "all"
			}
		}
		details := map[string]any{"operation": "remove", "item_id": itemID, "requested_amount": req.Amount, "effective_amount": amount}
		message, err := palDefenderDeleteItemFunc(userID, itemID, amount)
		if err != nil {
			recordPalDefenderCommandAudit(c, "adjust-item-remove", auditTarget, details, nil, err)
			writePalDefenderError(c, err)
			return
		}
		recordPalDefenderCommandAudit(c, "adjust-item-remove", auditTarget, details, message, nil)
		c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
	case "set":
		if req.TargetCount < 0 {
			writeBadRequestCode(c, "target_count must not be negative", "invalid_target_count")
			return
		}
		player, err := service.GetPlayer(getDB(), c.Param("player_uid"))
		if err != nil {
			if err == service.ErrNoRecord {
				writeNotFound(c, "Player not found")
				return
			}
			writeBadRequestErr(c, err)
			return
		}
		currentCount := getPlayerInventoryItemCount(player, itemID)
		delta := req.TargetCount - currentCount
		details := map[string]any{"operation": "set", "item_id": itemID, "target_count": req.TargetCount, "current_count": currentCount, "delta": delta}
		if delta == 0 {
			message := "No item changes required"
			recordPalDefenderCommandAudit(c, "adjust-item-set", auditTarget, details, message, nil)
			c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
			return
		}
		if delta > 0 {
			grantPlan := tool.PalDefenderGrantPlan{Items: []tool.PalDefenderPlanItem{{ItemID: itemID, Amount: delta}}}
			response, err := palDefenderGiveFunc(tool.PalDefenderGiveRequest{
				UserID: userID,
				Items:  []tool.PalDefenderGiveItem{{ItemID: itemID, Count: delta}},
			})
			if err != nil {
				recordPalDefenderAuditLogWithDetails(c, "adjust-item-set", "", auditTarget, "", nil, grantPlan, details, nil, response, err)
				writePalDefenderError(c, err)
				return
			}
			recordPalDefenderAuditLogWithDetails(c, "adjust-item-set", "", auditTarget, "", nil, grantPlan, details, response, response, nil)
			c.JSON(http.StatusOK, gin.H{"success": true, "result": response})
			return
		}
		amount := strconv.Itoa(-delta)
		if req.TargetCount == 0 {
			amount = "all"
		}
		details["effective_amount"] = amount
		message, err := palDefenderDeleteItemFunc(userID, itemID, amount)
		if err != nil {
			recordPalDefenderCommandAudit(c, "adjust-item-set", auditTarget, details, nil, err)
			writePalDefenderError(c, err)
			return
		}
		recordPalDefenderCommandAudit(c, "adjust-item-set", auditTarget, details, message, nil)
		c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
	default:
		writeBadRequestCode(c, "unsupported item operation", "unsupported_operation")
	}
}

func clearPlayerInventory(c *gin.Context) {
	baseTarget := playerActionTarget{PlayerUID: c.Param("player_uid")}
	if err := ensurePlayerOnlineForLiveAction(c.Param("player_uid")); err != nil {
		recordPalDefenderCommandAudit(c, "clear-inventory", baseTarget, nil, nil, err)
		writePlayerActionError(c, err)
		return
	}
	var req clearPlayerInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	userID, err := resolvePlayerActionUserID(c.Param("player_uid"), req.playerActionTarget)
	auditTarget := playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}
	if err != nil {
		recordPalDefenderCommandAudit(c, "clear-inventory", auditTarget, map[string]any{"containers": req.Containers}, nil, err)
		writePlayerActionError(c, err)
		return
	}
	containers, err := normalizeClearInventoryContainers(req.Containers)
	if err != nil {
		writeBadRequestErr(c, err)
		return
	}
	details := map[string]any{"containers": containers}
	message, err := palDefenderClearInventoryFunc(userID, containers)
	if err != nil {
		recordPalDefenderCommandAudit(c, "clear-inventory", auditTarget, details, nil, err)
		writePalDefenderError(c, err)
		return
	}
	recordPalDefenderCommandAudit(c, "clear-inventory", auditTarget, details, message, nil)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
}

func grantPlayerPal(c *gin.Context) {
	baseTarget := playerActionTarget{PlayerUID: c.Param("player_uid")}
	if err := ensurePlayerOnlineForLiveAction(c.Param("player_uid")); err != nil {
		recordPalDefenderCommandAudit(c, "grant-pal", baseTarget, nil, nil, err)
		writePlayerActionError(c, err)
		return
	}
	var req grantPlayerPalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	palID, err := tool.ValidatePalDefenderPalID(req.PalID)
	if err != nil {
		writeBadRequestErr(c, err)
		return
	}
	if req.Amount <= 0 {
		req.Amount = 1
	}
	if req.Level <= 0 {
		req.Level = 1
	}
	details := map[string]any{"pal_id": palID, "level": req.Level, "amount": req.Amount}
	userID, err := resolvePlayerActionUserID(c.Param("player_uid"), req.playerActionTarget)
	auditTarget := playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}
	if err != nil {
		recordPalDefenderCommandAudit(c, "grant-pal", auditTarget, details, nil, err)
		writePlayerActionError(c, err)
		return
	}
	pals := make([]tool.PalDefenderGivePal, 0, req.Amount)
	for i := 0; i < req.Amount; i++ {
		pals = append(pals, tool.PalDefenderGivePal{PalID: palID, Level: req.Level})
	}
	grantPlan := tool.PalDefenderGrantPlan{Pals: []tool.PalDefenderPlanPal{{PalID: palID, Level: req.Level, Amount: req.Amount}}}
	response, err := palDefenderGiveFunc(tool.PalDefenderGiveRequest{UserID: userID, Pals: pals})
	if err != nil {
		recordPlayerGrantAudit(c, "grant-pal", auditTarget, grantPlan, details, response, err)
		writePalDefenderError(c, err)
		return
	}
	recordPlayerGrantAudit(c, "grant-pal", auditTarget, grantPlan, details, response, nil)
	c.JSON(http.StatusOK, gin.H{"success": true, "result": response})
}

func grantPlayerPalEgg(c *gin.Context) {
	baseTarget := playerActionTarget{PlayerUID: c.Param("player_uid")}
	if err := ensurePlayerOnlineForLiveAction(c.Param("player_uid")); err != nil {
		recordPalDefenderCommandAudit(c, "grant-egg", baseTarget, nil, nil, err)
		writePlayerActionError(c, err)
		return
	}
	var req grantPlayerPalEggRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	eggID, err := tool.ValidatePalDefenderEggItemID(req.EggID)
	if err != nil {
		writeBadRequestErr(c, err)
		return
	}
	palID, err := tool.ValidatePalDefenderPalID(req.PalID)
	if err != nil {
		writeBadRequestErr(c, err)
		return
	}
	if req.Amount <= 0 {
		req.Amount = 1
	}
	if req.Level <= 0 {
		req.Level = 1
	}
	details := map[string]any{"egg_id": eggID, "pal_id": palID, "level": req.Level, "amount": req.Amount}
	userID, err := resolvePlayerActionUserID(c.Param("player_uid"), req.playerActionTarget)
	auditTarget := playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}
	if err != nil {
		recordPalDefenderCommandAudit(c, "grant-egg", auditTarget, details, nil, err)
		writePlayerActionError(c, err)
		return
	}
	eggs := make([]tool.PalDefenderGiveEgg, 0, req.Amount)
	for i := 0; i < req.Amount; i++ {
		eggs = append(eggs, tool.PalDefenderGiveEgg{ItemID: eggID, PalID: palID, Level: req.Level})
	}
	grantPlan := tool.PalDefenderGrantPlan{PalEggs: []tool.PalDefenderPlanEgg{{ItemID: eggID, PalID: palID, Level: req.Level, Amount: req.Amount}}}
	response, err := palDefenderGiveFunc(tool.PalDefenderGiveRequest{UserID: userID, PalEggs: eggs})
	if err != nil {
		recordPlayerGrantAudit(c, "grant-egg", auditTarget, grantPlan, details, response, err)
		writePalDefenderError(c, err)
		return
	}
	recordPlayerGrantAudit(c, "grant-egg", auditTarget, grantPlan, details, response, nil)
	c.JSON(http.StatusOK, gin.H{"success": true, "result": response})
}

func grantPlayerSupport(c *gin.Context) {
	baseTarget := playerActionTarget{PlayerUID: c.Param("player_uid")}
	if err := ensurePlayerOnlineForLiveAction(c.Param("player_uid")); err != nil {
		recordPalDefenderCommandAudit(c, "grant-support", baseTarget, nil, nil, err)
		writePlayerActionError(c, err)
		return
	}
	var req grantPlayerSupportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	if req.Amount <= 0 {
		writeBadRequestCode(c, "amount must be greater than 0", "invalid_amount")
		return
	}
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	details := map[string]any{"kind": kind, "amount": req.Amount}
	userID, err := resolvePlayerActionUserID(c.Param("player_uid"), req.playerActionTarget)
	auditTarget := playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}
	if err != nil {
		recordPalDefenderCommandAudit(c, "grant-support", auditTarget, details, nil, err)
		writePlayerActionError(c, err)
		return
	}
	pdReq := tool.PalDefenderGiveRequest{UserID: userID}
	grantPlan := tool.PalDefenderGrantPlan{}
	switch kind {
	case "exp":
		pdReq.EXP = req.Amount
		grantPlan.EXP = req.Amount
	case "relic", "lifmunk", "lifmunks":
		pdReq.Lifmunks = req.Amount
		grantPlan.Lifmunks = req.Amount
	case "tech", "tech_points", "technology_points":
		pdReq.TechnologyPoints = req.Amount
		grantPlan.TechnologyPoints = req.Amount
	case "ancient_tech", "ancient_tech_points", "ancient_technology_points":
		pdReq.AncientTechnologyPoints = req.Amount
		grantPlan.AncientTechnologyPoints = req.Amount
	default:
		writeBadRequestCode(c, "unsupported support grant kind", "unsupported_grant_kind")
		return
	}
	response, err := palDefenderGiveFunc(pdReq)
	if err != nil {
		recordPlayerGrantAudit(c, "grant-support", auditTarget, grantPlan, details, response, err)
		writePalDefenderError(c, err)
		return
	}
	recordPlayerGrantAudit(c, "grant-support", auditTarget, grantPlan, details, response, nil)
	c.JSON(http.StatusOK, gin.H{"success": true, "result": response})
}

func grantPlayerPalTemplate(c *gin.Context) {
	baseTarget := playerActionTarget{PlayerUID: c.Param("player_uid")}
	if err := ensurePlayerOnlineForLiveAction(c.Param("player_uid")); err != nil {
		recordPalDefenderCommandAudit(c, "grant-template", baseTarget, nil, nil, err)
		writePlayerActionError(c, err)
		return
	}
	var req grantPlayerPalTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	if req.Amount <= 0 {
		req.Amount = 1
	}
	details := map[string]any{"template_name": strings.TrimSpace(req.TemplateName), "amount": req.Amount}
	userID, err := resolvePlayerActionUserID(c.Param("player_uid"), req.playerActionTarget)
	auditTarget := playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}
	if err != nil {
		recordPalDefenderCommandAudit(c, "grant-template", auditTarget, details, nil, err)
		writePlayerActionError(c, err)
		return
	}
	templateName, err := tool.ValidatePalDefenderTemplateName(strings.TrimSpace(req.TemplateName))
	if err != nil {
		writeBadRequestErr(c, err)
		return
	}
	details["template_name"] = templateName
	templates := make([]string, 0, req.Amount)
	for i := 0; i < req.Amount; i++ {
		templates = append(templates, templateName)
	}
	grantPlan := tool.PalDefenderGrantPlan{PalTemplates: []tool.PalDefenderPlanTemplate{{TemplateName: templateName, Amount: req.Amount}}}
	response, err := palDefenderGiveFunc(tool.PalDefenderGiveRequest{UserID: userID, PalTemplates: templates})
	if err != nil {
		recordPlayerGrantAudit(c, "grant-template", auditTarget, grantPlan, details, response, err)
		writePalDefenderError(c, err)
		return
	}
	recordPlayerGrantAudit(c, "grant-template", auditTarget, grantPlan, details, response, nil)
	c.JSON(http.StatusOK, gin.H{"success": true, "result": response})
}

func exportPlayerPals(c *gin.Context) {
	baseTarget := playerActionTarget{PlayerUID: c.Param("player_uid")}
	if err := ensurePlayerOnlineForLiveAction(c.Param("player_uid")); err != nil {
		recordPalDefenderCommandAudit(c, "export-pals", baseTarget, nil, nil, err)
		writePlayerActionError(c, err)
		return
	}
	var req exportPlayerPalsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	userID, err := resolvePlayerActionUserID(c.Param("player_uid"), req.playerActionTarget)
	auditTarget := playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}
	if err != nil {
		recordPalDefenderCommandAudit(c, "export-pals", auditTarget, nil, nil, err)
		writePlayerActionError(c, err)
		return
	}
	message, err := palDefenderExportPalsFunc(userID)
	if err != nil {
		recordPalDefenderCommandAudit(c, "export-pals", auditTarget, nil, nil, err)
		writePalDefenderError(c, err)
		return
	}
	recordPalDefenderCommandAudit(c, "export-pals", auditTarget, nil, message, nil)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
}

func deletePlayerPals(c *gin.Context) {
	baseTarget := playerActionTarget{PlayerUID: c.Param("player_uid")}
	if err := ensurePlayerOnlineForLiveAction(c.Param("player_uid")); err != nil {
		recordPalDefenderCommandAudit(c, "delete-pals", baseTarget, nil, nil, err)
		writePlayerActionError(c, err)
		return
	}
	var req deletePlayerPalsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	userID, err := resolvePlayerActionUserID(c.Param("player_uid"), req.playerActionTarget)
	auditTarget := playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}
	if err != nil {
		recordPalDefenderCommandAudit(c, "delete-pals", auditTarget, map[string]any{"filters": req.Filters, "limit": req.Limit}, nil, err)
		writePlayerActionError(c, err)
		return
	}
	filter, err := buildDeletePalsFilter(req.Filters, req.Limit)
	if err != nil {
		writeBadRequestErr(c, err)
		return
	}
	details := map[string]any{"filters": req.Filters, "limit": req.Limit, "filter": filter}
	message, err := palDefenderDeletePalsFunc(userID, filter)
	if err != nil {
		recordPalDefenderCommandAudit(c, "delete-pals", auditTarget, details, nil, err)
		writePalDefenderError(c, err)
		return
	}
	recordPalDefenderCommandAudit(c, "delete-pals", auditTarget, details, message, nil)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
}
