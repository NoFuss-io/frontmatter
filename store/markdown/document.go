package markdown

// LineEnding is the newline sequence used inside the frontmatter block. It is
// detected from the opening fence line and re-applied when the block is
// rendered, so CRLF files stay CRLF.
type LineEnding string

const (
	LF   LineEnding = "\n"
	CRLF LineEnding = "\r\n"
)

// Document is a file split around its first frontmatter block:
//
//	Head + LinePrefix + "---" + EOL
//	LinePrefix + <yaml line> + EOL   (one per YAML line)
//	LinePrefix + "---" + Body
//
// Head is everything before the opening fence, including any comment opener
// that shares its line ("/*", `"""`). Body is everything after the closing
// "---", including the rest of that line ("*/", `"""`) and its newline. Head
// and Body are kept verbatim, so only the block itself is ever rewritten.
type Document struct {
	FrontMatter map[string]any
	Head        string
	// LinePrefix is the comment marker repeated on every block line, including
	// both fences: "# " for hash comments, "// " for slash comments, "" for
	// Markdown and block comments.
	LinePrefix  string
	Body        string
	LineEndings LineEnding
}
