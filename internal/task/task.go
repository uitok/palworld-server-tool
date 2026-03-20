package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/internal/system"

	"github.com/go-co-op/gocron/v2"
	"github.com/spf13/viper"
	"github.com/zaigie/palworld-server-tool/internal/logger"
	"github.com/zaigie/palworld-server-tool/internal/tool"
	"github.com/zaigie/palworld-server-tool/service"
	"go.etcd.io/bbolt"
)

var s gocron.Scheduler
var schedulerOnce sync.Once

func doBackup(db *bbolt.DB) (database.Backup, string, error) {
	path, err := tool.Backup()
	if err != nil {
		code := tool.SaveOperationErrorCode(err)
		if code == "" {
			code = "backup_failed"
		}
		return database.Backup{}, code, err
	}

	backupRecord := database.Backup{
		BackupId: uuid.New().String(),
		Path:     path,
		SaveTime: time.Now(),
	}
	if err := service.AddBackup(db, backupRecord); err != nil {
		if backupDir, dirErr := tool.GetBackupDir(); dirErr != nil {
			logger.Errorf("failed to resolve backup directory for cleanup: %v\n", dirErr)
		} else if removeErr := os.Remove(filepath.Join(backupDir, path)); removeErr != nil && !os.IsNotExist(removeErr) {
			logger.Errorf("failed to clean orphan backup archive %s: %v\n", path, removeErr)
		}
		return database.Backup{}, "backup_record_store_failed", err
	}

	keepDays := viper.GetInt("save.backup_keep_days")
	if keepDays == 0 {
		keepDays = 7
	}
	if err := tool.CleanOldBackups(db, keepDays); err != nil {
		return database.Backup{}, "backup_cleanup_failed", err
	}
	return backupRecord, "", nil
}

func runBackup(db *bbolt.DB) (database.Backup, int64, string, error) {
	startedAt := markTaskStart(TaskBackup)
	logger.Infof("task=%s started interval_seconds=%d\n", TaskBackup, viper.GetInt("save.backup_interval"))

	backupRecord, code, err := doBackup(db)
	if err != nil {
		durationMs := markTaskFailure(TaskBackup, startedAt, err, code)
		logger.Errorf("task=%s failed duration_ms=%d code=%s err=%v\n", TaskBackup, durationMs, code, err)
		return database.Backup{}, durationMs, code, err
	}

	keepDays := viper.GetInt("save.backup_keep_days")
	if keepDays == 0 {
		keepDays = 7
	}
	durationMs := markTaskSuccess(TaskBackup, startedAt)
	logger.Infof("task=%s completed duration_ms=%d backup=%s keep_days=%d\n", TaskBackup, durationMs, backupRecord.Path, keepDays)
	return backupRecord, durationMs, "", nil
}

func RunBackupNow(db *bbolt.DB) (database.Backup, int64, string, error) {
	return runBackup(db)
}

func BackupTask(db *bbolt.DB) {
	_, _, _, _ = runBackup(db)
}

func doPlayerSync(db *bbolt.DB) (int, string, error) {
	onlinePlayers, err := tool.ShowPlayers()
	if err != nil {
		return 0, "player_sync_fetch_failed", err
	}
	if err := service.PutPlayersOnline(db, onlinePlayers); err != nil {
		return 0, "player_sync_store_failed", err
	}

	if viper.GetBool("task.player_logging") {
		go PlayerLogging(onlinePlayers)
	}
	if viper.GetBool("manage.kick_non_whitelist") {
		go CheckAndKickPlayers(db, onlinePlayers)
	}
	return len(onlinePlayers), "", nil
}

func runPlayerSync(db *bbolt.DB) (int, int64, string, error) {
	startedAt := markTaskStart(TaskPlayerSync)
	logger.Infof("task=%s started interval_seconds=%d\n", TaskPlayerSync, viper.GetInt("task.sync_interval"))

	onlineCount, code, err := doPlayerSync(db)
	if err != nil {
		durationMs := markTaskFailure(TaskPlayerSync, startedAt, err, code)
		logger.Errorf("task=%s failed duration_ms=%d code=%s err=%v\n", TaskPlayerSync, durationMs, code, err)
		return 0, durationMs, code, err
	}

	durationMs := markTaskSuccess(TaskPlayerSync, startedAt)
	logger.Infof("task=%s completed duration_ms=%d online_players=%d\n", TaskPlayerSync, durationMs, onlineCount)
	return onlineCount, durationMs, "", nil
}

func RunPlayerSyncNow(db *bbolt.DB) (int, int64, string, error) {
	return runPlayerSync(db)
}

func PlayerSync(db *bbolt.DB) {
	_, _, _, _ = runPlayerSync(db)
}

func isPlayerWhitelisted(player database.OnlinePlayer, whitelist []database.PlayerW) bool {
	for _, whitelistedPlayer := range whitelist {
		if (player.PlayerUid != "" && player.PlayerUid == whitelistedPlayer.PlayerUID) ||
			(player.SteamId != "" && player.SteamId == whitelistedPlayer.SteamID) {
			return true
		}
	}
	return false
}

var playerCache map[string]string
var firstPoll = true

func PlayerLogging(players []database.OnlinePlayer) {
	loginMsg := viper.GetString("task.player_login_message")
	logoutMsg := viper.GetString("task.player_logout_message")

	tmp := make(map[string]string, len(players))
	for _, player := range players {
		if player.PlayerUid != "" {
			tmp[player.PlayerUid] = player.Nickname
		}
	}
	if !firstPoll {
		for id, name := range tmp {
			if _, ok := playerCache[id]; !ok {
				BroadcastVariableMessage(loginMsg, name, len(players))
			}
		}
		for id, name := range playerCache {
			if _, ok := tmp[id]; !ok {
				BroadcastVariableMessage(logoutMsg, name, len(players))
			}
		}
	}
	firstPoll = false
	playerCache = tmp
}

