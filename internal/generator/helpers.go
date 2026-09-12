package generator

import (
	"strings"
)

const (
	pageTitleDefault = "rfw.is"
	pageTitleSuffix  = " | rfw.is"
)

func pageTitle(title string) string {
	if title == "" {
		return pageTitleDefault
	}

	return title + pageTitleSuffix
}

func buildHTMLPath(path string) string {
	// This is a bit brittle and would break if a path contained two instances of
	// .md. Seems unlikely enough to tolerate for now.
	return strings.Replace(path, ".md", ".html", 1)
}
