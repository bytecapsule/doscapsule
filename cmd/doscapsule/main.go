// DOS Capsule — pure-local DOS game frontend in a single Go binary.
//
// The SolidJS UI is embedded via go:embed. The same core (internal/app) is
// exposed through two entry points: the localhost HTTP API (internal/httpapi)
// and the CLI (internal/cli). This file is only the composition root.
package main

import (
	"embed"
	"io/fs"
	"log"
	"os"

	"github.com/bytecapsule/doscapsule/internal/app"
	"github.com/bytecapsule/doscapsule/internal/cli"
	"github.com/bytecapsule/doscapsule/internal/httpapi"
)

//go:embed all:dist
var distFS embed.FS

//go:embed resources/fallback-cover.jpg
var fallbackCover []byte

// version is stamped at build time via -X main.version=<v> (see Makefile).
var version = "dev"

func main() {
	ui, err := fs.Sub(distFS, "dist")
	if err != nil {
		log.Fatal(err)
	}

	a := app.New()
	code := cli.Run(os.Args[1:], cli.Deps{
		Version: version,
		App:     a,
		Serve: func(a *app.App, opts cli.ServeOptions) error {
			return httpapi.Serve(a, ui, fallbackCover, httpapi.Options{
				Port:        opts.Port,
				OpenBrowser: opts.OpenBrowser,
				Logf:        log.Printf,
			})
		},
	})
	os.Exit(code)
}
