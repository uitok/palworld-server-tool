package api

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/internal/tool"
	"github.com/zaigie/palworld-server-tool/service"
)

type PlayerOrderBy string

const (
	OrderByLastOnline PlayerOrderBy = "last_online"
	OrderByLevel      PlayerOrderBy = "level"
)

// getPlayerActionUserId 获取用于 kick/ban/unban 操作的 userId
// 优先使用完整的 UserId（支持跨平台），兜底使用 steam_ + SteamId
func getPlayerActionUserId(player database.Player) string {
	if player.UserId != "" {
		return player.UserId
	}
	if player.SteamId != "" {
		return fmt.Sprintf("steam_%s", player.SteamId)
	}
	return ""
}

// listOnlinePlayers godoc
//
//	@Summary		List Online Players
//	@Description	List Online Players
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//
//	@Success		200	{object}	[]database.OnlinePlayer
//	@Failure		400	{object}	ErrorResponse
//	@Router			/api/online_player [get]
func listOnlinePlayers(c *gin.Context) {
	onlinePLayers, err := tool.ShowPlayers()
	if err != nil {
		writeBadRequestErr(c, err)
		return
	}
	service.PutPlayersOnline(getDB(), onlinePLayers)
	// 未登录隐藏敏感字段
	if !c.GetBool("loggedIn") {
		for i := range onlinePLayers {
			onlinePLayers[i].Ip = ""
			if onlinePLayers[i].UserId != "" {
				onlinePLayers[i].UserId = strings.Split(onlinePLayers[i].UserId, "_")[0] + "_"
			}
			onlinePLayers[i].SteamId = ""
		}
	}
	c.JSON(http.StatusOK, onlinePLayers)
}

// putPlayers godoc
//
//	@Summary		Put Players
//	@Description	Put Players Only For SavSync,PlayerSync
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//
//	@Security		ApiKeyAuth
//
//	@Param			players	body		[]database.Player	true	"Players"
//
//	@Success		200		{object}	SuccessResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Router			/api/player [put]
func putPlayers(c *gin.Context) {
	var players []database.Player
	if err := c.ShouldBindJSON(&players); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	if err := service.PutPlayers(getDB(), players); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	writeSuccess(c)
}

// listPlayers godoc
//
//	@Summary		List Players
//	@Description	List Players
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//
//	@Param			order_by	query		PlayerOrderBy	false	"order by field"	enum(last_online,level)
//	@Param			desc		query		bool			false	"order by desc"
//
//	@Success		200			{object}	[]database.TersePlayer
//	@Failure		400			{object}	ErrorResponse
//	@Router			/api/player [get]
func listPlayers(c *gin.Context) {
	orderBy := c.Query("order_by")
	desc := c.Query("desc")
	players, err := service.ListPlayers(getDB())
	if err != nil {
		writeBadRequestErr(c, err)
		return
	}
	//未登录隐藏字段
	if !c.GetBool("loggedIn") {
		for i := range players {
			players[i].Ip = ""
			if players[i].UserId != "" {
				players[i].UserId = strings.Split(players[i].UserId, "_")[0] + "_"
			}
			players[i].SteamId = ""
		}
	}
	//排序
	if orderBy == "level" {
		sort.Slice(players, func(i, j int) bool {
			if desc == "true" {
				return players[i].Level > players[j].Level
			}
			return players[i].Level < players[j].Level
		})
	}
	if orderBy == "last_online" {
		sort.Slice(players, func(i, j int) bool {
			if desc == "true" {
				return players[i].LastOnline.Sub(players[j].LastOnline) > 0
			}
			return players[i].LastOnline.Sub(players[j].LastOnline) < 0
		})
	}
	c.JSON(http.StatusOK, players)
}

// getPlayer godoc
//
//	@Summary		Get Player
//	@Description	Get Player
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//
//	@Param			player_uid	path		string	true	"Player UID"
//
//	@Success		200			{object}	database.Player
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	EmptyResponse
//	@Router			/api/player/{player_uid} [get]
func getPlayer(c *gin.Context) {
	player, err := service.GetPlayer(getDB(), c.Param("player_uid"))
	if err != nil {
		if err == service.ErrNoRecord {
			writeNotFound(c, "player not found")
			return
		}
		writeBadRequestErr(c, err)
		return
	}
	//未登录隐藏字段
	if !c.GetBool("loggedIn") {
		player.Ip = ""
		if player.UserId != "" {
			player.UserId = strings.Split(player.UserId, "_")[0] + "_"
		}
		player.SteamId = ""
	}
	c.JSON(http.StatusOK, player)
}

