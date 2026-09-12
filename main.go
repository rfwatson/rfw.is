package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rfwatson/rfw.is/internal/generator"
)

//go:embed content/*
var fs embed.FS

const outPath = "./dist/"

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	g := generator.New(fs)

	site, err := g.Generate(ctx)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	// TODO: extract to writer type
	if err = os.RemoveAll(outPath); err != nil {
		return fmt.Errorf("remove all: %w", err)
	}

	if err = os.MkdirAll(outPath, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	for path, page := range site {
		contentPath := filepath.Join(outPath, path)
		fptr, err := os.Create(contentPath)
		if err != nil {
			return fmt.Errorf("create file %s: %w", contentPath, err)
		}

		if _, err = io.Copy(fptr, page.Reader); err != nil {
			fptr.Close() //nolint:errcheck
			return fmt.Errorf("copy: %w", err)
		}

		if err = fptr.Close(); err != nil {
			return fmt.Errorf("close file %s: %w", contentPath, err)
		}
	}

	return nil
}
