package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/zaigie/palworld-server-tool/api"
	"github.com/zaigie/palworld-server-tool/docs"
	"github.com/zaigie/palworld-server-tool/internal/config"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/internal/logger"
	"github.com/zaigie/palworld-server-tool/internal/system"
	"github.com/zaigie/palworld-server-tool/internal/task"
	"github.com/zaigie/palworld-server-tool/service"
)

var (
	version string = "Develop"
	cfgFile string
	conf    config.Config
)

//go:embed assets/*
var assets embed.FS

//go:embed index.html
var indexHTML embed.FS

//go:embed pal-conf.html
var palConfHTML embed.FS

//go:embed favicon.ico
var faviconICO embed.FS

//go:embed map/*
var mapTiles embed.FS

type diagConfigSnapshot struct {
	ConfigSource         string `json:"config_source"`
	WebPort              int    `json:"web_port"`
	WebTLS               bool   `json:"web_tls"`
	WebPublicURLSet      bool   `json:"web_public_url_set"`
	RestEnabled          bool   `json:"rest_enabled"`
	RestAddress          string `json:"rest_address"`
	RestTimeoutSeconds   int    `json:"rest_timeout_seconds"`
	RconConfigured       bool   `json:"rcon_configured"`
	RconAddress          string `json:"rcon_address"`
	RconTimeoutSeconds   int    `json:"rcon_timeout_seconds"`
	SaveMode             string `json:"save_mode"`
	SaveSourceConfigured bool   `json:"save_source_configured"`
	SaveSyncInterval     int    `json:"save_sync_interval"`
	BackupInterval       int    `json:"backup_interval"`
	BackupKeepDays       int    `json:"backup_keep_days"`
	PalDefenderEnabled   bool   `json:"paldefender_enabled"`
	PalDefenderAddress   string `json:"paldefender_address"`
	PalDefenderTimeout   int    `json:"paldefender_timeout_seconds"`
	KickNonWhitelist     bool   `json:"kick_non_whitelist"`
}

type diagSnapshot struct {
	CheckedAt time.Time               `json:"checked_at"`
	Version   string                  `json:"version"`
	Host      string                  `json:"host"`
	ClientIP  string                  `json:"client_ip"`
	UserAgent string                  `json:"user_agent"`
	Path      string                  `json:"path"`
	Config    diagConfigSnapshot      `json:"config"`
	Tasks     task.TaskStatusSnapshot `json:"tasks"`
}

func setupFlags() {
	flag.StringVar(&cfgFile, "config", "", "config file")
	flag.Parse()
}

func buildAssetAliases(assetsFS fs.FS) map[string]string {
	aliases := make(map[string]string)
	entries, err := fs.ReadDir(assetsFS, ".")
	if err != nil {
		return aliases
	}
	for _, entry := range entries {
		name := entry.Name()
		switch {
		case strings.HasPrefix(name, "index-") && strings.HasSuffix(name, ".js"):
			aliases["index.js"] = name
		case strings.HasPrefix(name, "index-") && strings.HasSuffix(name, ".css"):
			aliases["index.css"] = name
		case strings.HasPrefix(name, "Home-") && strings.HasSuffix(name, ".js"):
			aliases["Home.js"] = name
		case strings.HasPrefix(name, "Home-") && strings.HasSuffix(name, ".css"):
			aliases["Home.css"] = name
		}
	}
	return aliases
}

func resolveAssetAlias(requestPath string, aliases map[string]string) string {
	switch {
	case strings.HasPrefix(requestPath, "index-") && strings.HasSuffix(requestPath, ".js"):
		return aliases["index.js"]
	case strings.HasPrefix(requestPath, "index-") && strings.HasSuffix(requestPath, ".css"):
		return aliases["index.css"]
	case strings.HasPrefix(requestPath, "Home-") && strings.HasSuffix(requestPath, ".js"):
		return aliases["Home.js"]
	case strings.HasPrefix(requestPath, "Home-") && strings.HasSuffix(requestPath, ".css"):
		return aliases["Home.css"]
	default:
		return ""
	}
}

