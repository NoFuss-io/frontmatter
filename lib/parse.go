package lib

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
)

var errNotImplemented = errors.New("not implemented")

// ── Low-level helpers ─────────────────────────────────────────────────────────

func peekRune(r *bufio.Reader) (rune, error) {
	ch, _, err := r.ReadRune()
	if err == nil {
		_ = r.UnreadRune()
	}
	return ch, err
}

func skipWS(r *bufio.Reader) {
	for {
		ch, err := peekRune(r)
		if err != nil || !unicode.IsSpace(ch) {
			return
		}
		_, _, _ = r.ReadRune()
	}
}

// readIdent reads an unquoted identifier (letter or _ start, then letter/digit/_).
func readIdent(r *bufio.Reader) (string, error) {
	ch, err := peekRune(r)
	if err != nil {
		return "", fmt.Errorf("expected identifier: %w", err)
	}
	if !unicode.IsLetter(ch) && ch != '_' {
		return "", fmt.Errorf("identifier must start with letter or _, got %q", ch)
	}
	var b strings.Builder
	for {
		ch, err = peekRune(r)
		if err != nil {
			break
		}
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' {
			break
		}
		_, _, _ = r.ReadRune()
		b.WriteRune(ch)
	}
	return b.String(), nil
}

// readQuotedIdent reads a backtick-delimited identifier.
func readQuotedIdent(r *bufio.Reader) (string, error) {
	ch, _, err := r.ReadRune()
	if err != nil || ch != '`' {
		return "", fmt.Errorf("expected backtick")
	}
	var b strings.Builder
	for {
		ch, _, err = r.ReadRune()
		if err != nil {
			return "", fmt.Errorf("unterminated quoted identifier")
		}
		if ch == '`' {
			break
		}
		b.WriteRune(ch)
	}
	name := b.String()
	if name == "" {
		return "", fmt.Errorf("quoted identifier cannot be empty")
	}
	return name, nil
}

// readFieldName reads a quoted or unquoted identifier name.
func readFieldName(r *bufio.Reader) (string, error) {
	ch, err := peekRune(r)
	if err != nil {
		return "", fmt.Errorf("expected field name: %w", err)
	}
	if ch == '`' {
		return readQuotedIdent(r)
	}
	return readIdent(r)
}

var typeNames = map[string]FieldType{
	"string":   TypeString,
	"bool":     TypeBool,
	"int":      TypeInt,
	"numeric":  TypeNumber,
	"date":     TypeDate,
	"datetime": TypeDatetime,
	"link":     TypeLink,
	"mdlink":   TypeMdLink,
	"list":     TypeList,
	"any":      TypeAny,
}

// ── Field ─────────────────────────────────────────────────────────────────────

func (f *Field) Parse(r io.Reader) error {
	return f.parse(bufio.NewReader(r))
}

func (f *Field) parse(r *bufio.Reader) error {
	skipWS(r)

	name, err := readFieldName(r)
	if err != nil {
		return err
	}
	f.Name = name
	f.Type = TypeAny

	ch, err := peekRune(r)
	if err != nil || ch != ':' {
		return nil
	}
	_, _, _ = r.ReadRune() // consume ':'

	typeName, err := readIdent(r)
	if err != nil {
		return fmt.Errorf("expected type name after ':': %w", err)
	}
	ft, ok := typeNames[typeName]
	if !ok {
		return fmt.Errorf("unknown type %q", typeName)
	}
	f.Type = ft

	if ft != TypeList {
		return nil
	}

	ch, err = peekRune(r)
	if err != nil || ch != ':' {
		return nil
	}
	_, _, _ = r.ReadRune() // consume ':'

	elemName, err := readIdent(r)
	if err != nil {
		return fmt.Errorf("expected element type after 'list:': %w", err)
	}
	et, ok := typeNames[elemName]
	if !ok {
		return fmt.Errorf("unknown list element type %q", elemName)
	}
	f.ElemType = &et
	return nil
}

// ── RenamePair ────────────────────────────────────────────────────────────────

func (p *RenamePair) Parse(r io.Reader) error {
	return p.parse(bufio.NewReader(r))
}

func (p *RenamePair) parse(r *bufio.Reader) error {
	skipWS(r)

	from, err := readFieldName(r)
	if err != nil {
		return fmt.Errorf("expected source field: %w", err)
	}

	skipWS(r)
	kw, err := readIdent(r)
	if err != nil {
		return fmt.Errorf("expected 'to': %w", err)
	}
	if strings.ToLower(kw) != "to" {
		return fmt.Errorf("expected 'to', got %q", kw)
	}

	skipWS(r)
	to, err := readFieldName(r)
	if err != nil {
		return fmt.Errorf("expected target field: %w", err)
	}

	p.From = from
	p.To = to
	return nil
}

// ── Queries ───────────────────────────────────────────────────────────────────

// ParseQuery reads a full query from r and returns a SelectQuery, UpdateQuery, or AlterQuery.
func ParseQuery(r io.Reader) (Query, error) {
	return nil, errNotImplemented
}

func (q *SelectQuery) Parse(r io.Reader) error {
	return errNotImplemented
}

func (q *UpdateQuery) Parse(r io.Reader) error {
	return errNotImplemented
}

func (q *AlterQuery) Parse(r io.Reader) error {
	return errNotImplemented
}

// ── Expressions ───────────────────────────────────────────────────────────────

// ParseExpr reads an expression from r and returns the appropriate Expr node.
func ParseExpr(r io.Reader) (Expr, error) {
	return nil, errNotImplemented
}

func (e *BinExpr) Parse(r io.Reader) error {
	return errNotImplemented
}

func (e *UnaryExpr) Parse(r io.Reader) error {
	return errNotImplemented
}

func (e *FieldExpr) Parse(r io.Reader) error {
	return errNotImplemented
}

func (e *LitExpr) Parse(r io.Reader) error {
	return errNotImplemented
}

func (a *Assign) Parse(r io.Reader) error {
	return errNotImplemented
}

func (s *SortTerm) Parse(r io.Reader) error {
	return errNotImplemented
}
