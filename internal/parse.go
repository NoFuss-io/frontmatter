package internal

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var errNotImplemented = errors.New("not implemented")

// ── Cursor ────────────────────────────────────────────────────────────────────

type cursor struct {
	src []byte
	pos int
}

func newCursor(b []byte) *cursor { return &cursor{src: b} }

func (c *cursor) peekN(n int) []byte {
	end := c.pos + n
	if end > len(c.src) {
		end = len(c.src)
	}
	if c.pos >= len(c.src) {
		return c.src[len(c.src):len(c.src)]
	}
	return c.src[c.pos:end]
}

func (c *cursor) peekRune() (rune, bool) {
	if c.pos >= len(c.src) {
		return 0, false
	}
	r, _ := utf8.DecodeRune(c.src[c.pos:])
	return r, true
}

func (c *cursor) readRune() (rune, bool) {
	if c.pos >= len(c.src) {
		return 0, false
	}
	r, n := utf8.DecodeRune(c.src[c.pos:])
	c.pos += n
	return r, true
}

func (c *cursor) advance(n int) {
	c.pos += n
	if c.pos > len(c.src) {
		c.pos = len(c.src)
	}
}

func readAllCursor(r io.Reader) (*cursor, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return newCursor(b), nil
}

// ── Low-level helpers ─────────────────────────────────────────────────────────

// skipWS consumes whitespace and '--' line comments. A comment runs from '--'
// to the next newline (or EOF) and is treated as a single whitespace separator.
func skipWS(c *cursor) {
	for {
		b := c.peekN(2)
		if len(b) >= 2 && b[0] == '-' && b[1] == '-' {
			for {
				ch, ok := c.readRune()
				if !ok || ch == '\n' {
					break
				}
			}
			continue
		}
		ch, ok := c.peekRune()
		if !ok || !unicode.IsSpace(ch) {
			return
		}
		c.readRune()
	}
}

// readIdent reads an unquoted identifier (letter or _ start, then letter/digit/_).
func readIdent(c *cursor) (string, error) {
	ch, ok := c.peekRune()
	if !ok {
		return "", fmt.Errorf("expected identifier: %w", io.EOF)
	}
	if !unicode.IsLetter(ch) && ch != '_' {
		return "", fmt.Errorf("identifier must start with letter or _, got %q", ch)
	}
	var b strings.Builder
	for {
		ch, ok = c.peekRune()
		if !ok {
			break
		}
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' {
			break
		}
		c.readRune()
		b.WriteRune(ch)
	}
	return b.String(), nil
}

