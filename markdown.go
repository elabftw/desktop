package main

import (
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

func renderMarkdownToHTML(input string) string {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)

	htmlFlags := html.CommonFlags
	renderer := html.NewRenderer(html.RendererOptions{
		Flags: htmlFlags,
	})

	return string(markdown.ToHTML([]byte(input), p, renderer))
}
