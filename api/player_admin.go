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
	player, err := service.GetPlayer(database.GetDB(), playerUID)
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

func resolvePlayerActionUserID(playerUID string, target playerActionTarget) (string, error) {
	if playerUID != "" {
		player, err := service.GetPlayer(database.GetDB(), playerUID)
		if err == nil {
			if userID := getPlayerActionUserId(player); userID != "" {
				return userID, nil
			}
		} else if err != service.ErrNoRecord {
			return "", err
		}
	}
	if strings.TrimSpace(target.UserID) != "" {
		return strings.TrimSpace(target.UserID), nil
	}
	if strings.TrimSpace(target.SteamID) != "" {
		return fmt.Sprintf("steam_%s", strings.TrimSpace(target.SteamID)), nil
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

func grantPlayerItems(c *gin.Context) {
	if err := ensurePlayerOnlineForLiveAction(c.Param("player_uid")); err != nil {
		writePlayerActionError(c, err)
		return
	}
	var req grantPlayerItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.ItemID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item_id is required"})
		return
	}
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be greater than 0"})
		return
	}
	userID, err := resolvePlayerActionUserID(c.Param("player_uid"), req.playerActionTarget)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	grantPlan := tool.PalDefenderGrantPlan{Items: []tool.PalDefenderPlanItem{{
		ItemID: strings.TrimSpace(req.ItemID),
		Amount: req.Amount,
	}}}
	response, err := tool.PalDefenderGive(tool.PalDefenderGiveRequest{
		UserID: userID,
		Items: []tool.PalDefenderGiveItem{{
			ItemID: strings.TrimSpace(req.ItemID),
			Count:  req.Amount,
		}},
	})
	if err != nil {
		recordPalDefenderAuditLog(c, "grant-item", "", playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}, "", nil, grantPlan, response, err)
		writePalDefenderError(c, err)
		return
	}
	recordPalDefenderAuditLog(c, "grant-item", "", playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}, "", nil, grantPlan, response, nil)
	c.JSON(http.StatusOK, gin.H{"success": true, "result": response})
}

func adjustPlayerItems(c *gin.Context) {
	if err := ensurePlayerOnlineForLiveAction(c.Param("player_uid")); err != nil {
		writePlayerActionError(c, err)
		return
	}
	var req adjustPlayerItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	itemID := strings.TrimSpace(req.ItemID)
	if itemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item_id is required"})
		return
	}
	userID, err := resolvePlayerActionUserID(c.Param("player_uid"), req.playerActionTarget)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	switch strings.ToLower(strings.TrimSpace(req.Operation)) {
	case "grant":
		if req.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be greater than 0"})
			return
		}
		response, err := tool.PalDefenderGive(tool.PalDefenderGiveRequest{
			UserID: userID,
			Items:  []tool.PalDefenderGiveItem{{ItemID: itemID, Count: req.Amount}},
		})
		if err != nil {
			writePalDefenderError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "result": response})
	case "remove":
		if req.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be greater than 0"})
			return
		}
		amount := strconv.Itoa(req.Amount)
		if player, err := service.GetPlayer(database.GetDB(), c.Param("player_uid")); err == nil {
			currentCount := getPlayerInventoryItemCount(player, itemID)
			if currentCount > 0 && req.Amount >= currentCount {
				amount = "all"
			}
		}
		message, err := tool.PalDefenderDeleteItem(userID, itemID, amount)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
	case "set":
		if req.TargetCount < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "target_count must not be negative"})
			return
		}
		player, err := service.GetPlayer(database.GetDB(), c.Param("player_uid"))
		if err != nil {
			if err == service.ErrNoRecord {
				c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		currentCount := getPlayerInventoryItemCount(player, itemID)
		delta := req.TargetCount - currentCount
		if delta == 0 {
			c.JSON(http.StatusOK, gin.H{"success": true, "message": "No item changes required"})
			return
		}
		if delta > 0 {
			response, err := tool.PalDefenderGive(tool.PalDefenderGiveRequest{
				UserID: userID,
				Items:  []tool.PalDefenderGiveItem{{ItemID: itemID, Count: delta}},
			})
			if err != nil {
				writePalDefenderError(c, err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"success": true, "result": response})
			return
		}
		amount := strconv.Itoa(-delta)
		if req.TargetCount == 0 {
			amount = "all"
		}
		message, err := tool.PalDefenderDeleteItem(userID, itemID, amount)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported item operation"})
	}
}

func clearPlayerInventory(c *gin.Context) {
	if err := ensurePlayerOnlineForLiveAction(c.Param("player_uid")); err != nil {
		writePlayerActionError(c, err)
		return
	}
	var req clearPlayerInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, err := resolvePlayerActionUserID(c.Param("player_uid"), req.playerActionTarget)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	containers, err := normalizeClearInventoryContainers(req.Containers)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	message, err := tool.PalDefenderClearInventory(userID, containers)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
}

func grantPlayerPal(c *gin.Context) {
	if err := ensurePlayerOnlineForLiveAction(c.Param("player_uid")); err != nil {
		writePlayerActionError(c, err)
		return
	}
	var req grantPlayerPalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	palID, err := tool.ValidatePalDefenderPalID(req.PalID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Amount <= 0 {
		req.Amount = 1
	}
	if req.Level <= 0 {
		req.Level = 1
	}
	userID, err := resolvePlayerActionUserID(c.Param("player_uid"), req.playerActionTarget)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pals := make([]tool.PalDefenderGivePal, 0, req.Amount)
	for i := 0; i < req.Amount; i++ {
		pals = append(pals, tool.PalDefenderGivePal{PalID: palID, Level: req.Level})
	}
	grantPlan := tool.PalDefenderGrantPlan{Pals: []tool.PalDefenderPlanPal{{PalID: palID, Level: req.Level, Amount: req.Amount}}}
	response, err := tool.PalDefenderGive(tool.PalDefenderGiveRequest{UserID: userID, Pals: pals})
	if err != nil {
		recordPalDefenderAuditLog(c, "grant-pal", "", playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}, "", nil, grantPlan, response, err)
		writePalDefenderError(c, err)
		return
	}
	recordPalDefenderAuditLog(c, "grant-pal", "", playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}, "", nil, grantPlan, response, nil)
	c.JSON(http.StatusOK, gin.H{"success": true, "result": response})
}

