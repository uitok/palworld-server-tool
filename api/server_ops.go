package api

import (
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/zaigie/palworld-server-tool/internal/task"
	"github.com/zaigie/palworld-server-tool/internal/tool"
	"github.com/zaigie/palworld-server-tool/service"
)

var (
	serverInfoFunc                = tool.Info
	serverMetricsFunc             = tool.Metrics
	palDefenderStatusSnapshotFunc = tool.PalDefenderStatusSnapshot
	taskStatusSnapshotFunc        = task.StatusSnapshot
	runPlayerSyncNowFunc          = task.RunPlayerSyncNow
	runSaveSyncNowFunc            = task.RunSaveSyncNow
	runBackupNowFunc              = task.RunBackupNow
	listBackupsFunc               = service.ListBackups
)

type ServerDependencyStatus struct {
	Enabled    bool   `json:"enabled"`
	Configured bool   `json:"configured"`
	Checked    bool   `json:"checked"`
	Reachable  bool   `json:"reachable"`
	Mode       string `json:"mode,omitempty"`
	Error      string `json:"error,omitempty"`
	ErrorCode  string `json:"error_code,omitempty"`
}

type ServerCapabilities struct {
	RestEnabled        bool `json:"rest_enabled"`
	RconConfigured     bool `json:"rcon_configured"`
	PalDefenderEnabled bool `json:"paldefender_enabled"`
	PlayerSyncEnabled  bool `json:"player_sync_enabled"`
	SaveSyncEnabled    bool `json:"save_sync_enabled"`
	BackupEnabled      bool `json:"backup_enabled"`
	KickNonWhitelist   bool `json:"kick_non_whitelist"`
	PlayerLogging      bool `json:"player_logging"`
}

type ServerDependencies struct {
	REST        ServerDependencyStatus `json:"rest"`
	RCON        ServerDependencyStatus `json:"rcon"`
	PalDefender ServerDependencyStatus `json:"paldefender"`
	SaveSource  ServerDependencyStatus `json:"save_source"`
}

type ServerBackupSummary struct {
	BackupId   string    `json:"backup_id"`
	Path       string    `json:"path"`
	SaveTime   time.Time `json:"save_time"`
	FileExists bool      `json:"file_exists"`
	SizeBytes  int64     `json:"size_bytes,omitempty"`
}

type ServerOverviewResponse struct {
	Success      bool                    `json:"success"`
	CheckedAt    time.Time               `json:"checked_at"`
	PanelVersion string                  `json:"panel_version,omitempty"`
	Server       *ServerInfo             `json:"server,omitempty"`
	Metrics      *ServerMetrics          `json:"metrics,omitempty"`
	Tasks        task.TaskStatusSnapshot `json:"tasks"`
	LatestBackup *ServerBackupSummary    `json:"latest_backup,omitempty"`
	Capabilities ServerCapabilities      `json:"capabilities"`
	Dependencies ServerDependencies      `json:"dependencies"`
}

