// Package markdown wraps markdown-related functionality.
package markdown

import (
	"fmt"
	"io"
	"time"

	mdmeta "github.com/yuin/goldmark-meta/v2"
	"github.com/yuin/goldmark/v2/ast"
	mdparser "github.com/yuin/goldmark/v2/parser"
	mdrenderer "github.com/yuin/goldmark/v2/renderer/html"
)

// FrontMatter holds front matter metadata.
type FrontMatter struct {
	Title       string
	PublishedAt time.Time
}

// Parse parses the markdown file in source, and renders HTML to dest.
func Parse(dest io.Writer, fm *FrontMatter, source io.Reader) error {
	bytes, err := io.ReadAll(source)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	parser := mdparser.New(mdparser.WithExtensions(mdmeta.Parser))
	doc := parser.Parse(bytes)

	metadata := doc.(*ast.Document).Metadata()
	if err := buildFrontMatter(fm, metadata); err != nil {
		return fmt.Errorf("build front matter: %w", err)
	}

	renderer := mdrenderer.New(mdrenderer.WithUnsafe())

	if err := renderer.Render(dest, bytes, doc); err != nil {
		return fmt.Errorf("render: %w", err)
	}

	return nil
}

// goldmark-meta does not expose front matter in a way that can be parsed into a
// struct. For now, just do it manually.
func buildFrontMatter(fm *FrontMatter, metadata map[string]any) error {
	if s, ok := metadata["title"].(string); ok {
		fm.Title = s
	}

	if t, ok := metadata["published_at"].(time.Time); ok {
		fm.PublishedAt = t.UTC()
	}

	return nil
}