func BroadcastVariableMessage(message string, username string, onlineNum int) {
	message = strings.ReplaceAll(message, "{username}", username)
	message = strings.ReplaceAll(message, "{online_num}", strconv.Itoa(onlineNum))
	arr := strings.Split(message, "\n")
	for _, msg := range arr {
		err := tool.Broadcast(msg)
		if err != nil {
			logger.Warnf("Broadcast fail, %s \n", err)
		}
		time.Sleep(1000 * time.Millisecond)
	}
}

func CheckAndKickPlayers(db *bbolt.DB, players []database.OnlinePlayer) {
	whitelist, err := service.ListWhitelist(db)
	if err != nil {
		logger.Errorf("%v\n", err)
	}
	for _, player := range players {
		if !isPlayerWhitelisted(player, whitelist) {
			identifier := player.SteamId
			if identifier == "" {
				logger.Warnf("Kicked %s fail, SteamId is empty \n", player.Nickname)
				continue
			}
			err := tool.KickPlayer(fmt.Sprintf("steam_%s", identifier))
			if err != nil {
				logger.Warnf("Kicked %s fail, %s \n", player.Nickname, err)
				continue
			}
			logger.Warnf("Kicked %s successful \n", player.Nickname)
		}
	}
	logger.Info("Check whitelist done\n")
}

func doSaveSync() (string, error) {
	err := tool.Decode(viper.GetString("save.path"))
	if err != nil {
		code := tool.SaveOperationErrorCode(err)
		if code == "" {
			code = "save_sync_failed"
		}
		return code, err
	}
	return "", nil
}

func runSaveSync() (int64, string, error) {
	startedAt := markTaskStart(TaskSaveSync)
	logger.Infof("task=%s started interval_seconds=%d\n", TaskSaveSync, viper.GetInt("save.sync_interval"))

	code, err := doSaveSync()
	if err != nil {
		durationMs := markTaskFailure(TaskSaveSync, startedAt, err, code)
		logger.Errorf("task=%s failed duration_ms=%d code=%s err=%v\n", TaskSaveSync, durationMs, code, err)
		return durationMs, code, err
	}

	durationMs := markTaskSuccess(TaskSaveSync, startedAt)
	logger.Infof("task=%s completed duration_ms=%d\n", TaskSaveSync, durationMs)
	return durationMs, "", nil
}

func RunSaveSyncNow() (int64, string, error) {
	return runSaveSync()
}

func SavSync() {
	_, _, _ = runSaveSync()
}

func CacheCleanupTask() {
	startedAt := markTaskStart(TaskCacheCleanup)
	logger.Infof("task=%s started interval_seconds=%d\n", TaskCacheCleanup, 300)

	err := system.LimitCacheDir(filepath.Join(os.TempDir(), "palworldsav-"), 5)
	if err != nil {
		const code = "cache_cleanup_failed"
		durationMs := markTaskFailure(TaskCacheCleanup, startedAt, err, code)
		logger.Errorf("task=%s failed duration_ms=%d code=%s err=%v\n", TaskCacheCleanup, durationMs, code, err)
		return
	}

	durationMs := markTaskSuccess(TaskCacheCleanup, startedAt)
	logger.Infof("task=%s completed duration_ms=%d\n", TaskCacheCleanup, durationMs)
}

func Schedule(db *bbolt.DB) {
	s := getScheduler()

	playerSyncInterval := time.Duration(viper.GetInt("task.sync_interval"))
	savSyncInterval := time.Duration(viper.GetInt("save.sync_interval"))
	backupInterval := time.Duration(viper.GetInt("save.backup_interval"))
	logger.Infof("Scheduler config: player_sync=%ds save_sync=%ds backup=%ds cache_cleanup=%ds\n", int(playerSyncInterval), int(savSyncInterval), int(backupInterval), 300)

	if playerSyncInterval > 0 {
		go PlayerSync(db)
		_, err := s.NewJob(
			gocron.DurationJob(playerSyncInterval*time.Second),
			gocron.NewTask(PlayerSync, db),
		)
		if err != nil {
			logger.Errorf("%v\n", err)
		}
	}

	if savSyncInterval > 0 {
		go SavSync()
		_, err := s.NewJob(
			gocron.DurationJob(savSyncInterval*time.Second),
			gocron.NewTask(SavSync),
		)
		if err != nil {
			logger.Errorf("%v\n", err)
		}
	}

	if backupInterval > 0 {
		go BackupTask(db)
		_, err := s.NewJob(
			gocron.DurationJob(backupInterval*time.Second),
			gocron.NewTask(BackupTask, db),
		)
		if err != nil {
			logger.Error(err)
		}
	}

	_, err := s.NewJob(
		gocron.DurationJob(300*time.Second),
		gocron.NewTask(CacheCleanupTask),
	)
	if err != nil {
		logger.Errorf("%v\n", err)
	}

	s.Start()
}

func Shutdown() {
	s := getScheduler()
	err := s.Shutdown()
	if err != nil {
		logger.Errorf("%v\n", err)
	}
}

func initScheduler() gocron.Scheduler {
	s, err := gocron.NewScheduler()
	if err != nil {
		logger.Errorf("%v\n", err)
	}
	return s
}

func getScheduler() gocron.Scheduler {
	schedulerOnce.Do(func() {
		s = initScheduler()
	})
	return s
}
