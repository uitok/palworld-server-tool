package task

import (
	"sync"
	"time"

	"github.com/spf13/viper"
)

const (
	TaskPlayerSync   = "player_sync"
	TaskSaveSync     = "save_sync"
	TaskBackup       = "backup"
	TaskCacheCleanup = "cache_cleanup"
)

type TaskRunStatus struct {
	Name            string    `json:"name"`
	Enabled         bool      `json:"enabled"`
	Running         bool      `json:"running"`
	IntervalSeconds int       `json:"interval_seconds"`
	LastStartedAt   time.Time `json:"last_started_at,omitempty"`
	LastFinishedAt  time.Time `json:"last_finished_at,omitempty"`
	LastSuccessAt   time.Time `json:"last_success_at,omitempty"`
	LastDurationMs  int64     `json:"last_duration_ms,omitempty"`
	SuccessCount    int64     `json:"success_count"`
	FailureCount    int64     `json:"failure_count"`
	LastError       string    `json:"last_error,omitempty"`
	LastErrorCode   string    `json:"last_error_code,omitempty"`
}

type TaskStatusSnapshot struct {
	CheckedAt    time.Time     `json:"checked_at"`
	PlayerSync   TaskRunStatus `json:"player_sync"`
	SaveSync     TaskRunStatus `json:"save_sync"`
	Backup       TaskRunStatus `json:"backup"`
	CacheCleanup TaskRunStatus `json:"cache_cleanup"`
}

var (
	taskStatusMu sync.RWMutex
	taskStatuses = map[string]TaskRunStatus{}
)

func taskIntervalSeconds(name string) int {
	switch name {
	case TaskPlayerSync:
		return viper.GetInt("task.sync_interval")
	case TaskSaveSync:
		return viper.GetInt("save.sync_interval")
	case TaskBackup:
		return viper.GetInt("save.backup_interval")
	case TaskCacheCleanup:
		return 300
	default:
		return 0
	}
}

func taskEnabled(name string) bool {
	return taskIntervalSeconds(name) > 0
}

func currentTaskStatus(name string) TaskRunStatus {
	status := taskStatuses[name]
	status.Name = name
	status.Enabled = taskEnabled(name)
	status.IntervalSeconds = taskIntervalSeconds(name)
	return status
}

func markTaskStart(name string) time.Time {
	now := time.Now().UTC()
	taskStatusMu.Lock()
	defer taskStatusMu.Unlock()
	status := currentTaskStatus(name)
	status.Running = true
	status.LastStartedAt = now
	taskStatuses[name] = status
	return now
}

func markTaskSuccess(name string, startedAt time.Time) int64 {
	now := time.Now().UTC()
	durationMs := now.Sub(startedAt).Milliseconds()
	taskStatusMu.Lock()
	defer taskStatusMu.Unlock()
	status := currentTaskStatus(name)
	status.Running = false
	status.LastFinishedAt = now
	status.LastSuccessAt = now
	status.LastDurationMs = durationMs
	status.SuccessCount++
	status.LastError = ""
	status.LastErrorCode = ""
	taskStatuses[name] = status
	return durationMs
}

func markTaskFailure(name string, startedAt time.Time, err error, code string) int64 {
	now := time.Now().UTC()
	durationMs := now.Sub(startedAt).Milliseconds()
	taskStatusMu.Lock()
	defer taskStatusMu.Unlock()
	status := currentTaskStatus(name)
	status.Running = false
	status.LastFinishedAt = now
	status.LastDurationMs = durationMs
	status.FailureCount++
	if err != nil {
		status.LastError = err.Error()
	}
	status.LastErrorCode = code
	taskStatuses[name] = status
	return durationMs
}

func StatusSnapshot() TaskStatusSnapshot {
	taskStatusMu.RLock()
	defer taskStatusMu.RUnlock()
	return TaskStatusSnapshot{
		CheckedAt:    time.Now().UTC(),
		PlayerSync:   currentTaskStatus(TaskPlayerSync),
		SaveSync:     currentTaskStatus(TaskSaveSync),
		Backup:       currentTaskStatus(TaskBackup),
		CacheCleanup: currentTaskStatus(TaskCacheCleanup),
	}
}
