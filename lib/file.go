package lib

import "io"

type FrontMatter map[string]any

type Document struct {
	FrontMatter FrontMatter
	Body        io.Reader
}
