package generator_test

import (
	"embed"
	"io"
	"io/fs"
	"testing"
	"time"

	"github.com/rfwatson/rfw.is/internal/generator"
	"github.com/rfwatson/rfw.is/internal/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/site-single-blog-post
var siteSingleBlogPost embed.FS

func TestGenerator(t *testing.T) {
	testCases := []struct {
		name     string
		fs       fs.FS
		wantSite map[string]func(*testing.T, string, markdown.FrontMatter)
		wantErr  string
	}{
		{
			name: "single blog post",
			fs:   mustBuildSubFS(t, siteSingleBlogPost, "testdata/site-single-blog-post"),
			wantSite: map[string]func(*testing.T, string, markdown.FrontMatter){
				"index.html": func(t *testing.T, content string, fm markdown.FrontMatter) {
					assert.Equal(t, "Blog", fm.Title)

					assert.Contains(t, content, "<html>")
					assert.Contains(t, content, "<title>Blog | rfw.is</title>")

					assert.Contains(t, content, "Hello world")
					assert.Contains(t, content, "/2026-01-01-hello-world.html")
				},
				"blog/2026-01-01-hello-world.html": func(t *testing.T, content string, fm markdown.FrontMatter) {
					assert.Equal(t, "Hello world", fm.Title)

					assert.Contains(t, content, "<html>")
					assert.Contains(t, content, "<title>Hello world | rfw.is</title>")

					assert.Equal(t, "Hello world", fm.Title)
					assert.Equal(t, time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), fm.PublishedAt)
				},
				"main.css": func(t *testing.T, content string, _ markdown.FrontMatter) {
					assert.Contains(t, content, "background-color: #ffffff")
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g, err := generator.New(tc.fs)
			require.NoError(t, err)

			site, err := g.Generate(t.Context())
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)

				require.Len(t, site, len(tc.wantSite), "expected %d pages but got %d", len(tc.wantSite), len(site))

				for path, page := range site {
					assertFunc, ok := tc.wantSite[path]
					require.Truef(t, ok, "no matching assert function for path %q", path)

					bytes, err := io.ReadAll(page.Reader)
					require.NoError(t, err)

					assertFunc(t, string(bytes), page.FrontMatter)
				}
			}
		})
	}
}

func mustBuildSubFS(t *testing.T, files fs.FS, path string) fs.FS {
	t.Helper()

	subFS, err := fs.Sub(files, path)
	require.NoError(t, err)

	return subFS
}
