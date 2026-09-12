// Package generator handles static site generation.
package generator

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/rfwatson/rfw.is/internal/markdown"
)

// Generator generates a static site.
type Generator struct {
	fs embed.FS
}

// Page is a page.
type Page struct {
	Reader      io.Reader
	FrontMatter markdown.FrontMatter
}

// New returns a new [*Generator].
func New(fs embed.FS) *Generator {
	return &Generator{fs: fs}
}

// Generate generates the static site.
func (g *Generator) Generate(ctx context.Context) (map[string]Page, error) {
	site := make(map[string]Page)

	if err := fs.WalkDir(g.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		content, err := g.generatePage(path)
		if err != nil {
			return fmt.Errorf("generate page: %w", err)
		}

		// TODO: this is brittle, use a regexp or similar.
		htmlPath := strings.ReplaceAll(path, ".md", ".html")
		site[filepath.Base(htmlPath)] = content

		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk dir: %w", err)
	}

	return site, nil
}

func (g *Generator) generatePage(path string) (Page, error) {
	fptr, err := g.fs.Open(path)
	if err != nil {
		return Page{}, fmt.Errorf("open file: %w", err)
	}
	defer fptr.Close() //nolint:errcheck

	var html bytes.Buffer
	var fm markdown.FrontMatter
	if err := markdown.Parse(&html, &fm, fptr); err != nil {
		return Page{}, fmt.Errorf("parse markdown: %w", err)
	}

	return Page{
		Reader:      &html,
		FrontMatter: fm,
	}, nil
}