// readQuotedIdent reads a backtick-delimited identifier.
func readQuotedIdent(c *cursor) (string, error) {
	ch, ok := c.readRune()
	if !ok || ch != '`' {
		return "", fmt.Errorf("expected backtick")
	}
	var b strings.Builder
	for {
		ch, ok = c.readRune()
		if !ok {
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
func readFieldName(c *cursor) (string, error) {
	ch, ok := c.peekRune()
	if !ok {
		return "", fmt.Errorf("expected field name: %w", io.EOF)
	}
	if ch == '`' {
		return readQuotedIdent(c)
	}
	return readIdent(c)
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
	c, err := readAllCursor(r)
	if err != nil {
		return err
	}
	return f.parse(c)
}

func (f *Field) parse(c *cursor) error {
	skipWS(c)

	name, err := readFieldName(c)
	if err != nil {
		return err
	}
	f.Name = name
	f.Type = TypeAny

	ch, ok := c.peekRune()
	if !ok || ch != ':' {
		return nil
	}
	c.readRune() // consume ':'

	typeName, err := readIdent(c)
	if err != nil {
		return fmt.Errorf("expected type name after ':': %w", err)
	}
	ft, ok := typeNames[typeName]
	if !ok {
		return fmt.Errorf("unknown type %q", typeName)
	}
	f.Type = ft

	if ft == TypeList {
		if ch, ok := c.peekRune(); ok && ch == ':' {
			return fmt.Errorf("list element type annotation is no longer supported; lists are list-of-string")
		}
	}
	return nil
}

// ── RenamePair ────────────────────────────────────────────────────────────────

func (p *RenamePair) Parse(r io.Reader) error {
	c, err := readAllCursor(r)
	if err != nil {
		return err
	}
	return p.parse(c)
}

func (p *RenamePair) parse(c *cursor) error {
	skipWS(c)

	from, err := readFieldName(c)
	if err != nil {
		return fmt.Errorf("expected source field: %w", err)
	}

	skipWS(c)
	kw, err := readIdent(c)
	if err != nil {
		return fmt.Errorf("expected 'to': %w", err)
	}
	if strings.ToLower(kw) != "to" {
		return fmt.Errorf("expected 'to', got %q", kw)
	}

	skipWS(c)
	to, err := readFieldName(c)
	if err != nil {
		return fmt.Errorf("expected target field: %w", err)
	}

	p.From = from
	p.To = to
	return nil
}

// ── Clause helpers ────────────────────────────────────────────────────────────

// expectKeyword consumes a specific keyword (case-insensitive) and errors otherwise.
func expectKeyword(c *cursor, kw string) error {
	skipWS(c)
	w := peekKeyword(c)
	if strings.ToLower(w) != kw {
		return fmt.Errorf("expected %q, got %q", kw, w)
	}
	consumeBytes(c, len(w))
	return nil
}

// atStopKeyword returns true if the next token is one of stopKws (case-insensitive)
// and is followed by whitespace or EOF (so it's not a prefix of a longer ident).
func atStopKeyword(c *cursor, stopKws []string) bool {
	b := c.peekN(32)
	if len(b) == 0 || !isIdentStart(b[0]) {
		return false
	}
	j := 1
	for j < len(b) && isIdentCont(b[j]) {
		j++
	}
	word := strings.ToLower(string(b[:j]))
	for _, kw := range stopKws {
		if word == kw {
			return j == len(b) || unicode.IsSpace(rune(b[j]))
		}
	}
	return false
}

// readGlobs reads whitespace-separated glob tokens until EOF, a stop keyword,
// or a ';' statement terminator. Tokens may be double- or single-quoted to
// include spaces (e.g. paths from shell expansion).
func readGlobs(c *cursor, stopKws ...string) []string {
	var out []string
	for {
		skipWS(c)
		b := c.peekN(1)
		if len(b) == 0 || b[0] == ';' {
			break
		}
		if atStopKeyword(c, stopKws) {
			break
		}
		var sb strings.Builder
		if b[0] == '"' || b[0] == '\'' {
			quote := rune(b[0])
			c.readRune()
			for {
				ch, ok := c.readRune()
				if !ok || ch == quote {
					break
				}
				if ch == '\\' {
					if next, ok := c.readRune(); ok {
						sb.WriteRune(next)
					}
				} else {
					sb.WriteRune(ch)
				}
			}
		} else {
			for {
				b := c.peekN(1)
				if len(b) == 0 || unicode.IsSpace(rune(b[0])) || b[0] == ';' {
					break
				}
				ch, _ := c.readRune()
				sb.WriteRune(ch)
			}
		}
		if sb.Len() > 0 {
			out = append(out, sb.String())
		}
	}
	return out
}

// readFieldList reads a comma-separated list of Field values, stopping at EOF or a stop keyword.
func readFieldList(c *cursor, stopKws ...string) ([]Field, error) {
	var out []Field
	for {
		skipWS(c)
		b := c.peekN(1)
		if len(b) == 0 || b[0] == ';' || atStopKeyword(c, stopKws) {
			break
		}
		var f Field
		if err := f.parse(c); err != nil {
			return nil, err
		}
		out = append(out, f)
		skipWS(c)
		b = c.peekN(1)
		if len(b) == 0 || b[0] != ',' {
			break
		}
		consumeBytes(c, 1)
	}
	return out, nil
}

// readExprList reads a comma-separated list of expressions, stopping at EOF or a stop keyword.
func readExprList(c *cursor, stopKws ...string) ([]Expr, error) {
	var out []Expr
	for {
		skipWS(c)
		b := c.peekN(1)
		if len(b) == 0 || b[0] == ';' || atStopKeyword(c, stopKws) {
			break
		}
		e, err := parseOrExpr(c)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
		skipWS(c)
		b = c.peekN(1)
		if len(b) == 0 || b[0] != ',' {
			break
		}
		consumeBytes(c, 1)
	}
	return out, nil
}

// readSortTermList reads a comma-separated list of SortTerm values, stopping at EOF or a stop keyword.
func readSortTermList(c *cursor, stopKws ...string) ([]SortTerm, error) {
	var out []SortTerm
	for {
		skipWS(c)
		b := c.peekN(1)
		if len(b) == 0 || b[0] == ';' || atStopKeyword(c, stopKws) {
			break
		}
		var t SortTerm
		if err := t.parse(c); err != nil {
			return nil, err
		}
		out = append(out, t)
		skipWS(c)
		b = c.peekN(1)
		if len(b) == 0 || b[0] != ',' {
			break
		}
		consumeBytes(c, 1)
	}
	return out, nil
}

// readIntLit reads an optional-sign decimal integer.
func readIntLit(c *cursor) (int, error) {
	skipWS(c)
	var sb strings.Builder
	if b := c.peekN(1); len(b) > 0 && b[0] == '-' {
		consumeBytes(c, 1)
		sb.WriteByte('-')
	}
	for {
		b := c.peekN(1)
		if len(b) == 0 || b[0] < '0' || b[0] > '9' {
			break
		}
		consumeBytes(c, 1)
		sb.WriteByte(b[0])
	}
	s := sb.String()
	if s == "" || s == "-" {
		return 0, fmt.Errorf("expected integer")
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q: %w", s, err)
	}
	return n, nil
}

// readAssignList reads a comma-separated list of Assign values, stopping at EOF or a stop keyword.
func readAssignList(c *cursor, stopKws ...string) ([]Assign, error) {
	var out []Assign
	for {
		skipWS(c)
		b := c.peekN(1)
		if len(b) == 0 || b[0] == ';' || atStopKeyword(c, stopKws) {
			break
		}
		var a Assign
		if err := a.parse(c); err != nil {
			return nil, err
		}
		out = append(out, a)
		skipWS(c)
		b = c.peekN(1)
		if len(b) == 0 || b[0] != ',' {
			break
		}
		consumeBytes(c, 1)
	}
	return out, nil
}

// readRenamePairs reads a comma-separated list of RenamePair values, stopping at EOF or a stop keyword.
func readRenamePairs(c *cursor, stopKws ...string) ([]RenamePair, error) {
	var out []RenamePair
	for {
		skipWS(c)
		b := c.peekN(1)
		if len(b) == 0 || b[0] == ';' || atStopKeyword(c, stopKws) {
			break
		}
		var p RenamePair
		if err := p.parse(c); err != nil {
			return nil, err
		}
		out = append(out, p)
		skipWS(c)
		b = c.peekN(1)
		if len(b) == 0 || b[0] != ',' {
			break
		}
		consumeBytes(c, 1)
	}
	return out, nil
}

// parseOptionalWhere consumes an optional "where <expr>" clause.
func parseOptionalWhere(c *cursor, dst *Expr) error {
	skipWS(c)
	if strings.ToLower(peekKeyword(c)) != "where" {
		return nil
	}
	consumeBytes(c, 5)
	e, err := parseOrExpr(c)
	if err != nil {
		return err
	}
	*dst = e
	return nil
}

// ── Queries ───────────────────────────────────────────────────────────────────

// ParseQuery reads a single query from r and returns a SelectQuery,
// UpdateQuery, or AlterQuery. Trailing input is not validated.
func ParseQuery(r io.Reader) (Query, error) {
	c, err := readAllCursor(r)
	if err != nil {
		return nil, err
	}
	return parseOneQuery(c)
}

// ParseProgram reads a sequence of statements from r and returns a Program.
// Statements are separated by ';'. '--' starts a line comment that runs to
// the next newline and is treated as whitespace. Consecutive ';' separators
// (including leading and trailing ones) are tolerated. An input that contains
// only whitespace and comments yields an empty Program.
func ParseProgram(r io.Reader) (Program, error) {
	c, err := readAllCursor(r)
	if err != nil {
		return Program{}, err
	}
	var p Program
	for {
		skipSeparators(c)
		b := c.peekN(1)
		if len(b) == 0 {
			return p, nil
		}
		n := len(p.Stmts) + 1
		q, err := parseOneQuery(c)
		if err != nil {
			if n > 1 {
				return p, fmt.Errorf("statement %d: %w", n, err)
			}
			return p, err
		}
		p.Stmts = append(p.Stmts, q)
		skipWS(c)
		b = c.peekN(1)
		if len(b) == 0 {
			return p, nil
		}
		if b[0] != ';' {
			return p, fmt.Errorf("statement %d: expected ';' or end of input, got %q", n, string(b[0]))
		}
		consumeBytes(c, 1)
	}
}

// skipSeparators consumes whitespace, comments, and any number of ';' tokens
// so that consecutive or leading separators are tolerated between statements.
func skipSeparators(c *cursor) {
	for {
		skipWS(c)
		b := c.peekN(1)
		if len(b) == 0 || b[0] != ';' {
			return
		}
		consumeBytes(c, 1)
	}
}

func parseOneQuery(c *cursor) (Query, error) {
	skipWS(c)
	kw := peekKeyword(c)
	switch strings.ToLower(kw) {
	case "select":
		var q SelectQuery
		if err := q.parse(c); err != nil {
			return nil, err
		}
		return q, nil
	case "update":
		var q UpdateQuery
		if err := q.parse(c); err != nil {
			return nil, err
		}
		return q, nil
	case "alter":
		var q AlterQuery
		if err := q.parse(c); err != nil {
			return nil, err
		}
		return q, nil
	}
	if kw == "" {
		return nil, fmt.Errorf("expected query keyword (select|update|alter)")
	}
	return nil, fmt.Errorf("unknown query keyword %q", kw)
}

func (q *SelectQuery) Parse(r io.Reader) error {
	c, err := readAllCursor(r)
	if err != nil {
		return err
	}
	return q.parse(c)
}

func (q *SelectQuery) parse(c *cursor) error {
	if err := expectKeyword(c, "select"); err != nil {
		return err
	}

	fields, err := readExprList(c, "from")
	if err != nil {
		return err
	}
	q.Select = fields

	if err := expectKeyword(c, "from"); err != nil {
		return err
	}

	globs := readGlobs(c, "where", "sort", "limit")
	if len(globs) == 0 {
		return fmt.Errorf("expected glob after 'from'")
	}
	q.From, err = ExpandGlobs(globs)
	if err != nil {
		return fmt.Errorf("could not expand globs: %w", err)
	}

	if err := parseOptionalWhere(c, &q.Where); err != nil {
		return err
	}

	skipWS(c)
	if strings.ToLower(peekKeyword(c)) == "sort" {
		consumeBytes(c, 4)
		if err := expectKeyword(c, "by"); err != nil {
			return err
		}
		terms, err := readSortTermList(c, "limit")
		if err != nil {
			return err
		}
		if len(terms) == 0 {
			return fmt.Errorf("expected sort term after 'sort by'")
		}
		q.SortBy = terms
	}

	skipWS(c)
	if strings.ToLower(peekKeyword(c)) == "limit" {
		consumeBytes(c, 5)
		n, err := readIntLit(c)
		if err != nil {
			return fmt.Errorf("limit: %w", err)
		}
		if n < 0 {
			return fmt.Errorf("limit must be non-negative, got %d", n)
		}
		q.Limit = n
	}

	return nil
}

func (q *UpdateQuery) Parse(r io.Reader) error {
	c, err := readAllCursor(r)
	if err != nil {
		return err
	}
	return q.parse(c)
}

func (q *UpdateQuery) parse(c *cursor) error {
	if err := expectKeyword(c, "update"); err != nil {
		return err
	}

	globs := readGlobs(c, "set")
	if len(globs) == 0 {
		return fmt.Errorf("expected glob after 'update'")
	}
	q.From = globs

	if err := expectKeyword(c, "set"); err != nil {
		return err
	}

	assigns, err := readAssignList(c, "where")
	if err != nil {
		return err
	}
	if len(assigns) == 0 {
		return fmt.Errorf("expected assignment after 'set'")
	}
	q.Set = assigns
	q.Select = assignsToFields(assigns)

	return parseOptionalWhere(c, &q.Where)
}

func assignsToFields(assigns []Assign) []Expr {
	out := make([]Expr, 0, len(assigns))
	for _, a := range assigns {
		out = append(out, FieldExpr{a.Field})
	}
	return out
}

func (q *AlterQuery) Parse(r io.Reader) error {
	c, err := readAllCursor(r)
	if err != nil {
		return err
	}
	return q.parse(c)
}

func (q *AlterQuery) parse(c *cursor) error {
	if err := expectKeyword(c, "alter"); err != nil {
		return err
	}

	globs := readGlobs(c, "drop", "rename")
	if len(globs) == 0 {
		return fmt.Errorf("expected glob after 'alter'")
	}
	q.From = globs

	skipWS(c)
	kw := peekKeyword(c)
	switch strings.ToLower(kw) {
	case "drop":
		consumeBytes(c, len(kw))
		q.Op = AlterDrop
		fields, err := readFieldList(c, "where")
		if err != nil {
			return err
		}
		if len(fields) == 0 {
			return fmt.Errorf("expected field after 'drop'")
		}
		q.Drop = fields
	case "rename":
		consumeBytes(c, len(kw))
		q.Op = AlterRename
		pairs, err := readRenamePairs(c, "where")
		if err != nil {
			return err
		}
		if len(pairs) == 0 {
			return fmt.Errorf("expected rename pair after 'rename'")
		}
		q.Rename = pairs
		q.Select = renamesToFields(pairs)
	default:
		return fmt.Errorf("expected 'drop' or 'rename' after globs")
	}

	return parseOptionalWhere(c, &q.Where)
}

func renamesToFields(pairs []RenamePair) []Expr {
	out := make([]Expr, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, FieldExpr{Field{Name: p.To, Type: TypeAny}})
	}
	return out
}

// ── LitExpr ───────────────────────────────────────────────────────────────────

func (e *LitExpr) Parse(r io.Reader) error {
	c, err := readAllCursor(r)
	if err != nil {
		return err
	}
	return e.parse(c)
}

func (e *LitExpr) parse(c *cursor) error {
	skipWS(c)

	prefix := c.peekN(2)
	if len(prefix) == 0 {
		return fmt.Errorf("expected literal: EOF")
	}
	b0 := prefix[0]

	switch {
	case b0 == '"' || b0 == '\'':
		return e.parseString(c, false)
	case (b0 == 'r' || b0 == 'R') && len(prefix) >= 2 && (prefix[1] == '"' || prefix[1] == '\''):
		c.readRune() // consume 'r'/'R'
		return e.parseString(c, true)
	case b0 == '-' || b0 == '.' || (b0 >= '0' && b0 <= '9'):
		return e.parseNumber(c)
	default:
		return e.parseKeyword(c)
	}
}

func (e *LitExpr) parseKeyword(c *cursor) error {
	word, err := readIdent(c)
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

func (e *LitExpr) parseNumber(c *cursor) error {
	var b strings.Builder
	isFloat := false

	// Optional leading minus
	ch, _ := c.peekRune()
	if ch == '-' {
		c.readRune()
		b.WriteRune('-')
	}

	// Hex prefix
	ch, ok := c.peekRune()
	if !ok {
		return fmt.Errorf("expected digit")
	}
	if ch == '0' {
		c.readRune()
		b.WriteRune('0')
		if next, ok := c.peekRune(); ok && (next == 'x' || next == 'X') {
			c.readRune()
			b.WriteRune(next)
			for {
				ch, ok = c.peekRune()
				if !ok || !isHexDigit(ch) {
					break
				}
				c.readRune()
				b.WriteRune(ch)
			}
			e.Kind = LitInt
			e.Value = b.String()
			return nil
		}
	}

	// Integer digits
	for {
		ch, ok = c.peekRune()
		if !ok || ch < '0' || ch > '9' {
			break
		}
		c.readRune()
		b.WriteRune(ch)
	}

	// Optional fractional part
	if ch, _ := c.peekRune(); ch == '.' {
		c.readRune()
		b.WriteRune('.')
		isFloat = true
		for {
			ch, ok = c.peekRune()
			if !ok || ch < '0' || ch > '9' {
				break
			}
			c.readRune()
			b.WriteRune(ch)
		}
	}

	// Optional exponent
	if ch, _ := c.peekRune(); ch == 'e' || ch == 'E' {
		c.readRune()
		b.WriteRune(ch)
		isFloat = true
		if ch, _ := c.peekRune(); ch == '+' || ch == '-' {
			c.readRune()
			b.WriteRune(ch)
		}
		for {
			ch, ok = c.peekRune()
			if !ok || ch < '0' || ch > '9' {
				break
			}
			c.readRune()
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

func (e *LitExpr) parseString(c *cursor, raw bool) error {
	ch, ok := c.readRune()
	if !ok {
		return fmt.Errorf("expected string: %w", io.EOF)
	}
	quote := ch

	// Triple-quote check
	triple := false
	if p := c.peekN(2); len(p) >= 2 && p[0] == byte(quote) && p[1] == byte(quote) {
		c.readRune()
		c.readRune()
		triple = true
	}

	var b strings.Builder
	for {
		if triple {
			if p := c.peekN(3); len(p) >= 3 && p[0] == byte(quote) && p[1] == byte(quote) && p[2] == byte(quote) {
				c.readRune()
				c.readRune()
				c.readRune()
				break
			}
		}
		ch, ok = c.readRune()
		if !ok {
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
		esc, ok := c.readRune()
		if !ok {
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
			d1, ok1 := c.readRune()
			d2, ok2 := c.readRune()
			if !ok1 || !ok2 {
				return fmt.Errorf("incomplete octal escape")
			}
			b.WriteRune(rune(int(esc-'0')*64 + int(d1-'0')*8 + int(d2-'0')))
		case 'x':
			v, err := readHexChars(c, 2)
			if err != nil {
				return err
			}
			b.WriteRune(rune(v))
		case 'u':
			v, err := readHexChars(c, 4)
			if err != nil {
				return err
			}
			b.WriteRune(rune(v))
		case 'U':
			v, err := readHexChars(c, 8)
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

func readHexChars(c *cursor, n int) (int64, error) {
	var val int64
	for i := 0; i < n; i++ {
		ch, ok := c.readRune()
		if !ok {
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
	c, err := readAllCursor(r)
	if err != nil {
		return nil, err
	}
	return parseOrExpr(c)
}

func parseOrExpr(c *cursor) (Expr, error) {
	left, err := parseAndExpr(c)
	if err != nil {
		return nil, err
	}
	for {
		skipWS(c)
		kw := peekKeyword(c)
		if strings.ToLower(kw) != "or" {
			return left, nil
		}
		consumeBytes(c, len(kw))
		right, err := parseAndExpr(c)
		if err != nil {
			return nil, err
		}
		left = BinExpr{Op: BinOr, Left: left, Right: right}
	}
}

func parseAndExpr(c *cursor) (Expr, error) {
	left, err := parseNotExpr(c)
	if err != nil {
		return nil, err
	}
	for {
		skipWS(c)
		kw := peekKeyword(c)
		if strings.ToLower(kw) != "and" {
			return left, nil
		}
		consumeBytes(c, len(kw))
		right, err := parseNotExpr(c)
		if err != nil {
			return nil, err
		}
		left = BinExpr{Op: BinAnd, Left: left, Right: right}
	}
}

func parseNotExpr(c *cursor) (Expr, error) {
	skipWS(c)
	kw := peekKeyword(c)
	if strings.ToLower(kw) == "not" {
		consumeBytes(c, len(kw))
		inner, err := parseComparison(c)
		if err != nil {
			return nil, err
		}
		return UnaryExpr{Op: UnaryNot, Operand: inner}, nil
	}
	return parseComparison(c)
}

func parseComparison(c *cursor) (Expr, error) {
	left, err := parseArith(c)
	if err != nil {
		return nil, err
	}
	skipWS(c)
	b := c.peekN(2)
	var op BinOp
	var n int
	switch {
	case len(b) >= 2 && b[0] == '!' && b[1] == '=':
		op, n = BinNe, 2
	case len(b) >= 2 && b[0] == '<' && b[1] == '=':
		op, n = BinLe, 2
	case len(b) >= 2 && b[0] == '>' && b[1] == '=':
		op, n = BinGe, 2
	case len(b) >= 1 && b[0] == '=':
		op, n = BinEq, 1
	case len(b) >= 1 && b[0] == '<':
		op, n = BinLt, 1
	case len(b) >= 1 && b[0] == '>':
		op, n = BinGt, 1
	default:
		return left, nil
	}
	consumeBytes(c, n)
	right, err := parseArith(c)
	if err != nil {
		return nil, err
	}
	return BinExpr{Op: op, Left: left, Right: right}, nil
}

func parseArith(c *cursor) (Expr, error) {
	left, err := parseTerm(c)
	if err != nil {
		return nil, err
	}
	for {
		skipWS(c)
		b := c.peekN(1)
		if len(b) == 0 || (b[0] != '+' && b[0] != '-') {
			return left, nil
		}
		op := BinAdd
		if b[0] == '-' {
			op = BinSub
		}
		consumeBytes(c, 1)
		right, err := parseTerm(c)
		if err != nil {
			return nil, err
		}
		left = BinExpr{Op: op, Left: left, Right: right}
	}
}

func parseTerm(c *cursor) (Expr, error) {
	left, err := parseFactor(c)
	if err != nil {
		return nil, err
	}
	for {
		skipWS(c)
		b := c.peekN(1)
		if len(b) == 0 || (b[0] != '*' && b[0] != '/') {
			return left, nil
		}
		op := BinMul
		if b[0] == '/' {
			op = BinDiv
		}
		consumeBytes(c, 1)
		right, err := parseFactor(c)
		if err != nil {
			return nil, err
		}
		left = BinExpr{Op: op, Left: left, Right: right}
	}
}

func parseFactor(c *cursor) (Expr, error) {
	skipWS(c)
	b := c.peekN(2)
	if len(b) >= 1 && b[0] == '-' {
		// Negative-number literal: leave the '-' for LitExpr to consume.
		if len(b) >= 2 && ((b[1] >= '0' && b[1] <= '9') || b[1] == '.') {
			return parsePrimary(c)
		}
		consumeBytes(c, 1)
		operand, err := parseFactor(c)
		if err != nil {
			return nil, err
		}
		return UnaryExpr{Op: UnaryNeg, Operand: operand}, nil
	}
	return parsePrimary(c)
}

func parsePrimary(c *cursor) (Expr, error) {
	skipWS(c)
	b := c.peekN(2)
	if len(b) == 0 {
		return nil, fmt.Errorf("expected expression: unexpected EOF")
	}
	b0 := b[0]
	switch {
	case b0 == '(':
		consumeBytes(c, 1)
		e, err := parseOrExpr(c)
		if err != nil {
			return nil, err
		}
		skipWS(c)
		ch, ok := c.readRune()
		if !ok || ch != ')' {
			return nil, fmt.Errorf("expected ')'")
		}
		return e, nil
	case b0 == '"' || b0 == '\'' || (b0 >= '0' && b0 <= '9') || b0 == '.':
		var lit LitExpr
		if err := lit.parse(c); err != nil {
			return nil, err
		}
		return lit, nil
	case (b0 == 'r' || b0 == 'R') && len(b) >= 2 && (b[1] == '"' || b[1] == '\''):
		var lit LitExpr
		if err := lit.parse(c); err != nil {
			return nil, err
		}
		return lit, nil
	case b0 == '`':
		var f Field
		if err := f.parse(c); err != nil {
			return nil, err
		}
		return FieldExpr{Field: f}, nil
	case isIdentStart(b0):
		word := peekKeyword(c)
		switch strings.ToLower(word) {
		case "true", "false", "null":
			var lit LitExpr
			if err := lit.parse(c); err != nil {
				return nil, err
			}
			return lit, nil
		case "and", "or", "not":
			return nil, fmt.Errorf("unexpected keyword %q", word)
		}
		var f Field
		if err := f.parse(c); err != nil {
			return nil, err
		}
		return FieldExpr{Field: f}, nil
	}
	return nil, fmt.Errorf("unexpected character %q", b0)
}

// peekKeyword returns the next ASCII identifier without consuming it.
// Returns "" if the next byte is not an identifier start.
func peekKeyword(c *cursor) string {
	b := c.peekN(16)
	if len(b) == 0 || !isIdentStart(b[0]) {
		return ""
	}
	j := 1
	for j < len(b) && isIdentCont(b[j]) {
		j++
	}
	return string(b[:j])
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isIdentCont(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func consumeBytes(c *cursor, n int) {
	c.advance(n)
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
	c, err := readAllCursor(r)
	if err != nil {
		return err
	}
	return a.parse(c)
}

func (a *Assign) parse(c *cursor) error {
	skipWS(c)
	if err := a.Field.parse(c); err != nil {
		return err
	}

	skipWS(c)
	b := c.peekN(2)
	var op AssignOp
	var n int
	switch {
	case len(b) >= 2 && b[0] == '+' && b[1] == '=':
		op, n = OpAdd, 2
	case len(b) >= 2 && b[0] == '-' && b[1] == '=':
		op, n = OpSub, 2
	case len(b) >= 1 && b[0] == '=':
		op, n = OpSet, 1
	default:
		// Cast-only form: no operator, no value.
		a.Op = OpSet
		a.Value = nil
		return nil
	}
	consumeBytes(c, n)

	e, err := parseOrExpr(c)
	if err != nil {
		return err
	}
	a.Op = op
	a.Value = e

	if lit, ok := e.(LitExpr); ok {
		if err := validateLitAssign(a.Field, lit); err != nil {
			return err
		}
	}
	return nil
}

// validateLitAssign rejects assignments where a literal value cannot possibly
// be cast to the field's declared type. Only catches statically obvious errors;
// runtime cast failures are handled per-clause.
func validateLitAssign(f Field, lit LitExpr) error {
	if lit.Kind == LitNull || f.Type == TypeAny {
		return nil
	}
	if lit.Kind != LitString {
		return nil
	}
	switch f.Type {
	case TypeInt:
		if _, err := strconv.ParseInt(lit.Value, 0, 64); err != nil {
			return fmt.Errorf("cannot assign string %q to int field %q", lit.Value, f.Name)
		}
	case TypeNumber:
		if _, err := strconv.ParseFloat(lit.Value, 64); err != nil {
			return fmt.Errorf("cannot assign string %q to number field %q", lit.Value, f.Name)
		}
	case TypeBool:
		s := strings.ToLower(lit.Value)
		if s != "true" && s != "false" {
			return fmt.Errorf("cannot assign string %q to bool field %q", lit.Value, f.Name)
		}
	}
	return nil
}

func (s *SortTerm) Parse(r io.Reader) error {
	c, err := readAllCursor(r)
	if err != nil {
		return err
	}
	return s.parse(c)
}

func (s *SortTerm) parse(c *cursor) error {
	e, err := parseOrExpr(c)
	if err != nil {
		return err
	}
	s.Expr = e

	skipWS(c)
	kw := peekKeyword(c)
	switch strings.ToLower(kw) {
	case "asc":
		consumeBytes(c, len(kw))
		s.Desc = false
	case "desc":
		consumeBytes(c, len(kw))
		s.Desc = true
	}
	return nil
}
