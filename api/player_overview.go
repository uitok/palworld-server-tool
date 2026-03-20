package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/service"
)

type PlayerOverviewListResponse struct {
	Success bool                             `json:"success"`
	Items   []database.PlayerOverviewSummary `json:"items"`
}

type PlayerOverviewDetailResponse struct {
	Success  bool                    `json:"success"`
	Overview database.PlayerOverview `json:"overview"`
}

type PlayerItemSearchResponse struct {
	Success  bool                           `json:"success"`
	Keyword  string                         `json:"keyword"`
	Items    []database.PlayerItemSearchHit `json:"items"`
}

type PlayerPalSearchResponse struct {
	Success bool                          `json:"success"`
	Keyword string                        `json:"keyword"`
	Items   []database.PlayerPalSearchHit `json:"items"`
}

func listPlayerOverviews(c *gin.Context) {
	items, err := service.ListPlayerOverviews(getDB(), service.PlayerOverviewFilter{
		Keyword:       c.Query("keyword"),
		OnlineOnly:    parseTruthyQuery(c.Query("online_only")),
		WhitelistOnly: parseTruthyQuery(c.Query("whitelist_only")),
		GuildOnly:     parseTruthyQuery(c.Query("guild_only")),
	})
	if err != nil {
		writeBadRequestErr(c, err)
		return
	}
	if !c.GetBool("loggedIn") {
		for i := range items {
			maskPlayerOverviewSummary(&items[i])
		}
	}
	c.JSON(http.StatusOK, PlayerOverviewListResponse{Success: true, Items: items})
}

func getPlayerOverviewDetail(c *gin.Context) {
	overview, err := service.GetPlayerOverview(getDB(), c.Param("player_uid"))
	if err != nil {
		if err == service.ErrNoRecord {
			writeNotFound(c, "player not found")
			return
		}
		writeBadRequestErr(c, err)
		return
	}
	if !c.GetBool("loggedIn") {
		maskPlayerOverviewSummary(&overview.Summary)
		maskPlayerDetail(&overview.Player)
	}
	c.JSON(http.StatusOK, PlayerOverviewDetailResponse{Success: true, Overview: overview})
}

func searchPlayerItems(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword == "" {
		writeBadRequestCode(c, "keyword is required", "player_search_keyword_required")
		return
	}
	items, err := service.SearchPlayerItems(getDB(), keyword, c.Query("player_uid"))
	if err != nil {
		if err == service.ErrNoRecord {
			writeNotFound(c, "player not found")
			return
		}
		writeBadRequestErr(c, err)
		return
	}
	c.JSON(http.StatusOK, PlayerItemSearchResponse{Success: true, Keyword: keyword, Items: items})
}

func searchPlayerPals(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword == "" {
		writeBadRequestCode(c, "keyword is required", "player_search_keyword_required")
		return
	}
	items, err := service.SearchPlayerPals(getDB(), keyword, c.Query("player_uid"))
	if err != nil {
		if err == service.ErrNoRecord {
			writeNotFound(c, "player not found")
			return
		}
		writeBadRequestErr(c, err)
		return
	}
	c.JSON(http.StatusOK, PlayerPalSearchResponse{Success: true, Keyword: keyword, Items: items})
}

func parseTruthyQuery(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func maskPlayerOverviewSummary(summary *database.PlayerOverviewSummary) {
	if summary == nil {
		return
	}
	if summary.UserId != "" {
		summary.UserId = strings.Split(summary.UserId, "_")[0] + "_"
	}
	summary.SteamId = ""
}

func maskPlayerDetail(player *database.Player) {
	if player == nil {
		return
	}
	player.Ip = ""
	if player.UserId != "" {
		player.UserId = strings.Split(player.UserId, "_")[0] + "_"
	}
	player.SteamId = ""
}
