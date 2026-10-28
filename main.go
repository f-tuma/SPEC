package main

import (
	"context"
	"embed"
	"io/fs"
	"os"
	"os/signal"

	"github.com/f-tuma/SPEC/internal/cli"
	"github.com/f-tuma/SPEC/internal/spec"
)

// Include dot directories as well as Markdown, templates and manifests.
//
//go:embed all:plugin
var resources embed.FS

func main() {
	plugin, err := fs.Sub(resources, "plugin")
	if err != nil {
		panic(err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	os.Exit(cli.Run(ctx, os.Args[1:], spec.New(plugin), os.Stdin, os.Stdout, os.Stderr))
}
