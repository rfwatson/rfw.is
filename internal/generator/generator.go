// Package generator handles static site generation.
package generator

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"
	"text/template"

	"github.com/rfwatson/rfw.is/internal/markdown"
)

// Generator generates a static site.
type Generator struct {
	fs        fs.FS
	blogPosts []Page
}

// Page is a page.
type Page struct {
	Reader      io.Reader
	Path        string
	FrontMatter markdown.FrontMatter
}

// New returns a new [*Generator].
func New(fs fs.FS) *Generator {
	return &Generator{fs: fs}
}

// Generate generates the static site.
func (g *Generator) Generate(ctx context.Context) (map[string]Page, error) {
	site := make(map[string]Page)

	// Start by generating the individual blog posts.
	if err := fs.WalkDir(g.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasPrefix(path, "blog/") {
			return nil
		}

		if strings.HasSuffix(path, "index.md") {
			return nil
		}

		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		if err := g.generateBlogPost(path); err != nil {
			return fmt.Errorf("generate page: %w", err)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk dir: %w", err)
	}

	// Now, render index.md.
	index, err := g.generateBlogIndex()
	if err != nil {
		return nil, fmt.Errorf("generate index: %w", err)
	}
	site["index.html"] = index

	for _, page := range g.blogPosts {
		site[page.Path] = page
	}

	return site, nil
}

func (g *Generator) generateBlogIndex() (Page, error) {
	blogPosts := slices.Clone(g.blogPosts)
	slices.SortFunc(blogPosts, func(a, b Page) int {
		return cmp.Compare(-a.FrontMatter.PublishedAt.Unix(), -b.FrontMatter.PublishedAt.Unix())
	})

	var list bytes.Buffer
	for _, blogPost := range blogPosts {
		fmt.Fprintf(&list, "- [%s](%s)\n", blogPost.FrontMatter.Title, blogPost.Path)
	}

	var listHTML bytes.Buffer
	var fm markdown.FrontMatter
	if err := markdown.Parse(&listHTML, &fm, &list); err != nil {
		return Page{}, fmt.Errorf("parse list: %w", err)
	}

	page, err := g.generatePage("blog/index.md", map[string]any{"Posts": listHTML.String()})
	if err != nil {
		return Page{}, err
	}

	return page, nil
}

func (g *Generator) generateBlogPost(path string) error {
	page, err := g.generatePage(path, nil)
	if err != nil {
		return err
	}

	g.blogPosts = append(g.blogPosts, page)

	return nil
}

func (g *Generator) generatePage(path string, data any) (Page, error) {
	fptr, err := g.fs.Open(path)
	if err != nil {
		return Page{}, fmt.Errorf("open file: %w", err)
	}
	defer fptr.Close() //nolint:errcheck

	content, err := io.ReadAll(fptr)
	if err != nil {
		return Page{}, fmt.Errorf("read file: %w", err)
	}

	tmpl, err := template.New("").Parse(string(content))
	if err != nil {
		return Page{}, fmt.Errorf("parse template: %w", err)
	}

	var rendered bytes.Buffer
	if err = tmpl.Execute(&rendered, data); err != nil {
		return Page{}, fmt.Errorf("execute template: %w", err)
	}

	var html bytes.Buffer
	var fm markdown.FrontMatter
	if err := markdown.Parse(&html, &fm, &rendered); err != nil {
		return Page{}, fmt.Errorf("parse markdown: %w", err)
	}

	// TODO: this is brittle, use a regexp or something.
	htmlPath := strings.ReplaceAll(path, ".md", ".html")

	return Page{
		Reader:      &html,
		Path:        htmlPath,
		FrontMatter: fm,
	}, nil
}
