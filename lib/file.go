package lib

import "io"

type frontMatter map[string]any

type Document struct {
	FrontMatter frontMatter
	Body        io.Reader
}
