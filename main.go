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
//	@in							header
//	@name						Authorization

// @license.name	Apache 2.0
// @license.url	http://www.apache.org/licenses/LICENSE-2.0.html
func main() {
	db := database.GetDB()
	defer db.Close()

	setupFlags()
	config.Init(cfgFile, &conf)
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
	api.RegisterRouter(router)

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

	router.GET("/", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, max-age=0")
		c.Writer.WriteHeader(http.StatusOK)
		file, _ := indexHTML.ReadFile("index.html")
		c.Writer.Write(file)
	})
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
  <div style="max-width:960px;margin:0 auto;">
    <h1 style="margin:0 0 16px;font-size:36px;color:#22c55e;">PST DIAG OK</h1>
    <p style="font-size:20px;margin:0 0 12px;">如果你能看到这行字，说明你已经打到当前服务了。</p>
    <div style="padding:16px;border-radius:12px;background:#1f2937;line-height:1.8;white-space:pre-wrap;word-break:break-all;">版本: %s
Host: %s
客户端IP: %s
User-Agent: %s
路径: /diag</div>
    <h2 style="margin:24px 0 12px;font-size:24px;">API 测试</h2>
    <pre id="api" style="padding:16px;border-radius:12px;background:#000;color:#93c5fd;min-height:120px;white-space:pre-wrap;">正在请求 /api/server ...</pre>
  </div>
  <script>
    const target = document.getElementById('api');
    fetch('/api/server', { cache: 'no-store' })
      .then(async (resp) => {
        const body = await resp.text();
        target.textContent = 'HTTP ' + resp.status + '\n' + body;
      })
      .catch((err) => {
        target.textContent = 'FETCH ERROR\n' + String(err);
      });
  </script>
</body>
</html>`, version, c.Request.Host, c.ClientIP(), c.Request.UserAgent())
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	})
	router.GET("/diag.txt", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, max-age=0")
		logger.Infof("Diag text requested from %s ua=%s\n", c.ClientIP(), c.Request.UserAgent())
		c.String(http.StatusOK, "PST DIAG OK\nversion=%s\nhost=%s\nclient_ip=%s\nuser_agent=%s\n", version, c.Request.Host, c.ClientIP(), c.Request.UserAgent())
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