func grantPlayerPalEgg(c *gin.Context) {
	if err := ensurePlayerOnlineForLiveAction(c.Param("player_uid")); err != nil {
		writePlayerActionError(c, err)
		return
	}
	var req grantPlayerPalEggRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	eggID, err := tool.ValidatePalDefenderEggItemID(req.EggID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	palID, err := tool.ValidatePalDefenderPalID(req.PalID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Amount <= 0 {
		req.Amount = 1
	}
	if req.Level <= 0 {
		req.Level = 1
	}
	userID, err := resolvePlayerActionUserID(c.Param("player_uid"), req.playerActionTarget)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	eggs := make([]tool.PalDefenderGiveEgg, 0, req.Amount)
	for i := 0; i < req.Amount; i++ {
		eggs = append(eggs, tool.PalDefenderGiveEgg{ItemID: eggID, PalID: palID, Level: req.Level})
	}
	grantPlan := tool.PalDefenderGrantPlan{PalEggs: []tool.PalDefenderPlanEgg{{ItemID: eggID, PalID: palID, Level: req.Level, Amount: req.Amount}}}
	response, err := tool.PalDefenderGive(tool.PalDefenderGiveRequest{UserID: userID, PalEggs: eggs})
	if err != nil {
		recordPalDefenderAuditLog(c, "grant-egg", "", playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}, "", nil, grantPlan, response, err)
		writePalDefenderError(c, err)
		return
	}
	recordPalDefenderAuditLog(c, "grant-egg", "", playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}, "", nil, grantPlan, response, nil)
	c.JSON(http.StatusOK, gin.H{"success": true, "result": response})
}

func grantPlayerSupport(c *gin.Context) {
	if err := ensurePlayerOnlineForLiveAction(c.Param("player_uid")); err != nil {
		writePlayerActionError(c, err)
		return
	}
	var req grantPlayerSupportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be greater than 0"})
		return
	}
	userID, err := resolvePlayerActionUserID(c.Param("player_uid"), req.playerActionTarget)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pdReq := tool.PalDefenderGiveRequest{UserID: userID}
	grantPlan := tool.PalDefenderGrantPlan{}
	switch strings.ToLower(strings.TrimSpace(req.Kind)) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported support grant kind"})
		return
	}
	response, err := tool.PalDefenderGive(pdReq)
	if err != nil {
		recordPalDefenderAuditLog(c, "grant-support", "", playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}, "", nil, grantPlan, response, err)
		writePalDefenderError(c, err)
		return
	}
	recordPalDefenderAuditLog(c, "grant-support", "", playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}, "", nil, grantPlan, response, nil)
	c.JSON(http.StatusOK, gin.H{"success": true, "result": response})
}

func grantPlayerPalTemplate(c *gin.Context) {
	if err := ensurePlayerOnlineForLiveAction(c.Param("player_uid")); err != nil {
		writePlayerActionError(c, err)
		return
	}
	var req grantPlayerPalTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Amount <= 0 {
		req.Amount = 1
	}
	userID, err := resolvePlayerActionUserID(c.Param("player_uid"), req.playerActionTarget)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	templateName, err := tool.ValidatePalDefenderTemplateName(strings.TrimSpace(req.TemplateName))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	templates := make([]string, 0, req.Amount)
	for i := 0; i < req.Amount; i++ {
		templates = append(templates, templateName)
	}
	grantPlan := tool.PalDefenderGrantPlan{PalTemplates: []tool.PalDefenderPlanTemplate{{TemplateName: templateName, Amount: req.Amount}}}
	response, err := tool.PalDefenderGive(tool.PalDefenderGiveRequest{UserID: userID, PalTemplates: templates})
	if err != nil {
		recordPalDefenderAuditLog(c, "grant-template", "", playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}, "", nil, grantPlan, response, err)
		writePalDefenderError(c, err)
		return
	}
	recordPalDefenderAuditLog(c, "grant-template", "", playerActionTarget{PlayerUID: c.Param("player_uid"), UserID: userID, SteamID: req.SteamID}, "", nil, grantPlan, response, nil)
	c.JSON(http.StatusOK, gin.H{"success": true, "result": response})
}

func exportPlayerPals(c *gin.Context) {
	if err := ensurePlayerOnlineForLiveAction(c.Param("player_uid")); err != nil {
		writePlayerActionError(c, err)
		return
	}
	var req exportPlayerPalsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, err := resolvePlayerActionUserID(c.Param("player_uid"), req.playerActionTarget)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	message, err := tool.PalDefenderExportPals(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
}

func deletePlayerPals(c *gin.Context) {
	if err := ensurePlayerOnlineForLiveAction(c.Param("player_uid")); err != nil {
		writePlayerActionError(c, err)
		return
	}
	var req deletePlayerPalsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, err := resolvePlayerActionUserID(c.Param("player_uid"), req.playerActionTarget)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filter, err := buildDeletePalsFilter(req.Filters, req.Limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	message, err := tool.PalDefenderDeletePals(userID, filter)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
}
