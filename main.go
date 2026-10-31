package main

import (
	"embed"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// The built frontend is embedded into the binary.
//
//go:embed all:frontend/dist
var assets embed.FS

// webviewDataPath keeps the WebView2 profile inside the app's own config
// folder instead of a folder named after the binary.
func webviewDataPath() string {
	if v := os.Getenv("XIANGSU_WEBVIEW_DATA"); v != "" {
		_ = os.MkdirAll(v, 0o755)
		return v
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = "."
	}
	dir := filepath.Join(base, "xiangsu", "webview")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	return dir
}

// browserArgs returns extra Chromium flags.
//
// On some Windows machines a security suite blocks the WebView2 renderer
// sandbox, which makes the browser process exit before a window can be shown.
// The app only ever loads its own bundled assets and never navigates to remote
// pages, so the renderer runs unsandboxed by default. Set
// XIANGSU_WEBVIEW_SANDBOX=1 to keep the sandbox on.
//
// GPU compositing is off by default as well: on the same setups the renderer
// dies (window vanishes, process exits) as soon as a repaint is triggered by a
// click. Software rendering costs nothing here — the canvas is a few hundred
// bricks. Set XIANGSU_WEBVIEW_GPU=1 to turn hardware acceleration back on, and
// pass extra flags with XIANGSU_WEBVIEW_ARGS="--disable-gpu".
func browserArgs() []string {
	var args []string
	if os.Getenv("XIANGSU_WEBVIEW_SANDBOX") != "1" {
		args = append(args, "--no-sandbox")
	}
	if os.Getenv("XIANGSU_WEBVIEW_GPU") != "1" {
		args = append(args, "--disable-gpu")
	}
	if extra := os.Getenv("XIANGSU_WEBVIEW_ARGS"); extra != "" {
		args = append(args, strings.Split(extra, ",")...)
	}
	return args
}

func main() {
	logLevel := slog.LevelInfo
	if os.Getenv("XIANGSU_DEBUG") != "" {
		logLevel = slog.LevelDebug
	}
	app := application.New(application.Options{
		Name:        "像素积木",
		Description: "乐高微颗粒图设计工具",
		LogLevel:    logLevel,
		Services: []application.Service{
			application.NewService(&StudioService{}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Windows: application.WindowsOptions{
			WebviewUserDataPath:   webviewDataPath(),
			WebviewBrowserPath:    os.Getenv("XIANGSU_WEBVIEW_PATH"),
			AdditionalBrowserArgs: browserArgs(),
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "像素积木 · 乐高微颗粒图",
		Width:            1440,
		Height:           920,
		MinWidth:         1080,
		MinHeight:        700,
		BackgroundColour: application.NewRGB(246, 244, 239),
		Windows: application.WindowsWindow{
			Theme: application.Light,
		},
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 0,
			Backdrop:                application.MacBackdropNormal,
			TitleBar:                application.MacTitleBarDefault,
		},
		URL: "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
