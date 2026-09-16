// Package markdown wraps markdown-related functionality.
package markdown

import (
	"fmt"
	"io"
	"time"

	chromahtml "github.com/alecthomas/chroma/v3/formatters/html"
	highlighting "github.com/yuin/goldmark-highlighting/v3"
	mdmeta "github.com/yuin/goldmark-meta/v2"
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/extension"
	mdparser "github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/renderer/html"
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

	parser := mdparser.New(mdparser.WithExtensions(
		mdmeta.Parser,
		extension.StrikethroughParser,
		extension.TableParser,
		highlighting.Parser,
	))
	doc := parser.Parse(bytes)

	metadata := doc.(*ast.Document).Metadata()
	if err := buildFrontMatter(fm, metadata); err != nil {
		return fmt.Errorf("build front matter: %w", err)
	}

	renderer := html.New(
		html.WithUnsafe(),
		html.WithExtensions(
			extension.StrikethroughHTMLRenderer,
			extension.TableHTMLRenderer,
			highlighting.NewHTMLRenderer(
				highlighting.WithStyle("gruvbox"),
				highlighting.WithFormatterOptions(
					chromahtml.WithLineNumbers(true),
				),
			),
		),
	)

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
