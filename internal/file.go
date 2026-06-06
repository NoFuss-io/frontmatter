package internal

type FilePath = string

type FrontMatter map[string]any

type Document struct {
	FrontMatter FrontMatter
	Body        string
}
