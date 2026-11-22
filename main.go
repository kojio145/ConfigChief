// ConfigChief: the window of the tool. The steps themselves are in
// internal/ops, what the screens show is made in internal/ui; this package
// only opens the window and passes the calls on (spec 17.2).
//
// Started with a command ("ConfigChief.exe generate --project ..."), the same
// program runs the command line (internal/cli) and opens no window.
package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"github.com/kojio145/configchief/internal/cli"
	"github.com/kojio145/configchief/internal/errlog"
	"github.com/kojio145/configchief/internal/ui"
)

// version is set by the build: -ldflags "-X main.version=1.2.3".
var version = "0.0.0-dev"

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// A command: the answer goes to the terminal the program was started
	// from, and the program ends with the exit code of the command.
	if args := os.Args[1:]; cli.IsCommand(args) {
		out, errOut := terminal()
		cli.Version = version
		code := cli.Run(args, out, errOut)
		out.Sync()
		errOut.Sync()
		os.Exit(code)
	}
	app := NewApp(ui.New(version, ui.SettingsFile()))
	// A case folder given at the start is opened right away (spec 14.1).
	if len(os.Args) > 1 {
		app.openAtStart(os.Args[1])
	}
	errlog.Version = version
	err := wails.Run(&options.App{
		Title:     "ConfigChief",
		Width:     winWidth,
		Height:    winHeight,
		MinWidth:  winMinWidth,
		MinHeight: winMinHeight,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 246, G: 247, B: 249, A: 1},
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose,
		OnShutdown:       app.shutdown,
		Bind:             []interface{}{app},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	})
	if err != nil {
		println("error:", err.Error())
	}
}
