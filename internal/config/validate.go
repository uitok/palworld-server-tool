package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/zaigie/palworld-server-tool/internal/system"
)

const (
	placeholderSavePath   = "/path/to/your/Pal/Saved"
	placeholderSavCliPath = "/path/to/your/sav_cli"
)

type ValidationIssue struct {
	Field   string
	Message string
}

type ValidationError struct {
	Issues []ValidationIssue
}

func (e *ValidationError) add(field, message string) {
	e.Issues = append(e.Issues, ValidationIssue{Field: field, Message: message})
}

func (e *ValidationError) HasIssues() bool {
	return len(e.Issues) > 0
}

func (e *ValidationError) Error() string {
	if len(e.Issues) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("configuration validation failed")
	for _, issue := range e.Issues {
		builder.WriteString("\n- ")
		builder.WriteString(issue.Field)
		builder.WriteString(": ")
		builder.WriteString(issue.Message)
	}
	return builder.String()
}

func Validate(conf *Config) error {
	issues := &ValidationError{}

	validateWeb(conf, issues)
	validateTask(conf, issues)
	validateRcon(conf, issues)
	validateRest(conf, issues)
	validateSave(conf, issues)
	validatePalDefender(conf, issues)

	if !issues.HasIssues() {
		return nil
	}
	return issues
}

func validateWeb(conf *Config, issues *ValidationError) {
	if strings.TrimSpace(conf.Web.Password) == "" {
		issues.add("web.password", "must not be empty")
	}
	if conf.Web.Port <= 0 || conf.Web.Port > 65535 {
		issues.add("web.port", "must be between 1 and 65535")
	}
	if !conf.Web.Tls {
		return
	}
	validateExistingFile("web.cert_path", conf.Web.CertPath, issues)
	validateExistingFile("web.key_path", conf.Web.KeyPath, issues)
}

func validateTask(conf *Config, issues *ValidationError) {
	if conf.Task.SyncInterval < 0 {
		issues.add("task.sync_interval", "must be greater than or equal to 0")
	}
	if conf.Task.PlayerLogging && conf.Task.SyncInterval == 0 {
		issues.add("task.player_logging", "requires task.sync_interval to be greater than 0")
	}
	if conf.Task.PlayerLogging {
		if strings.TrimSpace(conf.Task.PlayerLoginMessage) == "" {
			issues.add("task.player_login_message", "must not be empty when task.player_logging is enabled")
		}
		if strings.TrimSpace(conf.Task.PlayerLogoutMessage) == "" {
			issues.add("task.player_logout_message", "must not be empty when task.player_logging is enabled")
		}
	}
	if conf.Manage.KickNonWhitelist && conf.Task.SyncInterval == 0 {
		issues.add("manage.kick_non_whitelist", "requires task.sync_interval to be greater than 0")
	}
}

func validateRcon(conf *Config, issues *ValidationError) {
	configured := strings.TrimSpace(conf.Rcon.Address) != "" || strings.TrimSpace(conf.Rcon.Password) != "" || conf.Rcon.UseBase64
	if !configured {
		return
	}
	if strings.TrimSpace(conf.Rcon.Address) == "" {
		issues.add("rcon.address", "must not be empty when rcon is configured")
	} else {
		validateHostPort("rcon.address", conf.Rcon.Address, issues)
	}
	if strings.TrimSpace(conf.Rcon.Password) == "" {
		issues.add("rcon.password", "must not be empty when rcon is configured")
	}
	if conf.Rcon.Timeout <= 0 {
		issues.add("rcon.timeout", "must be greater than 0")
	}
}

func validateRest(conf *Config, issues *ValidationError) {
	needsREST := conf.Task.SyncInterval > 0 || conf.Manage.KickNonWhitelist
	if !needsREST {
		return
	}
	if strings.TrimSpace(conf.Rest.Address) == "" {
		issues.add("rest.address", "must not be empty when task.sync_interval is enabled")
	} else {
		validateHTTPURL("rest.address", conf.Rest.Address, issues)
	}
	if strings.TrimSpace(conf.Rest.Username) == "" {
		issues.add("rest.username", "must not be empty when task.sync_interval is enabled")
	}
	if strings.TrimSpace(conf.Rest.Password) == "" {
		issues.add("rest.password", "must not be empty when task.sync_interval is enabled")
	}
	if conf.Rest.Timeout <= 0 {
		issues.add("rest.timeout", "must be greater than 0")
	}
}

