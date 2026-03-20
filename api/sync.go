package api

import (
	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/task"
)

type From string

const (
	FromRest From = "rest"
	FromSav  From = "sav"
)

// syncData godoc
//
//	@Summary		Sync Data
//	@Description	Sync Data
//	@Tags			Sync
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			from	query		From	true	"from"	enum(rest,sav)
//
//	@Success		200		{object}	ServerOperationResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Router			/api/sync [post]
func syncData(c *gin.Context) {
	from := c.Query("from")
	if from == "" {
		var req SyncRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			from = string(req.From)
		}
	}

	switch From(from) {
	case FromRest:
		onlinePlayers, durationMs, code, err := runPlayerSyncNowFunc(getDB())
		if err != nil {
			writeOperationErr(c, code, err)
			return
		}
		writeOperationSuccess(c, "sync", task.TaskPlayerSync, string(FromRest), "player sync completed", durationMs, gin.H{"online_players": onlinePlayers})
	case FromSav:
		durationMs, code, err := runSaveSyncNowFunc()
		if err != nil {
			writeOperationErr(c, code, err)
			return
		}
		writeOperationSuccess(c, "sync", task.TaskSaveSync, string(FromSav), "save sync completed", durationMs, nil)
	default:
		writeBadRequestCode(c, "invalid from", "invalid_sync_source")
	}
}
