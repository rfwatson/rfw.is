package generator_test

import (
	"embed"
	"io"
	"testing"

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
		fs       embed.FS
		wantSite map[string]func(*testing.T, []byte, markdown.FrontMatter)
		wantErr  string
	}{
		{
			name: "single blog post",
			fs:   siteSingleBlogPost,
			wantSite: map[string]func(*testing.T, []byte, markdown.FrontMatter){
				"2026-01-01-hello-world.html": func(t *testing.T, bytes []byte, fm markdown.FrontMatter) {
					assert.Equal(t, "Hello world", fm.Title)
					assert.Equal(t, []byte("<h1>Hello world</h1>\n<p>This is a blog post.</p>\n"), bytes)
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := generator.New(tc.fs)
			site, err := g.Generate(t.Context())
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)

				require.Len(t, site, len(tc.wantSite))

				for path, page := range site {
					assertFunc, ok := tc.wantSite[path]
					require.Truef(t, ok, "no matching assert function for path %q", path)

					bytes, err := io.ReadAll(page.Reader)
					require.NoError(t, err)

					assertFunc(t, bytes, page.FrontMatter)
				}
			}
		})
	}
}