//	@SecurityDefinitions.apikey	ApiKeyAuth
//	@in						header
//	@name					Authorization

// @license.name	Apache 2.0
// @license.url	http://www.apache.org/licenses/LICENSE-2.0.html
func detectSaveSourceMode(path string) string {
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

func currentConfigSource() string {
	configSource := viper.ConfigFileUsed()
	if strings.TrimSpace(configSource) == "" {
		return "environment/defaults"
	}
	return configSource
}

func buildDiagConfigSnapshot(conf config.Config) diagConfigSnapshot {
	return diagConfigSnapshot{
		ConfigSource:         currentConfigSource(),
		WebPort:              conf.Web.Port,
		WebTLS:               conf.Web.Tls,
		WebPublicURLSet:      strings.TrimSpace(conf.Web.PublicUrl) != "",
		RestEnabled:          conf.Task.SyncInterval > 0 || conf.Manage.KickNonWhitelist,
		RestAddress:          conf.Rest.Address,
		RestTimeoutSeconds:   conf.Rest.Timeout,
		RconConfigured:       strings.TrimSpace(conf.Rcon.Address) != "" && strings.TrimSpace(conf.Rcon.Password) != "",
		RconAddress:          conf.Rcon.Address,
		RconTimeoutSeconds:   conf.Rcon.Timeout,
		SaveMode:             detectSaveSourceMode(conf.Save.Path),
		SaveSourceConfigured: strings.TrimSpace(conf.Save.Path) != "",
		SaveSyncInterval:     conf.Save.SyncInterval,
		BackupInterval:       conf.Save.BackupInterval,
		BackupKeepDays:       conf.Save.BackupKeepDays,
		PalDefenderEnabled:   conf.PalDefender.Enabled,
		PalDefenderAddress:   conf.PalDefender.Address,
		PalDefenderTimeout:   conf.PalDefender.Timeout,
		KickNonWhitelist:     conf.Manage.KickNonWhitelist,
	}
}

func buildDiagSnapshot(c *gin.Context) diagSnapshot {
	return diagSnapshot{
		CheckedAt: time.Now().UTC(),
		Version:   version,
		Host:      c.Request.Host,
		ClientIP:  c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Path:      c.Request.URL.Path,
		Config:    buildDiagConfigSnapshot(conf),
		Tasks:     task.StatusSnapshot(),
	}
}

func formatDiagTaskStatus(label string, status task.TaskRunStatus) string {
	return fmt.Sprintf("%s: enabled=%t running=%t interval_seconds=%d success_count=%d failure_count=%d last_error_code=%s last_error=%s",
		label, status.Enabled, status.Running, status.IntervalSeconds, status.SuccessCount, status.FailureCount, status.LastErrorCode, status.LastError)
}

func formatDiagText(snapshot diagSnapshot) string {
	return fmt.Sprintf(`PST DIAG OK
checked_at=%s
version=%s
host=%s
client_ip=%s
user_agent=%s
path=%s
config_source=%s
web_port=%d
web_tls=%t
rest_enabled=%t
rest_address=%s
rcon_configured=%t
rcon_address=%s
save_mode=%s
paldefender_enabled=%t
player_sync=%s
save_sync=%s
backup=%s
cache_cleanup=%s
`,
		snapshot.CheckedAt.Format(time.RFC3339),
		snapshot.Version,
		snapshot.Host,
		snapshot.ClientIP,
		snapshot.UserAgent,
		snapshot.Path,
		snapshot.Config.ConfigSource,
		snapshot.Config.WebPort,
		snapshot.Config.WebTLS,
		snapshot.Config.RestEnabled,
		snapshot.Config.RestAddress,
		snapshot.Config.RconConfigured,
		snapshot.Config.RconAddress,
		snapshot.Config.SaveMode,
		snapshot.Config.PalDefenderEnabled,
		formatDiagTaskStatus("player_sync", snapshot.Tasks.PlayerSync),
		formatDiagTaskStatus("save_sync", snapshot.Tasks.SaveSync),
		formatDiagTaskStatus("backup", snapshot.Tasks.Backup),
		formatDiagTaskStatus("cache_cleanup", snapshot.Tasks.CacheCleanup),
	)
}

func logStartupConfiguration(conf config.Config) {
	logger.Infof("Config source: %s\n", currentConfigSource())
	logger.Infof("Web config: port=%d tls=%t public_url_set=%t\n", conf.Web.Port, conf.Web.Tls, strings.TrimSpace(conf.Web.PublicUrl) != "")
	logger.Infof("REST config: enabled=%t address=%s timeout=%ds\n", conf.Task.SyncInterval > 0 || conf.Manage.KickNonWhitelist, conf.Rest.Address, conf.Rest.Timeout)
	logger.Infof("RCON config: configured=%t address=%s timeout=%ds base64=%t\n", strings.TrimSpace(conf.Rcon.Password) != "", conf.Rcon.Address, conf.Rcon.Timeout, conf.Rcon.UseBase64)
	logger.Infof("Save config: mode=%s sync_interval=%ds backup_interval=%ds keep_days=%d\n", detectSaveSourceMode(conf.Save.Path), conf.Save.SyncInterval, conf.Save.BackupInterval, conf.Save.BackupKeepDays)
	logger.Infof("PalDefender config: enabled=%t address=%s timeout=%ds\n", conf.PalDefender.Enabled, conf.PalDefender.Address, conf.PalDefender.Timeout)
	logger.Infof("Manage config: kick_non_whitelist=%t\n", conf.Manage.KickNonWhitelist)
}

func main() {
	setupFlags()
	config.Init(cfgFile, &conf)
	if err := config.Validate(&conf); err != nil {
		logger.Errorf("%s\n", err)
		os.Exit(1)
	}
	logStartupConfiguration(conf)

	db := database.GetDB()
	defer db.Close()

	if err := service.EnsureDefaultRconCommands(db); err != nil {
		logger.Errorf("Ensure default RCON commands failed: %v\n", err)
	}

	docs.SwaggerInfo.Title = "Palworld Manage API"
	docs.SwaggerInfo.Version = version
	docs.SwaggerInfo.Host = fmt.Sprintf("127.0.0.1:%d", viper.GetInt("web.port"))
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"http"}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("version", version)
		c.Next()
	})
	api.RegisterRouter(router, db)

	assetsFS, _ := fs.Sub(assets, "assets")
	assetAliases := buildAssetAliases(assetsFS)
	router.GET("/assets/*filepath", func(c *gin.Context) {
		filePath := strings.TrimPrefix(c.Param("filepath"), "/")
		if filePath == "" {
			c.Status(http.StatusNotFound)
			return
		}
		if _, err := fs.Stat(assetsFS, filePath); err == nil {
			c.FileFromFS(filePath, http.FS(assetsFS))
			return
		}
		if alias := resolveAssetAlias(filePath, assetAliases); alias != "" {
			c.Header("Cache-Control", "no-store")
			c.FileFromFS(alias, http.FS(assetsFS))
			return
		}
		c.Status(http.StatusNotFound)
	})
	router.GET("/favicon.ico", func(c *gin.Context) {
		c.FileFromFS("favicon.ico", http.FS(faviconICO))
	})
	router.HEAD("/favicon.ico", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	mapTilesFS, _ := fs.Sub(mapTiles, "map")
	router.StaticFS("/map/tiles", http.FS(mapTilesFS))

	serveIndex := func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, max-age=0")
		c.Writer.WriteHeader(http.StatusOK)
		file, _ := indexHTML.ReadFile("index.html")
		c.Writer.Write(file)
	}
	router.GET("/", serveIndex)
	router.GET("/ops", serveIndex)
	router.GET("/players", serveIndex)
	router.GET("/paldefender", serveIndex)
	router.GET("/pal-conf", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, max-age=0")
		c.Writer.WriteHeader(http.StatusOK)
		file, _ := palConfHTML.ReadFile("pal-conf.html")
		c.Writer.Write(file)
	})
	router.GET("/diag", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, max-age=0")
		logger.Infof("Diag page requested from %s ua=%s\n", c.ClientIP(), c.Request.UserAgent())
		html := fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>PST Diag</title>
</head>
<body style="margin:0;padding:24px;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#111827;color:#f9fafb;">
  <div style="max-width:1100px;margin:0 auto;">
    <h1 style="margin:0 0 16px;font-size:36px;color:#22c55e;">PST DIAG OK</h1>
    <p style="font-size:20px;margin:0 0 12px;">如果你能看到这行字，说明你已经打到当前服务了。</p>
    <div style="padding:16px;border-radius:12px;background:#1f2937;line-height:1.8;white-space:pre-wrap;word-break:break-all;">版本: %s
Host: %s
客户端IP: %s
User-Agent: %s
路径: /diag</div>
    <h2 style="margin:24px 0 12px;font-size:24px;">API 测试</h2>
    <pre id="api" style="padding:16px;border-radius:12px;background:#000;color:#93c5fd;min-height:120px;white-space:pre-wrap;">正在请求 /api/server ...</pre>
    <h2 style="margin:24px 0 12px;font-size:24px;">诊断快照</h2>
    <pre id="diagjson" style="padding:16px;border-radius:12px;background:#000;color:#86efac;min-height:240px;white-space:pre-wrap;">正在请求 /diag.json ...</pre>
  </div>
  <script>
    const apiTarget = document.getElementById('api');
    const diagTarget = document.getElementById('diagjson');
    fetch('/api/server', { cache: 'no-store' })
      .then(async (resp) => {
        const body = await resp.text();
        apiTarget.textContent = 'HTTP ' + resp.status + '\n' + body;
      })
      .catch((err) => {
        apiTarget.textContent = 'FETCH ERROR\n' + String(err);
      });
    fetch('/diag.json', { cache: 'no-store' })
      .then(async (resp) => {
        const body = await resp.json();
        diagTarget.textContent = JSON.stringify(body, null, 2);
      })
      .catch((err) => {
        diagTarget.textContent = 'FETCH ERROR\n' + String(err);
      });
  </script>
</body>
</html>`, version, c.Request.Host, c.ClientIP(), c.Request.UserAgent())
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	})
	router.GET("/diag.txt", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, max-age=0")
		logger.Infof("Diag text requested from %s ua=%s\n", c.ClientIP(), c.Request.UserAgent())
		c.String(http.StatusOK, formatDiagText(buildDiagSnapshot(c)))
	})
	router.GET("/diag.json", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, max-age=0")
		logger.Infof("Diag json requested from %s ua=%s\n", c.ClientIP(), c.Request.UserAgent())
		c.JSON(http.StatusOK, buildDiagSnapshot(c))
	})

	localIp, err := system.GetLocalIP()
	if err != nil {
		logger.Errorf("%v\n", err)
	}
	logger.Info("Starting PalWorld Server Tool...\n")
	logger.Infof("Version: %s\n", version)
	logger.Infof("Listening on http://127.0.0.1:%d or http://%s:%d\n", viper.GetInt("web.port"), localIp, viper.GetInt("web.port"))
	logger.Infof("Swagger on http://127.0.0.1:%d/swagger/index.html\n", viper.GetInt("web.port"))

	go task.Schedule(db)
	defer task.Shutdown()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if viper.GetBool("web.tls") {
			if err := router.RunTLS(fmt.Sprintf(":%d", viper.GetInt("web.port")), viper.GetString("web.cert_path"), viper.GetString("web.key_path")); err != nil {
				logger.Errorf("Server exited with TLS error: %v\n", err)
			}
		} else {
			if err := router.Run(fmt.Sprintf(":%d", viper.GetInt("web.port"))); err != nil {
				logger.Errorf("Server exited with error: %v\n", err)
			}
		}
	}()

	<-sigChan

	logger.Info("Server gracefully stopped\n")
}
