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
	fs   fs.FS
	tmpl *template.Template
}

// A Site is a collection of documents.
type Site map[string]Document

// A Document is a piece of content written to a single file in the generated
// site - for example, a single HTML or CSS file.
type Document struct {
	Content     io.Reader            // Content is the content of the document.
	Path        string               // Path is the path to which the document should be written.
	FrontMatter markdown.FrontMatter // FrontMatter is any front matter which was associated with the source file.
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
func (g *Generator) Generate(ctx context.Context) (Site, error) {
	site := make(Site)

	// Blog posts
	blogPosts, err := g.generateBlogPosts()
	if err != nil {
		return nil, fmt.Errorf("generate blog posts: %w", err)
	}
	for _, resource := range blogPosts {
		site[resource.Path] = resource
	}

	// Blog index
	index, err := g.generateBlogIndex(blogPosts)
	if err != nil {
		return nil, fmt.Errorf("generate index: %w", err)
	}
	site["index.html"] = index

	// Pages
	pages, err := g.generatePages()
	if err != nil {
		return nil, fmt.Errorf("generate pages: %w", err)
	}
	for _, resource := range pages {
		site[resource.Path] = resource
	}

	// CSS
	cssDocs, err := g.generateCSS(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate CSS: %w", err)
	}
	for _, doc := range cssDocs {
		site[doc.Path] = doc
	}

	return site, nil
}

func (g *Generator) generateBlogPosts() ([]Document, error) {
	var blogPosts []Document

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

		doc, err := g.generateHTML(path, "layout_post", nil)
		if err != nil {
			return err
		}

		blogPosts = append(blogPosts, doc)

		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk dir: %w", err)
	}

	return blogPosts, nil
}

func (g *Generator) generatePages() ([]Document, error) {
	var pages []Document

	if err := fs.WalkDir(g.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// NOTE: for now, pages must be at root level of the content folder.
		if strings.Contains(path, "/") {
			return nil
		}

		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		doc, err := g.generateHTML(path, "layout_index", nil)
		if err != nil {
			return err
		}

		pages = append(pages, doc)

		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk dir: %w", err)
	}

	return pages, nil
}

func (g *Generator) generateBlogIndex(blogPosts []Document) (Document, error) {
	blogPosts = slices.Clone(blogPosts)
	slices.SortFunc(blogPosts, func(a, b Document) int {
		return cmp.Compare(-a.FrontMatter.PublishedAt.Unix(), -b.FrontMatter.PublishedAt.Unix())
	})

	var list bytes.Buffer
	for _, blogPost := range blogPosts {
		fmt.Fprintf(
			&list,
			"- %s - [%s](%s)\n",
			blogPost.FrontMatter.PublishedAt.Format(dateFormatDateOnly),
			blogPost.FrontMatter.Title,
			blogPost.Path,
		)
	}

	var postsHTML bytes.Buffer
	var fm markdown.FrontMatter
	if err := markdown.Parse(&postsHTML, &fm, &list); err != nil {
		return Document{}, fmt.Errorf("parse list: %w", err)
	}

	doc, err := g.generateHTML("blog/index.md", "layout_index", map[string]any{"Posts": template.HTML(postsHTML.String())})
	if err != nil {
		return Document{}, err
	}

	return doc, nil
}

const dateFormatDateOnly = "2006-01-02"

// generateHTML generates a full HTML document from the provided
// markdown source in a series of transformations:
//
//  1. render markdown to HTML
//  2. wrap with html/template template and metadata tags, based on markdown and
//     front mattercontent
//  3. render the generated HTML inside a layout
func (g *Generator) generateHTML(md string, layout string, data any) (Document, error) {
	f, err := g.fs.Open(md)
	if err != nil {
		return Document{}, fmt.Errorf("open markdown: %w", err)
	}
	defer f.Close() //nolint:errcheck

	var html bytes.Buffer
	var fm markdown.FrontMatter
	if err = markdown.Parse(&html, &fm, f); err != nil {
		return Document{}, fmt.Errorf("parse markdown: %w", err)
	}

	htmlPath := buildHTMLPath(md)

	var postHTML strings.Builder

	fmt.Fprintf(&postHTML, `{{- define "title"}}%s{{- end}}`, fm.Title)
	fmt.Fprintf(&postHTML, `{{- define "url"}}%s{{- end}}`, htmlPath)
	fmt.Fprintf(&postHTML, `{{- define "page_title"}}%s{{- end}}`, pageTitle(fm.Title))
	fmt.Fprintf(&postHTML, `{{- define "published_at"}}%s{{- end}}`, fm.PublishedAt.Format(dateFormatDateOnly))
	fmt.Fprintf(&postHTML, `{{- define "layout"}}{{block "%s" . }}{{end}}{{- end}}`, layout)

	postHTML.WriteString(`{{define "content" -}}`)
	postHTML.Write(html.Bytes())
	postHTML.WriteString(`{{- end}}`)

	tmpl, err := g.tmpl.Clone()
	if err != nil {
		return Document{}, fmt.Errorf("clone templates: %w", err)
	}

	postTmpl, err := tmpl.Parse(postHTML.String())
	if err != nil {
		return Document{}, fmt.Errorf("new template: %w", err)
	}

	var docHTML bytes.Buffer
	if err := postTmpl.ExecuteTemplate(&docHTML, "layout_main", data); err != nil {
		return Document{}, fmt.Errorf("execute template: %w", err)
	}

	return Document{
		Content:     &docHTML,
		Path:        htmlPath,
		FrontMatter: fm,
	}, nil
}

func (g *Generator) generateCSS(ctx context.Context) ([]Document, error) {
	var docs []Document

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
		docs = append(docs, Document{Content: &stdout, Path: outPath})

		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk dir: %w", err)
	}

	return docs, nil
}
