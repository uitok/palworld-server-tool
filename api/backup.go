package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/logger"
	"github.com/zaigie/palworld-server-tool/internal/tool"
	"github.com/zaigie/palworld-server-tool/service"
)

func listBackups(c *gin.Context) {
	var startTimestamp, endTimestamp int64
	var startTime, endTime time.Time
	var err error

	startTimeStr, endTimeStr := c.Query("startTime"), c.Query("endTime")

	if startTimeStr != "" {
		startTimestamp, err = strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			writeBadRequestCode(c, "invalid start time", "invalid_start_time")
			return
		}
		startTime = time.Unix(0, startTimestamp*int64(time.Millisecond))
	}

	if endTimeStr != "" {
		endTimestamp, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			writeBadRequestCode(c, "invalid end time", "invalid_end_time")
			return
		}
		endTime = time.Unix(0, endTimestamp*int64(time.Millisecond))
	}

	backups, err := service.ListBackups(getDB(), startTime, endTime)
	if err != nil {
		writeBadRequestErr(c, err)
		return
	}

	c.JSON(http.StatusOK, backups)
}

func backupFilePath(fileName string) (string, error) {
	backupDir, err := tool.GetBackupDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(backupDir, fileName), nil
}

func removeStaleBackupRecord(backupID string) {
	if err := service.DeleteBackup(getDB(), backupID); err != nil {
		logger.Errorf("failed to remove stale backup record %s: %v\n", backupID, err)
	}
}

func downloadBackup(c *gin.Context) {
	backupID := c.Param("backup_id")
	backup, err := service.GetBackup(getDB(), backupID)
	if err != nil {
		if err == service.ErrNoRecord {
			writeNotFound(c, "backup not found")
			return
		}
		writeBadRequestErr(c, err)
		return
	}

	filePath, err := backupFilePath(backup.Path)
	if err != nil {
		writeInternalErrorErr(c, err)
		return
	}
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			removeStaleBackupRecord(backupID)
			writeError(c, http.StatusNotFound, "backup file is missing; stale record removed", "backup_file_missing", gin.H{"backup_id": backupID, "path": backup.Path}, 0)
			return
		}
		writeInternalErrorErr(c, err)
		return
	}
	if info.IsDir() {
		writeInternalError(c, "backup path points to a directory")
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", backup.Path))
	c.File(filePath)
}

func deleteBackup(c *gin.Context) {
	backupID := c.Param("backup_id")
	backup, err := service.GetBackup(getDB(), backupID)
	if err != nil {
		if err == service.ErrNoRecord {
			writeNotFound(c, "backup not found")
			return
		}
		writeBadRequestErr(c, err)
		return
	}

	filePath, err := backupFilePath(backup.Path)
	if err != nil {
		writeInternalErrorErr(c, err)
		return
	}
	removeErr := os.Remove(filePath)
	if removeErr != nil && !os.IsNotExist(removeErr) {
		writeInternalErrorErr(c, removeErr)
		return
	}
	if err := service.DeleteBackup(getDB(), backupID); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	if os.IsNotExist(removeErr) {
		writeSuccessMessage(c, "backup record removed; backup file was already missing")
		return
	}
	writeSuccess(c)
}
