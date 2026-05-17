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

// ── LitExpr ───────────────────────────────────────────────────────────────────

func (e *LitExpr) Parse(r io.Reader) error {
	return e.parse(bufio.NewReader(r))
}

func (e *LitExpr) parse(r *bufio.Reader) error {
	skipWS(r)

	prefix, _ := r.Peek(2)
	if len(prefix) == 0 {
		return fmt.Errorf("expected literal: EOF")
	}
	b0 := prefix[0]

	switch {
	case b0 == '"' || b0 == '\'':
		return e.parseString(r, false)
	case (b0 == 'r' || b0 == 'R') && len(prefix) >= 2 && (prefix[1] == '"' || prefix[1] == '\''):
		_, _, _ = r.ReadRune() // consume 'r'/'R'
		return e.parseString(r, true)
	case b0 == '-' || b0 == '.' || (b0 >= '0' && b0 <= '9'):
		return e.parseNumber(r)
	default:
		return e.parseKeyword(r)
	}
}

func (e *LitExpr) parseKeyword(r *bufio.Reader) error {
	word, err := readIdent(r)
	if err != nil {
		return fmt.Errorf("expected literal: %w", err)
	}
	switch strings.ToLower(word) {
	case "true", "false":
		e.Kind = LitBool
		e.Value = word
	case "null":
		e.Kind = LitNull
		e.Value = word
	default:
		return fmt.Errorf("unexpected identifier %q in literal context", word)
	}
	return nil
}

func (e *LitExpr) parseNumber(r *bufio.Reader) error {
	var b strings.Builder
	isFloat := false

	// Optional leading minus
	ch, _ := peekRune(r)
	if ch == '-' {
		_, _, _ = r.ReadRune()
		b.WriteRune('-')
	}

	// Hex prefix
	ch, err := peekRune(r)
	if err != nil {
		return fmt.Errorf("expected digit")
	}
	if ch == '0' {
		_, _, _ = r.ReadRune()
		b.WriteRune('0')
		if next, err := peekRune(r); err == nil && (next == 'x' || next == 'X') {
			_, _, _ = r.ReadRune()
			b.WriteRune(next)
			for {
				ch, err = peekRune(r)
				if err != nil || !isHexDigit(ch) {
					break
				}
				_, _, _ = r.ReadRune()
				b.WriteRune(ch)
			}
			e.Kind = LitInt
			e.Value = b.String()
			return nil
		}
	}

	// Integer digits
	for {
		ch, err = peekRune(r)
		if err != nil || ch < '0' || ch > '9' {
			break
		}
		_, _, _ = r.ReadRune()
		b.WriteRune(ch)
	}

	// Optional fractional part
	if ch, _ := peekRune(r); ch == '.' {
		_, _, _ = r.ReadRune()
		b.WriteRune('.')
		isFloat = true
		for {
			ch, err = peekRune(r)
			if err != nil || ch < '0' || ch > '9' {
				break
			}
			_, _, _ = r.ReadRune()
			b.WriteRune(ch)
		}
	}

	// Optional exponent
	if ch, _ := peekRune(r); ch == 'e' || ch == 'E' {
		_, _, _ = r.ReadRune()
		b.WriteRune(ch)
		isFloat = true
		if ch, _ := peekRune(r); ch == '+' || ch == '-' {
			_, _, _ = r.ReadRune()
			b.WriteRune(ch)
		}
		for {
			ch, err = peekRune(r)
			if err != nil || ch < '0' || ch > '9' {
				break
			}
			_, _, _ = r.ReadRune()
			b.WriteRune(ch)
		}
	}

	if isFloat {
		e.Kind = LitNumeric
	} else {
		e.Kind = LitInt
	}
	e.Value = b.String()
	return nil
}

func isHexDigit(ch rune) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func (e *LitExpr) parseString(r *bufio.Reader, raw bool) error {
	ch, _, err := r.ReadRune()
	if err != nil {
		return fmt.Errorf("expected string: %w", err)
	}
	quote := ch

	// Triple-quote check
	triple := false
	if p, _ := r.Peek(2); len(p) >= 2 && p[0] == byte(quote) && p[1] == byte(quote) {
		_, _, _ = r.ReadRune()
		_, _, _ = r.ReadRune()
		triple = true
	}

	var b strings.Builder
	for {
		if triple {
			if p, _ := r.Peek(3); len(p) >= 3 && p[0] == byte(quote) && p[1] == byte(quote) && p[2] == byte(quote) {
				_, _, _ = r.ReadRune()
				_, _, _ = r.ReadRune()
				_, _, _ = r.ReadRune()
				break
			}
		}
		ch, _, err = r.ReadRune()
		if err != nil {
			return fmt.Errorf("unterminated string literal")
		}
		if !triple && ch == quote {
			break
		}
		if raw || ch != '\\' {
			b.WriteRune(ch)
			continue
		}
		// Escape sequence
		esc, _, err := r.ReadRune()
		if err != nil {
			return fmt.Errorf("unterminated escape sequence")
		}
		switch esc {
		case 'a':
			b.WriteRune('\a')
		case 'b':
			b.WriteRune('\b')
		case 'f':
			b.WriteRune('\f')
		case 'n':
			b.WriteRune('\n')
		case 'r':
			b.WriteRune('\r')
		case 't':
			b.WriteRune('\t')
		case 'v':
			b.WriteRune('\v')
		case '\\':
			b.WriteRune('\\')
		case '\'':
			b.WriteRune('\'')
		case '"':
			b.WriteRune('"')
		case '`':
			b.WriteRune('`')
		case '?':
			b.WriteRune('?')
		case '0', '1', '2', '3': // octal \NNN
			d1, _, e1 := r.ReadRune()
			d2, _, e2 := r.ReadRune()
			if e1 != nil || e2 != nil {
				return fmt.Errorf("incomplete octal escape")
			}
			b.WriteRune(rune(int(esc-'0')*64 + int(d1-'0')*8 + int(d2-'0')))
		case 'x':
			v, err := readHexChars(r, 2)
			if err != nil {
				return err
			}
			b.WriteRune(rune(v))
		case 'u':
			v, err := readHexChars(r, 4)
			if err != nil {
				return err
			}
			b.WriteRune(rune(v))
		case 'U':
			v, err := readHexChars(r, 8)
			if err != nil {
				return err
			}
			b.WriteRune(rune(v))
		default:
			return fmt.Errorf("unknown escape sequence \\%c", esc)
		}
	}
	e.Kind = LitString
	e.Value = b.String()
	return nil
}

func readHexChars(r *bufio.Reader, n int) (int64, error) {
	var val int64
	for i := 0; i < n; i++ {
		ch, _, err := r.ReadRune()
		if err != nil {
			return 0, fmt.Errorf("incomplete hex escape")
		}
		var d int64
		switch {
		case ch >= '0' && ch <= '9':
			d = int64(ch - '0')
		case ch >= 'a' && ch <= 'f':
			d = int64(ch-'a') + 10
		case ch >= 'A' && ch <= 'F':
			d = int64(ch-'A') + 10
		default:
			return 0, fmt.Errorf("invalid hex digit %q", ch)
		}
		val = val*16 + d
	}
	return val, nil
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

func (a *Assign) Parse(r io.Reader) error {
	return errNotImplemented
}

func (s *SortTerm) Parse(r io.Reader) error {
	return errNotImplemented
}
