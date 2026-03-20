package tool

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/zaigie/palworld-server-tool/internal/auth"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/internal/logger"
	"github.com/zaigie/palworld-server-tool/internal/source"
	"github.com/zaigie/palworld-server-tool/internal/system"
	"github.com/zaigie/palworld-server-tool/service"
	"go.etcd.io/bbolt"
)

type Sturcture struct {
	Players []database.Player `json:"players"`
	Guilds  []database.Guild  `json:"guilds"`
}

var (
	downloadFromHTTP   = source.DownloadFromHttp
	parseK8sAddress    = source.ParseK8sAddress
	copyFromPod        = source.CopyFromPod
	parseDockerAddress = source.ParseDockerAddress
	copyFromContainer  = source.CopyFromContainer
	copyFromLocal      = source.CopyFromLocal
)

func getSavCli() (string, error) {
	savCliPath := viper.GetString("save.decode_path")
	if savCliPath == "" || savCliPath == "/path/to/your/sav_cli" {
		ed, err := system.GetExecDir()
		if err != nil {
			logger.Errorf("error getting exec directory: %s", err)
			return "", err
		}
		savCliPath = filepath.Join(ed, "sav_cli")
		if runtime.GOOS == "windows" {
			savCliPath += ".exe"
		}
	}
	if _, err := os.Stat(savCliPath); err != nil {
		return "", err
	}
	return savCliPath, nil
}

func Decode(file string) error {
	logger.Infof("starting sav decode from %s source\n", detectSaveSourceKind(file))

	savCli, err := getSavCli()
	if err != nil {
		return wrapSaveDecodeError(file, "cli", err)
	}

	levelFilePath, err := getFromSource(file, "decode")
	if err != nil {
		return err
	}
	defer cleanupSaveTempDir(levelFilePath)
	if err := ensureLevelSaveFile(levelFilePath); err != nil {
		return wrapSaveSourceError(file, "validate", err)
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", viper.GetInt("web.port"))
	if viper.GetBool("web.tls") && !strings.HasSuffix(baseURL, "/") {
		baseURL = viper.GetString("web.public_url")
	}

	requestURL := fmt.Sprintf("%s/api/", baseURL)
	tokenString, err := auth.GenerateToken()
	if err != nil {
		return wrapSaveDecodeError(file, "token", err)
	}
	execArgs := []string{"-f", levelFilePath, "--request", requestURL, "--token", tokenString}
	cmd := exec.Command(savCli, execArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return wrapSaveDecodeError(file, "start", err)
	}
	if err := cmd.Wait(); err != nil {
		return wrapSaveDecodeError(file, "wait", err)
	}

	logger.Infof("sav decode finished for %s source\n", detectSaveSourceKind(file))
	return nil
}

func Backup() (string, error) {
	sourcePath := viper.GetString("save.path")
	logger.Infof("starting backup from %s source\n", detectSaveSourceKind(sourcePath))

	levelFilePath, err := getFromSource(sourcePath, "backup")
	if err != nil {
		return "", err
	}
	defer cleanupSaveTempDir(levelFilePath)
	if err := ensureLevelSaveFile(levelFilePath); err != nil {
		return "", wrapSaveSourceError(sourcePath, "validate", err)
	}

	backupDir, err := GetBackupDir()
	if err != nil {
		return "", fmt.Errorf("failed to get backup directory: %w", err)
	}

	backupName := fmt.Sprintf("%s.zip", time.Now().Format("2006-01-02-15-04-05"))
	backupZipFile := filepath.Join(backupDir, backupName)
	if err := system.ZipDir(filepath.Dir(levelFilePath), backupZipFile); err != nil {
		return "", fmt.Errorf("failed to create backup zip: %w", err)
	}
	logger.Infof("backup archive created: %s\n", backupName)
	return backupName, nil
}

func GetBackupDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	backDir := filepath.Join(wd, "backups")
	if err = system.CheckAndCreateDir(backDir); err != nil {
		return "", err
	}
	return backDir, nil
}

