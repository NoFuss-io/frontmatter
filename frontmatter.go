package frontmatter

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/nofuss-io/frontmatter/internal"
	"gopkg.in/yaml.v3"
)

type (
	FrontMatter = internal.FrontMatter
	Document    = internal.Document

	SelectQuery = internal.SelectQuery
	UpdateQuery = internal.UpdateQuery
	AlterQuery  = internal.AlterQuery

	Row      = internal.Row
	SortTerm = internal.SortTerm
	Expr     = internal.Expr
)

func ExpandGlobs(globs []string) ([]string, error) {
	return internal.ExpandGlobs(globs)
}

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

func Write(d *Document, path string) error {
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

func FieldName(e Expr, idx int) string {
	return internal.FieldName(e, idx)
}

func PrintTable(w io.Writer, headers []string, paths []string, rows []Row) {
	internal.PrintTable(w, headers, paths, rows)
}

// SortRows reorders paths/rows in place according to terms. Null values sort
// last. Default direction asc unless term.Desc.
func SortRows(paths []string, rows []Row, terms []SortTerm, fms []FrontMatter) {
	type item struct {
		path string
		row  Row
		fm   FrontMatter
	}
	items := make([]item, len(paths))
	for i := range paths {
		items[i] = item{paths[i], rows[i], fms[i]}
	}
	sort.SliceStable(items, func(i, j int) bool {
		for _, t := range terms {
			vi := t.Eval(items[i].fm)
			vj := t.Eval(items[j].fm)
			c := compareValues(vi, vj)
			if c == 0 {
				continue
			}
			if t.Desc {
				return c > 0
			}
			return c < 0
		}
		return false
	})
	for i := range items {
		paths[i] = items[i].path
		rows[i] = items[i].row
		fms[i] = items[i].fm
	}
}

// Limit truncates paths and rows to n. n <= 0 means no limit.
func Limit(n int, paths []string, rows []Row) ([]string, []Row) {
	if n > 0 && len(rows) > n {
		return paths[:n], rows[:n]
	}
	return paths, rows
}

// compareValues returns -1, 0, +1. Null sorts after non-null. Numeric values
// compare numerically; everything else falls back to string form.
func compareValues(a, b internal.Value) int {
	if a.Null && b.Null {
		return 0
	}
	if a.Null {
		return 1
	}
	if b.Null {
		return -1
	}
	if isNumeric(a.Kind) && isNumeric(b.Kind) {
		af, _ := internal.Cast(a, internal.TypeNumber)
		bf, _ := internal.Cast(b, internal.TypeNumber)
		if af.Null || bf.Null {
			return strings.Compare(internal.FormatValue(a), internal.FormatValue(b))
		}
		x, y := af.Data.(float64), bf.Data.(float64)
		switch {
		case x < y:
			return -1
		case x > y:
			return 1
		}
		return 0
	}
	if (a.Kind == internal.TypeDate || a.Kind == internal.TypeDatetime) &&
		(b.Kind == internal.TypeDate || b.Kind == internal.TypeDatetime) {
		ta := a.Data.(time.Time)
		tb := b.Data.(time.Time)
		switch {
		case ta.Before(tb):
			return -1
		case ta.After(tb):
			return 1
		}
		return 0
	}
	return strings.Compare(internal.FormatValue(a), internal.FormatValue(b))
}

func isNumeric(k internal.FieldType) bool {
	return k == internal.TypeInt || k == internal.TypeNumber || k == internal.TypeBool
}
