package frontmatter

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/nofuss-io/frontmatter/internal"
)

type (
	FrontMatter = internal.FrontMatter
	Document    = internal.Document

	Program     = internal.Program
	Query       = internal.Query
	SelectQuery = internal.SelectQuery
	UpdateQuery = internal.UpdateQuery
	AlterQuery  = internal.AlterQuery
)

var (
	ParseProgram = internal.ParseProgram
	ParseQuery   = internal.ParseQuery
	PrintTable   = internal.PrintTable
)

func ReadDocument(path string) (*Document, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	doc := &Document{FrontMatter: make(FrontMatter)}
	s := string(raw)

	if !strings.HasPrefix(s, "---\n") {
		doc.Body = s
		return doc, nil
	}
	rest := s[4:]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		if after, found := strings.CutSuffix(rest, "\n---"); found {
			if err := yaml.Unmarshal([]byte(after), &doc.FrontMatter); err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			return doc, nil
		}
		doc.Body = s
		return doc, nil
	}
	if err := yaml.Unmarshal([]byte(rest[:end]), &doc.FrontMatter); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	doc.Body = rest[end+5:]
	return doc, nil
}

func Write(path internal.FilePath, d *Document) error {
	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(d.FrontMatter); err != nil {
		return err
	}
	buf.WriteString("---\n")
	buf.WriteString(d.Body)
	return os.WriteFile(path, buf.Bytes(), 0644)
}
