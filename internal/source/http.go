package source

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/zaigie/palworld-server-tool/internal/logger"
	"github.com/zaigie/palworld-server-tool/internal/system"
)

func DownloadFromHttp(url, way string) (string, error) {
	logger.Infof("downloading sav.zip from %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if resp.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("unexpected http status %d: %w", resp.StatusCode, os.ErrNotExist)
		}
		return "", fmt.Errorf("unexpected http status %d", resp.StatusCode)
	}

	uuid := uuid.New().String()
	tempPath := filepath.Join(os.TempDir(), "palworldsav-http-"+way+"-"+uuid)
	absPath, err := filepath.Abs(tempPath)
	if err != nil {
		return "", err
	}
	if err = system.CleanAndCreateDir(absPath); err != nil {
		return "", err
	}
	cleanupTempDir := true
	defer func() {
		if cleanupTempDir {
			_ = os.RemoveAll(absPath)
		}
	}()

	tempZipFilePath := filepath.Join(absPath, "sav.zip")
	defer os.Remove(tempZipFilePath)

	zipOut, err := os.Create(tempZipFilePath)
	if err != nil {
		return "", err
	}
	defer zipOut.Close()
	if _, err = io.Copy(zipOut, resp.Body); err != nil {
		return "", err
	}

	if err = system.UnzipDir(tempZipFilePath, absPath); err != nil {
		return "", err
	}
	levelFilePath, err := system.GetLevelSavFilePath(absPath)
	if err != nil {
		return "", err
	}
	logger.Info("sav.zip downloaded and extracted\n")
	cleanupTempDir = false
	return levelFilePath, nil
}
