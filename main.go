// Command blog builds this site from Markdown, or serves it locally with live
// reload while you write.
//
//	go run . build          render content/ into public/
//	go run . build -drafts  include posts marked draft: true
//	go run . serve          preview on http://localhost:8080
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Romsnack/PersonalBlog/internal/config"
	"github.com/Romsnack/PersonalBlog/internal/render"
	"github.com/Romsnack/PersonalBlog/internal/serve"
)

const (
	configPath = "config.yaml"
	contentDir = "content"
	staticDir  = "static"
	outDir     = "public"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	cmd := "build"
	args := os.Args[1:]
	if len(args) > 0 && len(args[0]) > 0 && args[0][0] != '-' {
		cmd, args = args[0], args[1:]
	}

	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	drafts := fs.Bool("drafts", false, "include posts marked draft: true")
	addr := fs.String("addr", ":8080", "address for the preview server")
	if err := fs.Parse(args); err != nil {
		return err
	}

	switch cmd {
	case "build":
		cfg, err := config.Load(configPath)
		if err != nil {
			return err
		}
		if err := build(cfg, *drafts); err != nil {
			return err
		}
		fmt.Printf("built %s\n", outDir)
		return nil

	case "serve":
		cfg, err := config.Load(configPath)
		if err != nil {
			return err
		}
		// Locally the site is served from the root, so drop any deployment
		// subpath; and drafts are always worth seeing while writing.
		cfg.BasePath = ""
		cfg.Dev = true

		rebuild := func() error { return build(cfg, true) }
		if err := rebuild(); err != nil {
			return err
		}
		return serve.Run(*addr, outDir, []string{contentDir, staticDir}, rebuild)

	default:
		return fmt.Errorf("unknown command %q (want build or serve)", cmd)
	}
}

func build(cfg *config.Config, drafts bool) error {
	site, err := render.Load(cfg, contentDir, drafts)
	if err != nil {
		return err
	}
	return site.Build(outDir, staticDir)
}