// kickPlayer godoc
//
//	@Summary		Kick Player
//	@Description	Kick Player
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			player_uid	path		string	true	"Player UID"
//
//	@Success		200			{object}	SuccessResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Router			/api/player/{player_uid}/kick [post]
func kickPlayer(c *gin.Context) {
	playerUid := c.Param("player_uid")
	player, err := service.GetPlayer(getDB(), playerUid)
	if err != nil {
		if err == service.ErrNoRecord {
			writeNotFound(c, "Player not found")
			return
		}
		writeBadRequestErr(c, err)
		return
	}
	err = tool.KickPlayer(getPlayerActionUserId(player))
	if err != nil {
		writeBadRequestErr(c, err)
		return
	}
	writeSuccess(c)
}

// banPlayer godoc
//
//	@Summary		Ban Player
//	@Description	Ban Player
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			player_uid	path		string	true	"Player UID"
//
//	@Success		200			{object}	SuccessResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Router			/api/player/{player_uid}/ban [post]
func banPlayer(c *gin.Context) {
	playerUid := c.Param("player_uid")
	player, err := service.GetPlayer(getDB(), playerUid)
	if err != nil {
		if err == service.ErrNoRecord {
			writeNotFound(c, "Player not found")
			return
		}
		writeBadRequestErr(c, err)
		return
	}
	err = tool.BanPlayer(getPlayerActionUserId(player))
	if err != nil {
		writeBadRequestErr(c, err)
		return
	}
	writeSuccess(c)
}

// unbanPlayer godoc
//
//	@Summary		Unban Player
//	@Description	Unban Player
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			player_uid	path		string	true	"Player UID"
//
//	@Success		200			{object}	SuccessResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Router			/api/player/{player_uid}/unban [post]
func unbanPlayer(c *gin.Context) {
	playerUid := c.Param("player_uid")
	player, err := service.GetPlayer(getDB(), playerUid)
	if err != nil {
		if err == service.ErrNoRecord {
			writeNotFound(c, "Player not found")
			return
		}
		writeBadRequestErr(c, err)
		return
	}
	err = tool.UnBanPlayer(getPlayerActionUserId(player))
	if err != nil {
		writeBadRequestErr(c, err)
		return
	}
	writeSuccess(c)
}

// addWhite godoc
//
//	@Summary		Add White List
//	@Description	Add White List
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			player_uid	path		string	true	"Player UID"
//
//	@Success		200			{object}	SuccessResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Router			/api/whitelist [post]
func addWhite(c *gin.Context) {
	var player database.PlayerW
	if err := c.ShouldBindJSON(&player); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	if err := service.AddWhitelist(getDB(), player); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	writeSuccess(c)
}

// listWhite godoc
//
//	@Summary		List White List
//	@Description	List White List
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	[]database.PlayerW
//	@Failure		400	{object}	ErrorResponse
//	@Router			/api/whitelist [get]
func listWhite(c *gin.Context) {
	players, err := service.ListWhitelist(getDB())
	if err != nil {
		writeBadRequestErr(c, err)
		return
	}
	c.JSON(http.StatusOK, players)
}

// removeWhite godoc
//
//	@Summary		Remove White List
//	@Description	Remove White List
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			player_uid	path		string	true	"Player UID"
//
//	@Success		200			{object}	SuccessResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Router			/api/whitelist [delete]
func removeWhite(c *gin.Context) {
	var player database.PlayerW
	if err := c.ShouldBindJSON(&player); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	if err := service.RemoveWhitelist(getDB(), player); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	writeSuccess(c)
}

// putWhite godoc
//
//	@Summary		Put White List
//	@Description	Put White List
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			players	body		[]database.PlayerW	true	"Players"
//
//	@Success		200		{object}	SuccessResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Router			/api/whitelist [put]
func putWhite(c *gin.Context) {
	var players []database.PlayerW
	if err := c.ShouldBindJSON(&players); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	if err := service.PutWhitelist(getDB(), players); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	writeSuccess(c)
}