func validateSave(conf *Config, issues *ValidationError) {
	if conf.Save.SyncInterval < 0 {
		issues.add("save.sync_interval", "must be greater than or equal to 0")
	}
	if conf.Save.BackupInterval < 0 {
		issues.add("save.backup_interval", "must be greater than or equal to 0")
	}
	if conf.Save.BackupKeepDays < 0 {
		issues.add("save.backup_keep_days", "must be greater than or equal to 0")
	}

	needsSaveSource := conf.Save.SyncInterval > 0 || conf.Save.BackupInterval > 0
	if needsSaveSource {
		validateSavePath(conf.Save.Path, issues)
	}
	if conf.Save.SyncInterval == 0 {
		return
	}
	validateSavCli(conf.Save.DecodePath, issues)
	if conf.Web.Tls {
		if strings.TrimSpace(conf.Web.PublicUrl) == "" {
			issues.add("web.public_url", "must not be empty when web.tls is enabled and save.sync_interval is greater than 0")
		} else {
			validateHTTPURL("web.public_url", conf.Web.PublicUrl, issues)
		}
	}
}

func validatePalDefender(conf *Config, issues *ValidationError) {
	if !conf.PalDefender.Enabled {
		return
	}
	if strings.TrimSpace(conf.PalDefender.Address) == "" {
		issues.add("paldefender.address", "must not be empty when paldefender is enabled")
	} else {
		validateHTTPURLAllowImplicitScheme("paldefender.address", conf.PalDefender.Address, issues)
	}
	if strings.TrimSpace(conf.PalDefender.AuthKey) == "" {
		issues.add("paldefender.auth_key", "must not be empty when paldefender is enabled")
	}
	if conf.PalDefender.Timeout <= 0 {
		issues.add("paldefender.timeout", "must be greater than 0")
	}
}

func validateExistingFile(field, path string, issues *ValidationError) {
	path = strings.TrimSpace(path)
	if path == "" {
		issues.add(field, "must not be empty")
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		issues.add(field, fmt.Sprintf("file not accessible: %v", err))
		return
	}
	if info.IsDir() {
		issues.add(field, "must point to a file, not a directory")
	}
}

func validateHostPort(field, raw string, issues *ValidationError) {
	if _, _, err := net.SplitHostPort(strings.TrimSpace(raw)); err != nil {
		issues.add(field, "must use host:port format")
	}
}

func validateHTTPURL(field, raw string, issues *ValidationError) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil || parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
		issues.add(field, "must be a valid absolute HTTP(S) URL")
		return
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		issues.add(field, "must use http or https")
	}
}

func validateHTTPURLAllowImplicitScheme(field, raw string, issues *ValidationError) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		issues.add(field, "must not be empty")
		return
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	validateHTTPURL(field, raw, issues)
}

func validateSavePath(path string, issues *ValidationError) {
	path = strings.TrimSpace(path)
	if path == "" {
		issues.add("save.path", "must not be empty when save sync or backup is enabled")
		return
	}
	if path == placeholderSavePath {
		issues.add("save.path", "must be changed from the example placeholder path")
		return
	}
	switch {
	case strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://"):
		validateHTTPURL("save.path", path, issues)
	case strings.HasPrefix(path, "docker://"):
		validateDockerAddress(path, issues)
	case strings.HasPrefix(path, "k8s://"):
		validateK8sAddress(path, issues)
	default:
		info, err := os.Stat(path)
		if err != nil {
			issues.add("save.path", fmt.Sprintf("local path not accessible: %v", err))
			return
		}
		if !info.IsDir() && filepath.Base(path) != "Level.sav" {
			issues.add("save.path", "local file path must point to Level.sav or a directory containing it")
		}
	}
}

func validateDockerAddress(path string, issues *ValidationError) {
	payload := strings.TrimPrefix(path, "docker://")
	parts := strings.SplitN(payload, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		issues.add("save.path", "docker source must use docker://container:/path format")
	}
}

func validateK8sAddress(path string, issues *ValidationError) {
	payload := strings.TrimPrefix(path, "k8s://")
	parts := strings.SplitN(payload, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		issues.add("save.path", "k8s source must use k8s://namespace/pod/container:/path or k8s://pod/container:/path format")
		return
	}
	pathParts := strings.Split(parts[0], "/")
	if len(pathParts) != 2 && len(pathParts) != 3 {
		issues.add("save.path", "k8s source must use k8s://namespace/pod/container:/path or k8s://pod/container:/path format")
		return
	}
	for _, part := range pathParts {
		if strings.TrimSpace(part) == "" {
			issues.add("save.path", "k8s source path segments must not be empty")
			return
		}
	}
}

func validateSavCli(rawPath string, issues *ValidationError) {
	savCliPath, err := resolveSavCliPath(rawPath)
	if err != nil {
		issues.add("save.decode_path", fmt.Sprintf("cannot resolve sav_cli path: %v", err))
		return
	}
	if _, err := os.Stat(savCliPath); err != nil {
		issues.add("save.decode_path", fmt.Sprintf("sav_cli not found: %s", savCliPath))
	}
}

func resolveSavCliPath(rawPath string) (string, error) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath != "" && rawPath != placeholderSavCliPath {
		return rawPath, nil
	}
	execDir, err := system.GetExecDir()
	if err != nil {
		return "", err
	}
	savCliPath := filepath.Join(execDir, "sav_cli")
	if runtime.GOOS == "windows" {
		savCliPath += ".exe"
	}
	return savCliPath, nil
}