func CleanOldBackups(db *bbolt.DB, keepDays int) error {
	backupDir, err := GetBackupDir()
	if err != nil {
		return fmt.Errorf("failed to get backup directory: %s", err)
	}

	deadline := time.Now().AddDate(0, 0, -keepDays)
	backups, err := service.ListBackups(db, time.Time{}, time.Now())
	if err != nil {
		return fmt.Errorf("failed to list backups: %s", err)
	}

	removedFiles := 0
	removedRecords := 0
	missingFiles := 0
	staleRecords := 0
	invalidEntries := 0
	for _, backup := range backups {
		filePath := filepath.Join(backupDir, backup.Path)
		info, statErr := os.Stat(filePath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				if err := service.DeleteBackup(db, backup.BackupId); err != nil {
					logger.Errorf("failed to delete stale backup record from database: %s", err)
					continue
				}
				staleRecords++
				logger.Warnf("backup file missing, stale record removed: %s\n", backup.Path)
				continue
			}
			logger.Errorf("failed to stat backup file %s: %s\n", backup.Path, statErr)
			continue
		}
		if info.IsDir() {
			if err := service.DeleteBackup(db, backup.BackupId); err != nil {
				logger.Errorf("failed to delete invalid backup record from database: %s", err)
				continue
			}
			invalidEntries++
			logger.Warnf("backup path points to a directory, stale record removed: %s\n", backup.Path)
			continue
		}
		if !backup.SaveTime.Before(deadline) {
			continue
		}
		if err := os.Remove(filePath); err != nil {
			if os.IsNotExist(err) {
				missingFiles++
				logger.Warnf("backup file already missing during cleanup: %s\n", backup.Path)
			} else {
				logger.Errorf("failed to delete old backup file %s: %s\n", backup.Path, err)
				continue
			}
		} else {
			removedFiles++
		}

		if err := service.DeleteBackup(db, backup.BackupId); err != nil {
			logger.Errorf("failed to delete backup record from database: %s", err)
			continue
		}
		removedRecords++
	}

	if removedFiles > 0 || removedRecords > 0 || missingFiles > 0 || staleRecords > 0 || invalidEntries > 0 {
		logger.Infof("backup cleanup finished: removed_files=%d removed_records=%d missing_files=%d stale_records=%d invalid_entries=%d deadline=%s\n", removedFiles, removedRecords, missingFiles, staleRecords, invalidEntries, deadline.UTC().Format(time.RFC3339))
	}
	return nil
}

func getFromSource(file, way string) (string, error) {
	var levelFilePath string
	var err error

	if strings.HasPrefix(file, "http://") || strings.HasPrefix(file, "https://") {
		levelFilePath, err = downloadFromHTTP(file, way)
		if err != nil {
			return "", wrapSaveSourceError(file, "download", err)
		}
	} else if strings.HasPrefix(file, "k8s://") {
		namespace, podName, container, remotePath, err := parseK8sAddress(file)
		if err != nil {
			return "", wrapSaveSourceError(file, "parse", err)
		}
		levelFilePath, err = copyFromPod(namespace, podName, container, remotePath, way)
		if err != nil {
			return "", wrapSaveSourceError(file, "copy", err)
		}
	} else if strings.HasPrefix(file, "docker://") {
		containerID, remotePath, err := parseDockerAddress(file)
		if err != nil {
			return "", wrapSaveSourceError(file, "parse", err)
		}
		levelFilePath, err = copyFromContainer(containerID, remotePath, way)
		if err != nil {
			return "", wrapSaveSourceError(file, "copy", err)
		}
	} else {
		levelFilePath, err = copyFromLocal(file, way)
		if err != nil {
			return "", wrapSaveSourceError(file, "copy", err)
		}
	}
	return levelFilePath, nil
}

func cleanupSaveTempDir(levelFilePath string) {
	if strings.TrimSpace(levelFilePath) == "" {
		return
	}
	tempDir := filepath.Dir(levelFilePath)
	if !strings.HasPrefix(filepath.Base(tempDir), "palworldsav") {
		return
	}
	if err := os.RemoveAll(tempDir); err != nil {
		logger.Warnf("cleanup temporary save directory failed: %v\n", err)
	}
}

func ensureLevelSaveFile(levelFilePath string) error {
	if strings.TrimSpace(levelFilePath) == "" {
		return os.ErrNotExist
	}
	info, err := os.Stat(levelFilePath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("Level.sav path points to a directory: %s", levelFilePath)
	}
	if filepath.Base(levelFilePath) != "Level.sav" {
		return fmt.Errorf("unexpected save file name: %s", filepath.Base(levelFilePath))
	}
	return nil
}
