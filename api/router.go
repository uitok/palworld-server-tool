package api

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/zaigie/palworld-server-tool/internal/auth"
	"github.com/zaigie/palworld-server-tool/internal/httpx"
	"go.etcd.io/bbolt"
)

type SuccessResponse struct {
	Success bool `json:"success"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type ErrorResponse = httpx.ErrorResponse

type EmptyResponse struct{}

func ignoreLogPrefix(path string) bool {
	prefixes := []string{"/swagger/", "/assets/", "/favicon.ico", "/map"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		if !ignoreLogPrefix(param.Path) {
			statusColor := param.StatusCodeColor()
			methodColor := param.MethodColor()
			resetColor := param.ResetColor()
			return fmt.Sprintf("[GIN] %v |%s %3d %s| %13v | %15s |%s %-7s %s %#v\n%s",
				param.TimeStamp.Format("2006/01/02 - 15:04:05"),
				statusColor, param.StatusCode, resetColor,
				param.Latency,
				param.ClientIP,
				methodColor, param.Method, resetColor,
				param.Path,
				param.ErrorMessage,
			)
		}
		return ""
	})
}

func RegisterRouter(r *gin.Engine, db *bbolt.DB) {
	setDB(db)
	r.Use(Logger(), gin.Recovery())

	r.POST("/api/login", loginHandler)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	apiGroup := r.Group("/api")

	anonymousGroup := apiGroup.Group("")
	{
		anonymousGroup.GET("/server", getServer)
		anonymousGroup.GET("/server/tool", getServerTool)
		anonymousGroup.GET("/server/metrics", getServerMetrics)
		anonymousGroup.GET("/server/overview", getServerOverview)
		anonymousGroup.GET("/server/task-status", getServerTaskStatus)
		anonymousGroup.GET("/guild", listGuilds)
		anonymousGroup.GET("/guild/:admin_player_uid", getGuild)
	}
	// 根据登录状态返回不同结果
	OptionalGroup := apiGroup.Group("")
	OptionalGroup.Use(auth.OptionalJWTMiddleware())
	{
		OptionalGroup.GET("/online_player", listOnlinePlayers)
		OptionalGroup.GET("/player/overview", listPlayerOverviews)
		OptionalGroup.GET("/player/:player_uid/overview", getPlayerOverviewDetail)
		OptionalGroup.GET("/player", listPlayers)
		OptionalGroup.GET("/player/:player_uid", getPlayer)
	}

	authGroup := apiGroup.Group("")
	authGroup.Use(auth.JWTAuthMiddleware())
	{
		authGroup.POST("/server/broadcast", publishBroadcast)
		authGroup.POST("/server/shutdown", shutdownServer)
		authGroup.POST("/server/sync", syncData)
		authGroup.POST("/server/backup", createBackupNow)
		authGroup.GET("/server/paldefender/status", getPalDefenderStatus)
		authGroup.GET("/server/paldefender/audit", listPalDefenderAuditLogs)
		authGroup.GET("/server/paldefender/audit/export", exportPalDefenderAuditLogs)
		authGroup.POST("/server/paldefender/grant-batch", grantPalDefenderBatch)
		authGroup.POST("/server/paldefender/grant-batch/retry", retryPalDefenderBatch)
		authGroup.PUT("/player", putPlayers)
		authGroup.POST("/player/batch", batchPlayerAction)
		authGroup.GET("/player/search/items", searchPlayerItems)
		authGroup.GET("/player/search/pals", searchPlayerPals)
		authGroup.POST("/player/:player_uid/kick", kickPlayer)
		authGroup.POST("/player/:player_uid/ban", banPlayer)
		authGroup.POST("/player/:player_uid/unban", unbanPlayer)
		authGroup.POST("/player/:player_uid/items/grant", grantPlayerItems)
		authGroup.POST("/player/:player_uid/items/adjust", adjustPlayerItems)
		authGroup.POST("/player/:player_uid/items/clear", clearPlayerInventory)
		authGroup.POST("/player/:player_uid/support/grant", grantPlayerSupport)
		authGroup.POST("/player/:player_uid/pals/grant", grantPlayerPal)
		authGroup.POST("/player/:player_uid/pals/grant-egg", grantPlayerPalEgg)
		authGroup.POST("/player/:player_uid/pals/grant-template", grantPlayerPalTemplate)
		authGroup.POST("/player/:player_uid/pals/export", exportPlayerPals)
		authGroup.POST("/player/:player_uid/pals/delete", deletePlayerPals)
		authGroup.PUT("/guild", putGuilds)
		authGroup.POST("/sync", syncData)
		authGroup.GET("/whitelist", listWhite)
		authGroup.POST("/whitelist", addWhite)
		authGroup.DELETE("/whitelist", removeWhite)
		authGroup.PUT("/whitelist", putWhite)
		authGroup.GET("/rcon", listRconCommand)
		authGroup.POST("/rcon", addRconCommand)
		authGroup.POST("/rcon/import", importRconCommands)
		authGroup.POST("/rcon/preset", importRconPreset)
		authGroup.POST("/rcon/send", sendRconCommand)
		authGroup.POST("/rcon/raw", sendRawRconCommand)
		authGroup.PUT("/rcon/:uuid", putRconCommand)
		authGroup.DELETE("/rcon/:uuid", removeRconCommand)
		authGroup.GET("/backup", listBackups)
		authGroup.GET("/backup/:backup_id", downloadBackup)
		authGroup.DELETE("/backup/:backup_id", deleteBackup)
	}
}
