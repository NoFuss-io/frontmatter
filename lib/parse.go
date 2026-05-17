package lib

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
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

// ── Clause helpers ────────────────────────────────────────────────────────────

// expectKeyword consumes a specific keyword (case-insensitive) and errors otherwise.
func expectKeyword(r *bufio.Reader, kw string) error {
	skipWS(r)
	w := peekKeyword(r)
	if strings.ToLower(w) != kw {
		return fmt.Errorf("expected %q, got %q", kw, w)
	}
	consumeBytes(r, len(w))
	return nil
}

// atStopKeyword returns true if the next token is one of stopKws (case-insensitive)
// and is followed by whitespace or EOF (so it's not a prefix of a longer ident).
func atStopKeyword(r *bufio.Reader, stopKws []string) bool {
	b, _ := r.Peek(32)
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

// readGlobs reads whitespace-separated glob tokens until EOF or a stop keyword.
func readGlobs(r *bufio.Reader, stopKws ...string) []string {
	var out []string
	for {
		skipWS(r)
		b, _ := r.Peek(1)
		if len(b) == 0 {
			break
		}
		if atStopKeyword(r, stopKws) {
			break
		}
		var sb strings.Builder
		for {
			b, _ := r.Peek(1)
			if len(b) == 0 || unicode.IsSpace(rune(b[0])) {
				break
			}
			ch, _, _ := r.ReadRune()
			sb.WriteRune(ch)
		}
		if sb.Len() > 0 {
			out = append(out, sb.String())
		}
	}
	return out
}

// readFieldList reads a comma-separated list of Field values, stopping at EOF or a stop keyword.
func readFieldList(r *bufio.Reader, stopKws ...string) ([]Field, error) {
	var out []Field
	for {
		skipWS(r)
		b, _ := r.Peek(1)
		if len(b) == 0 || atStopKeyword(r, stopKws) {
			break
		}
		var f Field
		if err := f.parse(r); err != nil {
			return nil, err
		}
		out = append(out, f)
		skipWS(r)
		b, _ = r.Peek(1)
		if len(b) == 0 || b[0] != ',' {
			break
		}
		consumeBytes(r, 1)
	}
	return out, nil
}

// readExprList reads a comma-separated list of expressions, stopping at EOF or a stop keyword.
func readExprList(r *bufio.Reader, stopKws ...string) ([]Expr, error) {
	var out []Expr
	for {
		skipWS(r)
		b, _ := r.Peek(1)
		if len(b) == 0 || atStopKeyword(r, stopKws) {
			break
		}
		e, err := parseOrExpr(r)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
		skipWS(r)
		b, _ = r.Peek(1)
		if len(b) == 0 || b[0] != ',' {
			break
		}
		consumeBytes(r, 1)
	}
	return out, nil
}

// readSortTermList reads a comma-separated list of SortTerm values, stopping at EOF or a stop keyword.
func readSortTermList(r *bufio.Reader, stopKws ...string) ([]SortTerm, error) {
	var out []SortTerm
	for {
		skipWS(r)
		b, _ := r.Peek(1)
		if len(b) == 0 || atStopKeyword(r, stopKws) {
			break
		}
		var t SortTerm
		if err := t.parse(r); err != nil {
			return nil, err
		}
		out = append(out, t)
		skipWS(r)
		b, _ = r.Peek(1)
		if len(b) == 0 || b[0] != ',' {
			break
		}
		consumeBytes(r, 1)
	}
	return out, nil
}

// readIntLit reads an optional-sign decimal integer.
func readIntLit(r *bufio.Reader) (int, error) {
	skipWS(r)
	var sb strings.Builder
	if b, _ := r.Peek(1); len(b) > 0 && b[0] == '-' {
		consumeBytes(r, 1)
		sb.WriteByte('-')
	}
	for {
		b, _ := r.Peek(1)
		if len(b) == 0 || b[0] < '0' || b[0] > '9' {
			break
		}
		consumeBytes(r, 1)
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
func readAssignList(r *bufio.Reader, stopKws ...string) ([]Assign, error) {
	var out []Assign
	for {
		skipWS(r)
		b, _ := r.Peek(1)
		if len(b) == 0 || atStopKeyword(r, stopKws) {
			break
		}
		var a Assign
		if err := a.parse(r); err != nil {
			return nil, err
		}
		out = append(out, a)
		skipWS(r)
		b, _ = r.Peek(1)
		if len(b) == 0 || b[0] != ',' {
			break
		}
		consumeBytes(r, 1)
	}
	return out, nil
}

// readRenamePairs reads a comma-separated list of RenamePair values, stopping at EOF or a stop keyword.
func readRenamePairs(r *bufio.Reader, stopKws ...string) ([]RenamePair, error) {
	var out []RenamePair
	for {
		skipWS(r)
		b, _ := r.Peek(1)
		if len(b) == 0 || atStopKeyword(r, stopKws) {
			break
		}
		var p RenamePair
		if err := p.parse(r); err != nil {
			return nil, err
		}
		out = append(out, p)
		skipWS(r)
		b, _ = r.Peek(1)
		if len(b) == 0 || b[0] != ',' {
			break
		}
		consumeBytes(r, 1)
	}
	return out, nil
}

// parseOptionalWhere consumes an optional "where <expr>" clause.
func parseOptionalWhere(r *bufio.Reader, dst *Expr) error {
	skipWS(r)
	if strings.ToLower(peekKeyword(r)) != "where" {
		return nil
	}
	consumeBytes(r, 5)
	e, err := parseOrExpr(r)
	if err != nil {
		return err
	}
	*dst = e
	return nil
}

// ── Queries ───────────────────────────────────────────────────────────────────

// ParseQuery reads a full query from r and returns a SelectQuery, UpdateQuery, or AlterQuery.
func ParseQuery(r io.Reader) (Query, error) {
	return nil, errNotImplemented
}

func (q *SelectQuery) Parse(r io.Reader) error {
	return q.parse(bufio.NewReader(r))
}

func (q *SelectQuery) parse(r *bufio.Reader) error {
	if err := expectKeyword(r, "select"); err != nil {
		return err
	}

	fields, err := readExprList(r, "from")
	if err != nil {
		return err
	}
	if len(fields) == 0 {
		return fmt.Errorf("expected field after 'select'")
	}
	q.Fields = fields

	if err := expectKeyword(r, "from"); err != nil {
		return err
	}

	globs := readGlobs(r, "where", "sort", "limit")
	if len(globs) == 0 {
		return fmt.Errorf("expected glob after 'from'")
	}
	q.From = globs

	if err := parseOptionalWhere(r, &q.Where); err != nil {
		return err
	}

	skipWS(r)
	if strings.ToLower(peekKeyword(r)) == "sort" {
		consumeBytes(r, 4)
		if err := expectKeyword(r, "by"); err != nil {
			return err
		}
		terms, err := readSortTermList(r, "limit")
		if err != nil {
			return err
		}
		if len(terms) == 0 {
			return fmt.Errorf("expected sort term after 'sort by'")
		}
		q.SortBy = terms
	}

	skipWS(r)
	if strings.ToLower(peekKeyword(r)) == "limit" {
		consumeBytes(r, 5)
		n, err := readIntLit(r)
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
	return q.parse(bufio.NewReader(r))
}

func (q *UpdateQuery) parse(r *bufio.Reader) error {
	if err := expectKeyword(r, "update"); err != nil {
		return err
	}

	globs := readGlobs(r, "set")
	if len(globs) == 0 {
		return fmt.Errorf("expected glob after 'update'")
	}
	q.From = globs

	if err := expectKeyword(r, "set"); err != nil {
		return err
	}

	assigns, err := readAssignList(r, "where")
	if err != nil {
		return err
	}
	if len(assigns) == 0 {
		return fmt.Errorf("expected assignment after 'set'")
	}
	q.Set = assigns

	return parseOptionalWhere(r, &q.Where)
}

func (q *AlterQuery) Parse(r io.Reader) error {
	return q.parse(bufio.NewReader(r))
}

func (q *AlterQuery) parse(r *bufio.Reader) error {
	if err := expectKeyword(r, "alter"); err != nil {
		return err
	}

	globs := readGlobs(r, "drop", "rename")
	if len(globs) == 0 {
		return fmt.Errorf("expected glob after 'alter'")
	}
	q.From = globs

	skipWS(r)
	kw := peekKeyword(r)
	switch strings.ToLower(kw) {
	case "drop":
		consumeBytes(r, len(kw))
		q.Op = AlterDrop
		fields, err := readFieldList(r, "where")
		if err != nil {
			return err
		}
		if len(fields) == 0 {
			return fmt.Errorf("expected field after 'drop'")
		}
		q.Drop = fields
	case "rename":
		consumeBytes(r, len(kw))
		q.Op = AlterRename
		pairs, err := readRenamePairs(r, "where")
		if err != nil {
			return err
		}
		if len(pairs) == 0 {
			return fmt.Errorf("expected rename pair after 'rename'")
		}
		q.Rename = pairs
	default:
		return fmt.Errorf("expected 'drop' or 'rename' after globs")
	}

	return parseOptionalWhere(r, &q.Where)
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
	return parseOrExpr(bufio.NewReader(r))
}

func parseOrExpr(r *bufio.Reader) (Expr, error) {
	left, err := parseAndExpr(r)
	if err != nil {
		return nil, err
	}
	for {
		skipWS(r)
		kw := peekKeyword(r)
		if strings.ToLower(kw) != "or" {
			return left, nil
		}
		consumeBytes(r, len(kw))
		right, err := parseAndExpr(r)
		if err != nil {
			return nil, err
		}
		left = BinExpr{Op: BinOr, Left: left, Right: right}
	}
}

func parseAndExpr(r *bufio.Reader) (Expr, error) {
	left, err := parseNotExpr(r)
	if err != nil {
		return nil, err
	}
	for {
		skipWS(r)
		kw := peekKeyword(r)
		if strings.ToLower(kw) != "and" {
			return left, nil
		}
		consumeBytes(r, len(kw))
		right, err := parseNotExpr(r)
		if err != nil {
			return nil, err
		}
		left = BinExpr{Op: BinAnd, Left: left, Right: right}
	}
}

func parseNotExpr(r *bufio.Reader) (Expr, error) {
	skipWS(r)
	kw := peekKeyword(r)
	if strings.ToLower(kw) == "not" {
		consumeBytes(r, len(kw))
		inner, err := parseComparison(r)
		if err != nil {
			return nil, err
		}
		return UnaryExpr{Op: UnaryNot, Operand: inner}, nil
	}
	return parseComparison(r)
}

func parseComparison(r *bufio.Reader) (Expr, error) {
	left, err := parseArith(r)
	if err != nil {
		return nil, err
	}
	skipWS(r)
	b, _ := r.Peek(2)
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
	consumeBytes(r, n)
	right, err := parseArith(r)
	if err != nil {
		return nil, err
	}
	return BinExpr{Op: op, Left: left, Right: right}, nil
}

func parseArith(r *bufio.Reader) (Expr, error) {
	left, err := parseTerm(r)
	if err != nil {
		return nil, err
	}
	for {
		skipWS(r)
		b, _ := r.Peek(1)
		if len(b) == 0 || (b[0] != '+' && b[0] != '-') {
			return left, nil
		}
		op := BinAdd
		if b[0] == '-' {
			op = BinSub
		}
		consumeBytes(r, 1)
		right, err := parseTerm(r)
		if err != nil {
			return nil, err
		}
		left = BinExpr{Op: op, Left: left, Right: right}
	}
}

func parseTerm(r *bufio.Reader) (Expr, error) {
	left, err := parseFactor(r)
	if err != nil {
		return nil, err
	}
	for {
		skipWS(r)
		b, _ := r.Peek(1)
		if len(b) == 0 || (b[0] != '*' && b[0] != '/') {
			return left, nil
		}
		op := BinMul
		if b[0] == '/' {
			op = BinDiv
		}
		consumeBytes(r, 1)
		right, err := parseFactor(r)
		if err != nil {
			return nil, err
		}
		left = BinExpr{Op: op, Left: left, Right: right}
	}
}

func parseFactor(r *bufio.Reader) (Expr, error) {
	skipWS(r)
	b, _ := r.Peek(2)
	if len(b) >= 1 && b[0] == '-' {
		// Negative-number literal: leave the '-' for LitExpr to consume.
		if len(b) >= 2 && ((b[1] >= '0' && b[1] <= '9') || b[1] == '.') {
			return parsePrimary(r)
		}
		consumeBytes(r, 1)
		operand, err := parseFactor(r)
		if err != nil {
			return nil, err
		}
		return UnaryExpr{Op: UnaryNeg, Operand: operand}, nil
	}
	return parsePrimary(r)
}

func parsePrimary(r *bufio.Reader) (Expr, error) {
	skipWS(r)
	b, _ := r.Peek(2)
	if len(b) == 0 {
		return nil, fmt.Errorf("expected expression: unexpected EOF")
	}
	b0 := b[0]
	switch {
	case b0 == '(':
		consumeBytes(r, 1)
		e, err := parseOrExpr(r)
		if err != nil {
			return nil, err
		}
		skipWS(r)
		ch, _, err := r.ReadRune()
		if err != nil || ch != ')' {
			return nil, fmt.Errorf("expected ')'")
		}
		return e, nil
	case b0 == '"' || b0 == '\'' || (b0 >= '0' && b0 <= '9') || b0 == '.':
		var lit LitExpr
		if err := lit.parse(r); err != nil {
			return nil, err
		}
		return lit, nil
	case (b0 == 'r' || b0 == 'R') && len(b) >= 2 && (b[1] == '"' || b[1] == '\''):
		var lit LitExpr
		if err := lit.parse(r); err != nil {
			return nil, err
		}
		return lit, nil
	case b0 == '`':
		var f Field
		if err := f.parse(r); err != nil {
			return nil, err
		}
		return FieldExpr{Field: f}, nil
	case isIdentStart(b0):
		word := peekKeyword(r)
		switch strings.ToLower(word) {
		case "true", "false", "null":
			var lit LitExpr
			if err := lit.parse(r); err != nil {
				return nil, err
			}
			return lit, nil
		case "and", "or", "not":
			return nil, fmt.Errorf("unexpected keyword %q", word)
		}
		var f Field
		if err := f.parse(r); err != nil {
			return nil, err
		}
		return FieldExpr{Field: f}, nil
	}
	return nil, fmt.Errorf("unexpected character %q", b0)
}

// peekKeyword returns the next ASCII identifier without consuming it.
// Returns "" if the next byte is not an identifier start.
func peekKeyword(r *bufio.Reader) string {
	b, _ := r.Peek(16)
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

func consumeBytes(r *bufio.Reader, n int) {
	for i := 0; i < n; i++ {
		_, _, _ = r.ReadRune()
	}
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
	return a.parse(bufio.NewReader(r))
}

func (a *Assign) parse(r *bufio.Reader) error {
	skipWS(r)
	if err := a.Field.parse(r); err != nil {
		return err
	}

	skipWS(r)
	b, _ := r.Peek(2)
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
	consumeBytes(r, n)

	e, err := parseOrExpr(r)
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
	return s.parse(bufio.NewReader(r))
}

func (s *SortTerm) parse(r *bufio.Reader) error {
	e, err := parseOrExpr(r)
	if err != nil {
		return err
	}
	s.Expr = e

	skipWS(r)
	kw := peekKeyword(r)
	switch strings.ToLower(kw) {
	case "asc":
		consumeBytes(r, len(kw))
		s.Desc = false
	case "desc":
		consumeBytes(r, len(kw))
		s.Desc = true
	}
	return nil
}
