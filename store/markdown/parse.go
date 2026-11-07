package markdown

import "errors"

// Parse splits raw file content around its first frontmatter block. The
// file's extension (taken from path) selects which comment syntaxes are
// recognised. See Document for how the parts fit together.
//
// TODO(#21): implement.
func Parse(path string, raw []byte) (Document, error) {
	return Document{}, errors.New("markdown.Parse: not implemented")
}
