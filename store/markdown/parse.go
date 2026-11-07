package markdown

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// line is one line of the source. text excludes the line ending.
type line struct {
	start, end int // byte offsets; end is just past the line ending
	text       string
}

func splitLines(s string) []line {
	var out []line
	for start := 0; start < len(s); {
		end := len(s)
		if i := strings.IndexByte(s[start:], '\n'); i >= 0 {
			end = start + i + 1
		}
		text := strings.TrimSuffix(strings.TrimSuffix(s[start:end], "\n"), "\r")
		out = append(out, line{start: start, end: end, text: text})
		start = end
	}
	return out
}

func isFence(s string) bool { return strings.TrimRight(s, " \t") == "---" }

func endingOf(s string, l line) LineEnding {
	if strings.HasSuffix(s[l.start:l.end], "\r\n") {
		return CRLF
	}
	return LF
}

// Parse splits raw file content around its first frontmatter block. The
// file's extension (taken from path) selects which comment syntaxes are
// recognised. See Document for how the parts fit together.
//
// Markdown files only recognise a block on the first line, and invalid YAML
// in it is an error. In other files the whole file is scanned, and a
// candidate whose content is not a YAML mapping (e.g. a "# ---" separator
// comment) is skipped. A file without a block yields an empty FrontMatter and
// a Document that renders a new block at the insertion point.
func Parse(path string, raw []byte) (Document, error) {
	doc, _, err := parseAs(path, raw, languageFor(path, string(raw)))
	return doc, err
}

// parseAs is Parse with a given language. found reports whether the file had
// a block, as opposed to a Document positioned for inserting one.
func parseAs(path string, raw []byte, lang language) (doc Document, found bool, err error) {
	s := string(raw)
	lines := splitLines(s)

	if lang.markdown {
		if len(lines) > 0 && isFence(lines[0].text) {
			doc, ok, err := collect(s, lines, 0, 1, "", nil, "")
			if err != nil {
				return Document{}, false, fmt.Errorf("%s: %w", path, err)
			}
			if ok {
				return doc, true, nil
			}
		}
		return emptyDocument(s, lines, lang), false, nil
	}

	for i, l := range lines {
		for _, m := range lang.line {
			for _, p := range linePrefixes(m) {
				if rest, ok := strings.CutPrefix(l.text, p); ok && isFence(rest) {
					if doc, ok, _ := collect(s, lines, i, i+1, p, &m, ""); ok {
						return doc, true, nil
					}
				}
			}
		}
		for _, b := range lang.block {
			t := strings.TrimRight(l.text, " \t")
			switch {
			case t == b.open+"---" && b.open != "=begin":
				// "/*---": Head ends with the opener, fence shares its line.
				if doc, ok, _ := collect(s, lines, i, i+1, "", nil, b.close); ok {
					doc.Head = s[:l.start+len(b.open)]
					return doc, true, nil
				}
			case t == b.open && i+1 < len(lines) && isFence(lines[i+1].text):
				// "/*" alone, "---" on the next line.
				if doc, ok, _ := collect(s, lines, i+1, i+2, "", nil, b.close); ok {
					return doc, true, nil
				}
			}
		}
	}
	return emptyDocument(s, lines, lang), false, nil
}

// linePrefixes returns the accepted prefixes for a line-comment marker, most
// specific first. The space-less form is skipped for markers ending in '-',
// where "-----" would otherwise read as a fence.
func linePrefixes(marker string) []string {
	if strings.HasSuffix(marker, "-") {
		return []string{marker + " "}
	}
	return []string{marker + " ", marker}
}

