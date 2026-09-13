// Package generator handles static site generation.
package generator

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rfwatson/rfw.is/internal/markdown"
)

// Generator generates a static site.
type Generator struct {
	fs        fs.FS
	tmpl      *template.Template
	blogPosts []Page
}

// Page is a page.
type Page struct {
	Reader      io.Reader
	Path        string
	FrontMatter markdown.FrontMatter
}

// New returns a new [*Generator].
func New(f fs.FS) (*Generator, error) {
	tmpl, err := template.New("").ParseFS(f, "blog/*.md", "layouts/*.html.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	return &Generator{fs: f, tmpl: tmpl}, nil
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

	// Finally, render scss.
	cssPages, err := g.generateCSS(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate CSS: %w", err)
	}

	for _, page := range cssPages {
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

	var postsHTML bytes.Buffer
	var fm markdown.FrontMatter
	if err := markdown.Parse(&postsHTML, &fm, &list); err != nil {
		return Page{}, fmt.Errorf("parse list: %w", err)
	}

	page, err := g.generatePage("blog/index.md", map[string]any{"Posts": template.HTML(postsHTML.String())})
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

// generatePage generates a full HTML page by combining a series of templates:
//
// 1. per-page markdown (render to HTML)
// 2. wrap with html/template template and metadata tags, based on markdown content
// 3. render the final HTML inside a layout
func (g *Generator) generatePage(path string, data any) (Page, error) {
	md, err := g.fs.Open(path)
	if err != nil {
		return Page{}, fmt.Errorf("open markdown: %w", err)
	}
	defer md.Close() //nolint:errcheck

	var html bytes.Buffer
	var fm markdown.FrontMatter
	if err = markdown.Parse(&html, &fm, md); err != nil {
		return Page{}, fmt.Errorf("parse markdown: %w", err)
	}

	var postHTML strings.Builder
	fmt.Fprintf(&postHTML, `{{- define "title"}}%s{{- end}}`, pageTitle(fm.Title))
	postHTML.WriteString(`{{define "content" -}}`)
	postHTML.Write(html.Bytes())
	postHTML.WriteString(`{{- end}}`)

	tmpl, err := g.tmpl.Clone()
	if err != nil {
		return Page{}, fmt.Errorf("clone templates: %w", err)
	}

	postTmpl, err := tmpl.Parse(postHTML.String())
	if err != nil {
		return Page{}, fmt.Errorf("new template: %w", err)
	}

	var pageHTML bytes.Buffer
	if err := postTmpl.ExecuteTemplate(&pageHTML, "layout", data); err != nil {
		return Page{}, fmt.Errorf("execute template: %w", err)
	}

	return Page{
		Reader:      &pageHTML,
		Path:        buildHTMLPath(path),
		FrontMatter: fm,
	}, nil
}

func (g *Generator) generateCSS(ctx context.Context) ([]Page, error) {
	var pages []Page

	if err := fs.WalkDir(g.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.Contains(path, "static/") {
			return nil
		}

		if !strings.HasSuffix(path, ".scss") {
			return nil
		}

		f, err := g.fs.Open(path)
		if err != nil {
			return fmt.Errorf("open scss file: %w", err)
		}
		defer f.Close() //nolint:errcheck

		var stdout, stderr bytes.Buffer
		cmd := exec.CommandContext(ctx, "sass", "--stdin")
		cmd.Stdin = f
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("run sass: %w (stderr: %s)", err, stderr.String())
		}

		outPath := strings.Replace(filepath.Base(path), ".scss", ".css", 1)
		pages = append(pages, Page{Reader: &stdout, Path: outPath})

		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk dir: %w", err)
	}

	return pages, nil
}