type ServerOperationResponse struct {
	Success    bool   `json:"success"`
	Action     string `json:"action"`
	Task       string `json:"task,omitempty"`
	Source     string `json:"source,omitempty"`
	Message    string `json:"message,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Details    any    `json:"details,omitempty"`
}

type SyncRequest struct {
	From From `json:"from"`
}

func writeOperationSuccess(c *gin.Context, action, taskName, source, message string, durationMs int64, details any) {
	c.JSON(200, ServerOperationResponse{
		Success:    true,
		Action:     action,
		Task:       taskName,
		Source:     source,
		Message:    message,
		DurationMs: durationMs,
		Details:    details,
	})
}

func writeOperationErr(c *gin.Context, fallbackCode string, err error) {
	if err == nil {
		writeBadRequestCode(c, "operation failed", fallbackCode)
		return
	}
	if code := tool.SaveOperationErrorCode(err); code != "" {
		writeBadRequestDetails(c, err.Error(), code, tool.SaveOperationErrorDetails(err), 1)
		return
	}
	writeError(c, 400, err.Error(), fallbackCode, nil, 0)
}

func detectOverviewSaveSourceMode(path string) string {
	trimmedPath := strings.TrimSpace(path)
	switch {
	case trimmedPath == "":
		return "disabled"
	case strings.HasPrefix(trimmedPath, "http://") || strings.HasPrefix(trimmedPath, "https://"):
		return "http"
	case strings.HasPrefix(trimmedPath, "docker://"):
		return "docker"
	case strings.HasPrefix(trimmedPath, "k8s://"):
		return "k8s"
	default:
		return "local"
	}
}

func buildServerCapabilities() ServerCapabilities {
	restConfigured := strings.TrimSpace(viper.GetString("rest.address")) != ""
	return ServerCapabilities{
		RestEnabled:        restConfigured || viper.GetInt("task.sync_interval") > 0 || viper.GetBool("manage.kick_non_whitelist"),
		RconConfigured:     strings.TrimSpace(viper.GetString("rcon.address")) != "" && strings.TrimSpace(viper.GetString("rcon.password")) != "",
		PalDefenderEnabled: viper.GetBool("paldefender.enabled"),
		PlayerSyncEnabled:  viper.GetInt("task.sync_interval") > 0,
		SaveSyncEnabled:    viper.GetInt("save.sync_interval") > 0,
		BackupEnabled:      viper.GetInt("save.backup_interval") > 0,
		KickNonWhitelist:   viper.GetBool("manage.kick_non_whitelist"),
		PlayerLogging:      viper.GetBool("task.player_logging"),
	}
}

func buildRCONDependencyStatus() ServerDependencyStatus {
	configured := strings.TrimSpace(viper.GetString("rcon.address")) != "" && strings.TrimSpace(viper.GetString("rcon.password")) != ""
	return ServerDependencyStatus{
		Enabled:    configured,
		Configured: configured,
		Checked:    false,
		Reachable:  false,
	}
}

func buildSaveSourceDependencyStatus() ServerDependencyStatus {
	savePath := strings.TrimSpace(viper.GetString("save.path"))
	mode := detectOverviewSaveSourceMode(savePath)
	status := ServerDependencyStatus{
		Enabled:    savePath != "",
		Configured: savePath != "",
		Checked:    false,
		Reachable:  false,
		Mode:       mode,
	}
	if !status.Configured {
		return status
	}
	if mode != "local" {
		return status
	}
	status.Checked = true
	if _, err := os.Stat(savePath); err != nil {
		status.Error = err.Error()
		status.ErrorCode = "save_source_not_found"
		return status
	}
	status.Reachable = true
	return status
}

func latestBackupSummary() *ServerBackupSummary {
	backups, err := listBackupsFunc(getDB(), time.Time{}, time.Time{})
	if err != nil || len(backups) == 0 {
		return nil
	}
	latest := backups[len(backups)-1]
	summary := &ServerBackupSummary{
		BackupId: latest.BackupId,
		Path:     latest.Path,
		SaveTime: latest.SaveTime,
	}
	filePath, err := backupFilePath(latest.Path)
	if err != nil {
		return summary
	}
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return summary
	}
	summary.FileExists = true
	summary.SizeBytes = info.Size()
	return summary
}

func getServerOverview(c *gin.Context) {
	panelVersion, _ := c.Get("version")
	capabilities := buildServerCapabilities()
	dependencies := ServerDependencies{
		REST: ServerDependencyStatus{
			Enabled:    capabilities.RestEnabled,
			Configured: strings.TrimSpace(viper.GetString("rest.address")) != "" && strings.TrimSpace(viper.GetString("rest.username")) != "" && strings.TrimSpace(viper.GetString("rest.password")) != "",
			Checked:    false,
			Reachable:  false,
		},
		RCON:       buildRCONDependencyStatus(),
		SaveSource: buildSaveSourceDependencyStatus(),
	}

	overview := ServerOverviewResponse{
		Success:      true,
		CheckedAt:    time.Now().UTC(),
		Tasks:        taskStatusSnapshotFunc(),
		LatestBackup: latestBackupSummary(),
		Capabilities: capabilities,
		Dependencies: dependencies,
	}
	if version, ok := panelVersion.(string); ok {
		overview.PanelVersion = version
	}

	if overview.Dependencies.REST.Configured {
		overview.Dependencies.REST.Checked = true
		if info, err := serverInfoFunc(); err == nil {
			overview.Server = &ServerInfo{Version: info["version"], Name: info["name"]}
			overview.Dependencies.REST.Reachable = true
		} else {
			overview.Dependencies.REST.Error = err.Error()
			overview.Dependencies.REST.ErrorCode = "rest_unreachable"
		}
		if metrics, err := serverMetricsFunc(); err == nil {
			overview.Metrics = &ServerMetrics{
				ServerFps:        metrics["server_fps"].(int),
				CurrentPlayerNum: metrics["current_player_num"].(int),
				ServerFrameTime:  metrics["server_frame_time"].(float64),
				MaxPlayerNum:     metrics["max_player_num"].(int),
				Uptime:           metrics["uptime"].(int),
				Days:             metrics["days"].(int),
			}
			overview.Dependencies.REST.Reachable = true
			overview.Dependencies.REST.Error = ""
			overview.Dependencies.REST.ErrorCode = ""
		} else if overview.Dependencies.REST.Error == "" {
			overview.Dependencies.REST.Error = err.Error()
			overview.Dependencies.REST.ErrorCode = "rest_unreachable"
		}
	}

	palStatus := palDefenderStatusSnapshotFunc()
	overview.Dependencies.PalDefender = ServerDependencyStatus{
		Enabled:    palStatus.Enabled,
		Configured: palStatus.Configured,
		Checked:    palStatus.Enabled,
		Reachable:  palStatus.Reachable,
		Error:      palStatus.Error,
		ErrorCode:  palStatus.ErrorCode,
	}

	c.JSON(200, overview)
}

func createBackupNow(c *gin.Context) {
	backupRecord, durationMs, code, err := runBackupNowFunc(getDB())
	if err != nil {
		writeOperationErr(c, code, err)
		return
	}
	writeOperationSuccess(c, "backup", task.TaskBackup, "manual", "backup completed", durationMs, gin.H{
		"backup": ServerBackupSummary{
			BackupId:   backupRecord.BackupId,
			Path:       backupRecord.Path,
			SaveTime:   backupRecord.SaveTime,
			FileExists: true,
		},
	})
}