// collect reads a block whose opening fence is on lines[fence] and whose
// content starts at lines[first]. prefix is stripped from every line; a line
// that is only the marker (bare) is an empty YAML line. closer, when set, is
// the block-comment close that must follow the closing fence, on the same
// line or the next. It reports ok=false when the candidate is not a valid
// block, and returns err only for YAML that fails to parse.
func collect(s string, lines []line, fence, first int, prefix string, marker *string, closer string) (Document, bool, error) {
	var yamlLines []string
	for k := first; k < len(lines); k++ {
		t := lines[k].text
		switch {
		case prefix == "":
		case strings.HasPrefix(t, prefix):
			t = t[len(prefix):]
		case strings.HasPrefix(t, *marker):
			t = t[len(*marker):]
		default:
			return Document{}, false, nil
		}
		fenceAt := lines[k].start + len(lines[k].text) - len(t) // offset of t in s

		closed := false
		switch {
		case closer == "":
			closed = isFence(t)
		case strings.TrimRight(t, " \t") == "---"+closer:
			closed = true
		case isFence(t):
			if k+1 >= len(lines) || strings.TrimSpace(lines[k+1].text) != closer {
				return Document{}, false, nil
			}
			closed = true
		case strings.Contains(t, closer):
			return Document{}, false, nil // comment ends before the block does
		}
		if !closed {
			yamlLines = append(yamlLines, t)
			continue
		}

		fm := map[string]any{}
		if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), &fm); err != nil {
			return Document{}, false, err
		}
		return Document{
			FrontMatter: fm,
			Head:        s[:lines[fence].start],
			LinePrefix:  prefix,
			Body:        s[fenceAt+3:],
			LineEndings: endingOf(s, lines[fence]),
		}, true, nil
	}
	return Document{}, false, nil
}

// emptyDocument describes a file without a block, positioned so that Render
// inserts a new one after the preamble (shebang and magic comments).
func emptyDocument(s string, lines []line, lang language) Document {
	eol := LF
	if len(lines) > 0 {
		eol = endingOf(s, lines[0])
	}
	cut := 0
	if !lang.markdown {
		for i, l := range lines {
			if (i == 0 && (strings.HasPrefix(l.text, "#!") || strings.HasPrefix(l.text, "<?php"))) ||
				magicComment.MatchString(l.text) {
				cut = l.end
				continue
			}
			break
		}
	}
	head, rest := s[:cut], s[cut:]
	if head != "" && !strings.HasSuffix(head, "\n") {
		head += string(eol)
	}
	doc := Document{FrontMatter: map[string]any{}, LineEndings: eol}
	switch {
	case lang.markdown:
		doc.Head, doc.Body = head, string(eol)+rest
	case len(lang.line) > 0:
		doc.Head, doc.Body = head, string(eol)+rest
		doc.LinePrefix = lang.line[0] + " "
	default:
		b := lang.block[0]
		doc.Head, doc.Body = head+b.open, b.close+string(eol)+rest
	}
	return doc
}

// Render is the inverse of Parse: it writes doc.FrontMatter as YAML between
// fences, each line carrying doc.LinePrefix, and wraps it in Head and Body.
func Render(doc Document) ([]byte, error) {
	eol := string(doc.LineEndings)
	if eol == "" {
		eol = string(LF)
	}
	p := doc.LinePrefix

	var buf bytes.Buffer
	buf.WriteString(doc.Head)
	buf.WriteString(p)
	buf.WriteString("---")
	buf.WriteString(eol)
	if len(doc.FrontMatter) > 0 {
		var y bytes.Buffer
		enc := yaml.NewEncoder(&y)
		enc.SetIndent(2)
		if err := enc.Encode(marshalFields(doc.FrontMatter)); err != nil {
			return nil, err
		}
		if err := enc.Close(); err != nil {
			return nil, err
		}
		for _, l := range strings.Split(strings.TrimSuffix(y.String(), "\n"), "\n") {
			if l == "" {
				buf.WriteString(strings.TrimRight(p, " "))
			} else {
				buf.WriteString(p)
				buf.WriteString(l)
			}
			buf.WriteString(eol)
		}
	}
	buf.WriteString(p)
	buf.WriteString("---")
	buf.WriteString(doc.Body)
	return buf.Bytes(), nil
}
