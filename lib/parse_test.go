package lib_test

import (
	"strings"
	"testing"

	"github.com/backlin/frontmatter/lib"
)

// r wraps a string in a Reader for use with Parse methods.
func r(s string) *strings.Reader { return strings.NewReader(s) }

// ptr returns a pointer to v, for constructing *FieldType in expected values.
func ptr[T any](v T) *T { return &v }

// ── Field ─────────────────────────────────────────────────────────────────────

func TestField_Parse(t *testing.T) {
	tests := []struct {
		in      string
		want    lib.Field
		wantErr bool
	}{
		// Untyped identifier
		{in: "title", want: lib.Field{Name: "title", Type: lib.TypeAny}},
		{in: "_private", want: lib.Field{Name: "_private", Type: lib.TypeAny}},
		{in: "field_123", want: lib.Field{Name: "field_123", Type: lib.TypeAny}},

		// Typed identifiers
		{in: "title:string", want: lib.Field{Name: "title", Type: lib.TypeString}},
		{in: "active:bool", want: lib.Field{Name: "active", Type: lib.TypeBool}},
		{in: "count:int", want: lib.Field{Name: "count", Type: lib.TypeInt}},
		{in: "price:numeric", want: lib.Field{Name: "price", Type: lib.TypeNumber}},
		{in: "created:date", want: lib.Field{Name: "created", Type: lib.TypeDate}},
		{in: "modified:datetime", want: lib.Field{Name: "modified", Type: lib.TypeDatetime}},
		{in: "ref:link", want: lib.Field{Name: "ref", Type: lib.TypeLink}},
		{in: "ref:mdlink", want: lib.Field{Name: "ref", Type: lib.TypeMdLink}},

		// List types
		{in: "tags:list", want: lib.Field{Name: "tags", Type: lib.TypeList}},
		{in: "tags:list:string", want: lib.Field{Name: "tags", Type: lib.TypeList, ElemType: ptr(lib.TypeString)}},
		{in: "ratings:list:int", want: lib.Field{Name: "ratings", Type: lib.TypeList, ElemType: ptr(lib.TypeInt)}},
		{in: "dates:list:date", want: lib.Field{Name: "dates", Type: lib.TypeList, ElemType: ptr(lib.TypeDate)}},

		// Quoted identifiers (backtick) — allow spaces, symbols, reserved words
		{in: "`created-at`", want: lib.Field{Name: "created-at", Type: lib.TypeAny}},
		{in: "`field with spaces`", want: lib.Field{Name: "field with spaces", Type: lib.TypeAny}},
		{in: "`from`", want: lib.Field{Name: "from", Type: lib.TypeAny}}, // reserved keyword
		{in: "`from`:date", want: lib.Field{Name: "from", Type: lib.TypeDate}},

		// Errors
		{in: "", wantErr: true},                 // empty
		{in: "1foo", wantErr: true},             // starts with digit
		{in: "foo:unknown", wantErr: true},      // unknown type
		{in: "``", wantErr: true},               // empty quoted identifier
		{in: "foo:list:unknown", wantErr: true}, // unknown list element type
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			var f lib.Field
			err := f.Parse(r(tc.in))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q): expected error, got nil", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error: %v", tc.in, err)
			}
			if f.Name != tc.want.Name {
				t.Errorf("Name = %q, want %q", f.Name, tc.want.Name)
			}
			if f.Type != tc.want.Type {
				t.Errorf("Type = %v, want %v", f.Type, tc.want.Type)
			}
			if tc.want.ElemType == nil && f.ElemType != nil {
				t.Errorf("ElemType = %v, want nil", *f.ElemType)
			}
			if tc.want.ElemType != nil {
				if f.ElemType == nil {
					t.Errorf("ElemType = nil, want %v", *tc.want.ElemType)
				} else if *f.ElemType != *tc.want.ElemType {
					t.Errorf("ElemType = %v, want %v", *f.ElemType, *tc.want.ElemType)
				}
			}
		})
	}
}

// ── LitExpr ───────────────────────────────────────────────────────────────────

func TestLitExpr_Parse(t *testing.T) {
	tests := []struct {
		in      string
		want    lib.LitExpr
		wantErr bool
	}{
		// Integer literals
		{in: "42", want: lib.LitExpr{Kind: lib.LitInt, Value: "42"}},
		{in: "-7", want: lib.LitExpr{Kind: lib.LitInt, Value: "-7"}},
		{in: "0", want: lib.LitExpr{Kind: lib.LitInt, Value: "0"}},
		{in: "0xFF", want: lib.LitExpr{Kind: lib.LitInt, Value: "0xFF"}},
		{in: "0x1A2B", want: lib.LitExpr{Kind: lib.LitInt, Value: "0x1A2B"}},

		// Numeric (float) literals
		{in: "3.14", want: lib.LitExpr{Kind: lib.LitNumeric, Value: "3.14"}},
		{in: "-2.5", want: lib.LitExpr{Kind: lib.LitNumeric, Value: "-2.5"}},
		{in: "1.0e10", want: lib.LitExpr{Kind: lib.LitNumeric, Value: "1.0e10"}},
		{in: "6.022E-23", want: lib.LitExpr{Kind: lib.LitNumeric, Value: "6.022E-23"}},
		{in: ".5", want: lib.LitExpr{Kind: lib.LitNumeric, Value: ".5"}},

		// Boolean literals (case insensitive per spec)
		{in: "true", want: lib.LitExpr{Kind: lib.LitBool, Value: "true"}},
		{in: "false", want: lib.LitExpr{Kind: lib.LitBool, Value: "false"}},
		{in: "TRUE", want: lib.LitExpr{Kind: lib.LitBool, Value: "TRUE"}},
		{in: "False", want: lib.LitExpr{Kind: lib.LitBool, Value: "False"}},

		// Null literal (case insensitive per spec)
		{in: "null", want: lib.LitExpr{Kind: lib.LitNull, Value: "null"}},
		{in: "NULL", want: lib.LitExpr{Kind: lib.LitNull, Value: "NULL"}},

		// Double-quoted strings
		{in: `"hello"`, want: lib.LitExpr{Kind: lib.LitString, Value: "hello"}},
		{in: `"hello world"`, want: lib.LitExpr{Kind: lib.LitString, Value: "hello world"}},
		{in: `""`, want: lib.LitExpr{Kind: lib.LitString, Value: ""}},
		{in: `"it's a \"test\""`, want: lib.LitExpr{Kind: lib.LitString, Value: `it's a "test"`}},

		// Single-quoted strings
		{in: `'hello'`, want: lib.LitExpr{Kind: lib.LitString, Value: "hello"}},
		{in: `'it\'s fine'`, want: lib.LitExpr{Kind: lib.LitString, Value: "it's fine"}},

		// Triple-quoted strings (multi-line)
		{in: `"""hello"""`, want: lib.LitExpr{Kind: lib.LitString, Value: "hello"}},
		{in: "'''line1\nline2'''", want: lib.LitExpr{Kind: lib.LitString, Value: "line1\nline2"}},

		// Raw strings (backslash not interpreted)
		{in: `r"C:\Users\name"`, want: lib.LitExpr{Kind: lib.LitString, Value: `C:\Users\name`}},
		{in: `r'no\escape'`, want: lib.LitExpr{Kind: lib.LitString, Value: `no\escape`}},

		// Escape sequences in regular strings
		{in: `"\n"`, want: lib.LitExpr{Kind: lib.LitString, Value: "\n"}},
		{in: `"\t"`, want: lib.LitExpr{Kind: lib.LitString, Value: "\t"}},
		{in: `"\\"`, want: lib.LitExpr{Kind: lib.LitString, Value: `\`}},

		// Errors
		{in: "", wantErr: true},              // empty
		{in: `"unterminated`, wantErr: true}, // missing closing quote
		{in: `'unterminated`, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			var lit lib.LitExpr
			err := lit.Parse(r(tc.in))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q): expected error, got nil", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error: %v", tc.in, err)
			}
			if lit.Kind != tc.want.Kind {
				t.Errorf("Kind = %v, want %v", lit.Kind, tc.want.Kind)
			}
			if lit.Value != tc.want.Value {
				t.Errorf("Value = %q, want %q", lit.Value, tc.want.Value)
			}
		})
	}
}

// ── ParseExpr ─────────────────────────────────────────────────────────────────

func TestParseExpr(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
		// check is called with the parsed Expr if wantErr is false.
		// Use type assertions to verify the tree structure.
		check func(t *testing.T, e lib.Expr)
	}{
		// Atoms
		{
			in: "title",
			check: func(t *testing.T, e lib.Expr) {
				fe, ok := e.(lib.FieldExpr)
				if !ok {
					t.Fatalf("got %T, want FieldExpr", e)
				}
				if fe.Field.Name != "title" {
					t.Errorf("Name = %q, want %q", fe.Field.Name, "title")
				}
			},
		},
		{
			in: "42",
			check: func(t *testing.T, e lib.Expr) {
				lit, ok := e.(lib.LitExpr)
				if !ok {
					t.Fatalf("got %T, want LitExpr", e)
				}
				if lit.Kind != lib.LitInt {
					t.Errorf("Kind = %v, want LitInt", lit.Kind)
				}
			},
		},

		// Comparison
		{
			in: "price:numeric > 10",
			check: func(t *testing.T, e lib.Expr) {
				bin, ok := e.(lib.BinExpr)
				if !ok {
					t.Fatalf("got %T, want BinExpr", e)
				}
				if bin.Op != lib.BinGt {
					t.Errorf("Op = %v, want BinGt", bin.Op)
				}
				lhs, ok := bin.Left.(lib.FieldExpr)
				if !ok {
					t.Fatalf("Left: got %T, want FieldExpr", bin.Left)
				}
				if lhs.Field.Name != "price" || lhs.Field.Type != lib.TypeNumber {
					t.Errorf("Left = %+v, want {price:numeric}", lhs.Field)
				}
				rhs, ok := bin.Right.(lib.LitExpr)
				if !ok {
					t.Fatalf("Right: got %T, want LitExpr", bin.Right)
				}
				if rhs.Kind != lib.LitInt || rhs.Value != "10" {
					t.Errorf("Right = %+v, want LitInt 10", rhs)
				}
			},
		},
		{
			in: `url:string = "https://example.com"`,
			check: func(t *testing.T, e lib.Expr) {
				bin, ok := e.(lib.BinExpr)
				if !ok {
					t.Fatalf("got %T, want BinExpr", e)
				}
				if bin.Op != lib.BinEq {
					t.Errorf("Op = %v, want BinEq", bin.Op)
				}
			},
		},

		// Unary not
		{
			in: "not active",
			check: func(t *testing.T, e lib.Expr) {
				un, ok := e.(lib.UnaryExpr)
				if !ok {
					t.Fatalf("got %T, want UnaryExpr", e)
				}
				if un.Op != lib.UnaryNot {
					t.Errorf("Op = %v, want UnaryNot", un.Op)
				}
			},
		},

		// and binds tighter than or: "a and b or c" == "(a and b) or c"
		{
			in: "a and b or c",
			check: func(t *testing.T, e lib.Expr) {
				top, ok := e.(lib.BinExpr)
				if !ok {
					t.Fatalf("got %T, want BinExpr", e)
				}
				if top.Op != lib.BinOr {
					t.Errorf("top Op = %v, want BinOr", top.Op)
				}
				left, ok := top.Left.(lib.BinExpr)
				if !ok {
					t.Fatalf("Left: got %T, want BinExpr", top.Left)
				}
				if left.Op != lib.BinAnd {
					t.Errorf("left Op = %v, want BinAnd", left.Op)
				}
			},
		},

		// Parentheses override precedence: "(a or b) and c"
		{
			in: "(a or b) and c",
			check: func(t *testing.T, e lib.Expr) {
				top, ok := e.(lib.BinExpr)
				if !ok {
					t.Fatalf("got %T, want BinExpr", e)
				}
				if top.Op != lib.BinAnd {
					t.Errorf("top Op = %v, want BinAnd", top.Op)
				}
				left, ok := top.Left.(lib.BinExpr)
				if !ok {
					t.Fatalf("Left: got %T, want BinExpr", top.Left)
				}
				if left.Op != lib.BinOr {
					t.Errorf("left Op = %v, want BinOr", left.Op)
				}
			},
		},

		// Arithmetic: * binds tighter than +
		{
			in: "a + b * c",
			check: func(t *testing.T, e lib.Expr) {
				top, ok := e.(lib.BinExpr)
				if !ok {
					t.Fatalf("got %T, want BinExpr", e)
				}
				if top.Op != lib.BinAdd {
					t.Errorf("top Op = %v, want BinAdd", top.Op)
				}
				right, ok := top.Right.(lib.BinExpr)
				if !ok {
					t.Fatalf("Right: got %T, want BinExpr", top.Right)
				}
				if right.Op != lib.BinMul {
					t.Errorf("right Op = %v, want BinMul", right.Op)
				}
			},
		},

		// Unary arithmetic negation
		{
			in: "-price",
			check: func(t *testing.T, e lib.Expr) {
				un, ok := e.(lib.UnaryExpr)
				if !ok {
					t.Fatalf("got %T, want UnaryExpr", e)
				}
				if un.Op != lib.UnaryNeg {
					t.Errorf("Op = %v, want UnaryNeg", un.Op)
				}
			},
		},

		// Errors
		{in: "", wantErr: true},
		{in: "a and", wantErr: true},   // dangling operator
		{in: "(a or b", wantErr: true}, // unclosed paren
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			e, err := lib.ParseExpr(r(tc.in))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseExpr(%q): expected error, got nil", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseExpr(%q): unexpected error: %v", tc.in, err)
			}
			if tc.check != nil {
				tc.check(t, e)
			}
		})
	}
}

// ── Assign ────────────────────────────────────────────────────────────────────

func TestAssign_Parse(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
		check   func(t *testing.T, a lib.Assign)
	}{
		// Cast-only form: no operator, just field with type
		{
			in: "foo:int",
			check: func(t *testing.T, a lib.Assign) {
				if a.Field.Name != "foo" || a.Field.Type != lib.TypeInt {
					t.Errorf("Field = %+v, want {foo:int}", a.Field)
				}
				if a.Op != lib.OpSet {
					t.Errorf("Op = %v, want OpSet", a.Op)
				}
				if a.Value != nil {
					t.Errorf("Value = %v, want nil", a.Value)
				}
			},
		},
		// Set literal
		{
			in: `title = "hello"`,
			check: func(t *testing.T, a lib.Assign) {
				if a.Field.Name != "title" {
					t.Errorf("Field.Name = %q, want title", a.Field.Name)
				}
				if a.Op != lib.OpSet {
					t.Errorf("Op = %v, want OpSet", a.Op)
				}
				lit, ok := a.Value.(lib.LitExpr)
				if !ok {
					t.Fatalf("Value: got %T, want LitExpr", a.Value)
				}
				if lit.Kind != lib.LitString || lit.Value != "hello" {
					t.Errorf("Value = %+v, want LitString hello", lit)
				}
			},
		},
		// Set null
		{
			in: "title = null",
			check: func(t *testing.T, a lib.Assign) {
				if a.Op != lib.OpSet {
					t.Errorf("Op = %v, want OpSet", a.Op)
				}
				lit, ok := a.Value.(lib.LitExpr)
				if !ok {
					t.Fatalf("Value: got %T, want LitExpr", a.Value)
				}
				if lit.Kind != lib.LitNull {
					t.Errorf("Kind = %v, want LitNull", lit.Kind)
				}
			},
		},
		// Set from field reference
		{
			in: "title = source",
			check: func(t *testing.T, a lib.Assign) {
				_, ok := a.Value.(lib.FieldExpr)
				if !ok {
					t.Fatalf("Value: got %T, want FieldExpr", a.Value)
				}
			},
		},
		// List addition
		{
			in: `tags += "recipe"`,
			check: func(t *testing.T, a lib.Assign) {
				if a.Op != lib.OpAdd {
					t.Errorf("Op = %v, want OpAdd", a.Op)
				}
			},
		},
		// List subtraction
		{
			in: `tags -= "recipe"`,
			check: func(t *testing.T, a lib.Assign) {
				if a.Op != lib.OpSub {
					t.Errorf("Op = %v, want OpSub", a.Op)
				}
			},
		},
		// Numeric increment via field ref
		{
			in: "count += 1",
			check: func(t *testing.T, a lib.Assign) {
				if a.Op != lib.OpAdd {
					t.Errorf("Op = %v, want OpAdd", a.Op)
				}
				lit, ok := a.Value.(lib.LitExpr)
				if !ok {
					t.Fatalf("Value: got %T, want LitExpr", a.Value)
				}
				if lit.Kind != lib.LitInt {
					t.Errorf("Kind = %v, want LitInt", lit.Kind)
				}
			},
		},
		// Static type error: string cannot be assigned to int
		{in: `foo:int = "hello"`, wantErr: true},

		// Errors
		{in: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			var a lib.Assign
			err := a.Parse(r(tc.in))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q): expected error, got nil", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error: %v", tc.in, err)
			}
			if tc.check != nil {
				tc.check(t, a)
			}
		})
	}
}

// ── SortTerm ──────────────────────────────────────────────────────────────────

func TestSortTerm_Parse(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
		check   func(t *testing.T, s lib.SortTerm)
	}{
		{
			in: "title",
			check: func(t *testing.T, s lib.SortTerm) {
				if s.Desc {
					t.Error("Desc = true, want false (default is asc)")
				}
				fe, ok := s.Expr.(lib.FieldExpr)
				if !ok {
					t.Fatalf("Expr: got %T, want FieldExpr", s.Expr)
				}
				if fe.Field.Name != "title" {
					t.Errorf("Name = %q, want title", fe.Field.Name)
				}
			},
		},
		{
			in: "date desc",
			check: func(t *testing.T, s lib.SortTerm) {
				if !s.Desc {
					t.Error("Desc = false, want true")
				}
			},
		},
		{
			in: "price:numeric asc",
			check: func(t *testing.T, s lib.SortTerm) {
				if s.Desc {
					t.Error("Desc = true, want false")
				}
				fe, ok := s.Expr.(lib.FieldExpr)
				if !ok {
					t.Fatalf("Expr: got %T, want FieldExpr", s.Expr)
				}
				if fe.Field.Type != lib.TypeNumber {
					t.Errorf("Type = %v, want TypeNumber", fe.Field.Type)
				}
			},
		},
		{in: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			var s lib.SortTerm
			err := s.Parse(r(tc.in))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q): expected error, got nil", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error: %v", tc.in, err)
			}
			if tc.check != nil {
				tc.check(t, s)
			}
		})
	}
}

// ── RenamePair ────────────────────────────────────────────────────────────────

func TestRenamePair_Parse(t *testing.T) {
	tests := []struct {
		in      string
		want    lib.RenamePair
		wantErr bool
	}{
		{in: "foo to bar", want: lib.RenamePair{From: "foo", To: "bar"}},
		{in: "old_name to new_name", want: lib.RenamePair{From: "old_name", To: "new_name"}},
		{in: "`created-at` to created_at", want: lib.RenamePair{From: "created-at", To: "created_at"}},
		// Errors
		{in: "foo", wantErr: true},    // missing "to"
		{in: "foo to", wantErr: true}, // missing target
		{in: "to bar", wantErr: true}, // missing source
		{in: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			var p lib.RenamePair
			err := p.Parse(r(tc.in))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q): expected error, got nil", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error: %v", tc.in, err)
			}
			if p != tc.want {
				t.Errorf("got %+v, want %+v", p, tc.want)
			}
		})
	}
}

// ── SelectQuery ───────────────────────────────────────────────────────────────

func TestSelectQuery_Parse(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
		check   func(t *testing.T, q lib.SelectQuery)
	}{
		{
			in: "select title from *.md",
			check: func(t *testing.T, q lib.SelectQuery) {
				if len(q.Fields) != 1 {
					t.Fatalf("len(Fields) = %d, want 1", len(q.Fields))
				}
				if len(q.From) != 1 || q.From[0] != "*.md" {
					t.Errorf("From = %v, want [*.md]", q.From)
				}
				if q.Where != nil {
					t.Error("Where != nil, want nil")
				}
				if q.Limit != 0 {
					t.Errorf("Limit = %d, want 0 (no limit)", q.Limit)
				}
			},
		},
		{
			in: "select title, date from *.md",
			check: func(t *testing.T, q lib.SelectQuery) {
				if len(q.Fields) != 2 {
					t.Errorf("len(Fields) = %d, want 2", len(q.Fields))
				}
			},
		},
		{
			in: "select title from *.md where published = true",
			check: func(t *testing.T, q lib.SelectQuery) {
				if q.Where == nil {
					t.Fatal("Where = nil, want expression")
				}
			},
		},
		{
			in: "select title from *.md sort by date desc",
			check: func(t *testing.T, q lib.SelectQuery) {
				if len(q.SortBy) != 1 {
					t.Fatalf("len(SortBy) = %d, want 1", len(q.SortBy))
				}
				if !q.SortBy[0].Desc {
					t.Error("SortBy[0].Desc = false, want true")
				}
			},
		},
		{
			in: "select title from *.md sort by date desc, title asc",
			check: func(t *testing.T, q lib.SelectQuery) {
				if len(q.SortBy) != 2 {
					t.Fatalf("len(SortBy) = %d, want 2", len(q.SortBy))
				}
			},
		},
		{
			in: "select title from *.md limit 5",
			check: func(t *testing.T, q lib.SelectQuery) {
				if q.Limit != 5 {
					t.Errorf("Limit = %d, want 5", q.Limit)
				}
			},
		},
		{
			in: "select title from *.md sort by date desc limit 10",
			check: func(t *testing.T, q lib.SelectQuery) {
				if len(q.SortBy) != 1 {
					t.Fatalf("len(SortBy) = %d, want 1", len(q.SortBy))
				}
				if q.Limit != 10 {
					t.Errorf("Limit = %d, want 10", q.Limit)
				}
			},
		},
		// Multiple globs
		{
			in: "select title from a/*.md b/*.md",
			check: func(t *testing.T, q lib.SelectQuery) {
				if len(q.From) != 2 {
					t.Errorf("From = %v, want 2 globs", q.From)
				}
			},
		},
		// Errors
		{in: "select title", wantErr: true},                     // missing from
		{in: "select from *.md", wantErr: true},                 // missing fields
		{in: "select title from", wantErr: true},                // missing glob
		{in: "select title from *.md limit -1", wantErr: true},  // negative limit
		{in: "select title from *.md limit foo", wantErr: true}, // non-integer limit
		{in: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			var q lib.SelectQuery
			err := q.Parse(r(tc.in))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q): expected error, got nil", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error: %v", tc.in, err)
			}
			if tc.check != nil {
				tc.check(t, q)
			}
		})
	}
}

// ── UpdateQuery ───────────────────────────────────────────────────────────────

func TestUpdateQuery_Parse(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
		check   func(t *testing.T, q lib.UpdateQuery)
	}{
		{
			in: `update *.md set title = "hello"`,
			check: func(t *testing.T, q lib.UpdateQuery) {
				if len(q.From) != 1 || q.From[0] != "*.md" {
					t.Errorf("From = %v, want [*.md]", q.From)
				}
				if len(q.Set) != 1 {
					t.Fatalf("len(Set) = %d, want 1", len(q.Set))
				}
				if q.Where != nil {
					t.Error("Where != nil, want nil")
				}
			},
		},
		{
			in: `update *.md set tags += "recipe", count -= 1`,
			check: func(t *testing.T, q lib.UpdateQuery) {
				if len(q.Set) != 2 {
					t.Errorf("len(Set) = %d, want 2", len(q.Set))
				}
			},
		},
		{
			in: "update *.md set foo:int where active",
			check: func(t *testing.T, q lib.UpdateQuery) {
				if len(q.Set) != 1 {
					t.Fatalf("len(Set) = %d, want 1", len(q.Set))
				}
				if q.Set[0].Value != nil {
					t.Error("Set[0].Value != nil, want nil (cast-only form)")
				}
				if q.Where == nil {
					t.Error("Where = nil, want expression")
				}
			},
		},
		// Errors
		{in: "update *.md", wantErr: true},        // missing set
		{in: "update set foo = 1", wantErr: true}, // missing glob
		{in: "update *.md set", wantErr: true},    // missing assignments
		{in: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			var q lib.UpdateQuery
			err := q.Parse(r(tc.in))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q): expected error, got nil", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error: %v", tc.in, err)
			}
			if tc.check != nil {
				tc.check(t, q)
			}
		})
	}
}

// ── AlterQuery ────────────────────────────────────────────────────────────────

func TestAlterQuery_Parse(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
		check   func(t *testing.T, q lib.AlterQuery)
	}{
		{
			in: "alter *.md drop title",
			check: func(t *testing.T, q lib.AlterQuery) {
				if q.Op != lib.AlterDrop {
					t.Errorf("Op = %v, want AlterDrop", q.Op)
				}
				if len(q.Drop) != 1 || q.Drop[0].Name != "title" {
					t.Errorf("Drop = %v, want [title]", q.Drop)
				}
			},
		},
		{
			in: "alter *.md drop title, date",
			check: func(t *testing.T, q lib.AlterQuery) {
				if len(q.Drop) != 2 {
					t.Errorf("len(Drop) = %d, want 2", len(q.Drop))
				}
			},
		},
		{
			in: "alter *.md rename foo to bar",
			check: func(t *testing.T, q lib.AlterQuery) {
				if q.Op != lib.AlterRename {
					t.Errorf("Op = %v, want AlterRename", q.Op)
				}
				if len(q.Rename) != 1 {
					t.Fatalf("len(Rename) = %d, want 1", len(q.Rename))
				}
				if q.Rename[0].From != "foo" || q.Rename[0].To != "bar" {
					t.Errorf("Rename[0] = %+v, want {foo bar}", q.Rename[0])
				}
			},
		},
		{
			in: "alter *.md rename foo to bar, baz to qux",
			check: func(t *testing.T, q lib.AlterQuery) {
				if len(q.Rename) != 2 {
					t.Errorf("len(Rename) = %d, want 2", len(q.Rename))
				}
			},
		},
		{
			in: "alter *.md drop title where published = false",
			check: func(t *testing.T, q lib.AlterQuery) {
				if q.Where == nil {
					t.Error("Where = nil, want expression")
				}
			},
		},
		// Errors
		{in: "alter *.md", wantErr: true},            // missing drop/rename
		{in: "alter *.md drop", wantErr: true},       // missing fields
		{in: "alter *.md rename foo", wantErr: true}, // missing "to target"
		{in: "alter drop title", wantErr: true},      // missing glob
		{in: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			var q lib.AlterQuery
			err := q.Parse(r(tc.in))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q): expected error, got nil", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error: %v", tc.in, err)
			}
			if tc.check != nil {
				tc.check(t, q)
			}
		})
	}
}

// ── ParseQuery (top-level dispatcher) ────────────────────────────────────────

func TestParseQuery(t *testing.T) {
	tests := []struct {
		in       string
		wantType string // "select", "update", "alter"
		wantErr  bool
	}{
		{in: "select title from *.md", wantType: "select"},
		{in: `update *.md set title = "foo"`, wantType: "update"},
		{in: "alter *.md drop title", wantType: "alter"},
		{in: "alter *.md rename foo to bar", wantType: "alter"},
		// Errors
		{in: "", wantErr: true},
		{in: "delete from *.md", wantErr: true}, // unsupported keyword
		{in: "foobar", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			q, err := lib.ParseQuery(r(tc.in))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseQuery(%q): expected error, got nil", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseQuery(%q): unexpected error: %v", tc.in, err)
			}
			switch tc.wantType {
			case "select":
				if _, ok := q.(lib.SelectQuery); !ok {
					t.Errorf("got %T, want SelectQuery", q)
				}
			case "update":
				if _, ok := q.(lib.UpdateQuery); !ok {
					t.Errorf("got %T, want UpdateQuery", q)
				}
			case "alter":
				if _, ok := q.(lib.AlterQuery); !ok {
					t.Errorf("got %T, want AlterQuery", q)
				}
			}
		})
	}
}

// ── ParseProgram ─────────────────────────────────────────────────────────────

func TestParseProgram(t *testing.T) {
	type stmtKind int
	const (
		kSelect stmtKind = iota
		kUpdate
		kAlter
	)

	tests := []struct {
		name    string
		in      string
		want    []stmtKind
		wantErr bool
	}{
		{name: "empty", in: "", want: nil},
		{name: "only whitespace", in: "  \n\t\n", want: nil},
		{name: "only comments", in: "-- foo\n-- bar\n", want: nil},
		{name: "only separators", in: ";;;\n;", want: nil},

		{name: "single no terminator", in: "select x from a", want: []stmtKind{kSelect}},
		{name: "single with terminator", in: "select x from a;", want: []stmtKind{kSelect}},
		{name: "trailing comment", in: "select x from a; -- done", want: []stmtKind{kSelect}},

		{
			name: "multi mixed",
			in: `select x from a;
update b/* set y = 1;
alter c/* drop z;`,
			want: []stmtKind{kSelect, kUpdate, kAlter},
		},
		{
			name: "consecutive separators tolerated",
			in:   "select x from a;;\n;;select y from b;",
			want: []stmtKind{kSelect, kSelect},
		},
		{
			name: "comments interleaved with statements",
			in: `-- header
select x from a;
-- middle
select y from b;
-- footer`,
			want: []stmtKind{kSelect, kSelect},
		},
		{
			name: "comment after clause keyword",
			in:   "select x -- pick x\nfrom a;",
			want: []stmtKind{kSelect},
		},
		{
			name: "semicolon inside string is not a separator",
			in:   `update a/* set msg = "a;b" where vegan = true;`,
			want: []stmtKind{kUpdate},
		},
		{
			name: "dashes inside string are not a comment",
			in:   `update a/* set note = "-- not a comment";`,
			want: []stmtKind{kUpdate},
		},
		{
			name: "backtick identifier with embedded semicolon",
			in:   "select `weird;name` from a;",
			want: []stmtKind{kSelect},
		},

		// Error cases
		{name: "unknown keyword", in: "delete from a;", wantErr: true},
		{name: "garbage between statements", in: "select x from a where vegan junk", wantErr: true},
		{name: "second statement bad", in: "select x from a; bogus", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := lib.ParseProgram(r(tc.in))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseProgram(%q): expected error, got nil", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseProgram(%q): unexpected error: %v", tc.in, err)
			}
			if len(p.Stmts) != len(tc.want) {
				t.Fatalf("got %d stmts, want %d (got=%v)", len(p.Stmts), len(tc.want), p.Stmts)
			}
			for i, want := range tc.want {
				switch want {
				case kSelect:
					if _, ok := p.Stmts[i].(lib.SelectQuery); !ok {
						t.Errorf("stmt %d: got %T, want SelectQuery", i, p.Stmts[i])
					}
				case kUpdate:
					if _, ok := p.Stmts[i].(lib.UpdateQuery); !ok {
						t.Errorf("stmt %d: got %T, want UpdateQuery", i, p.Stmts[i])
					}
				case kAlter:
					if _, ok := p.Stmts[i].(lib.AlterQuery); !ok {
						t.Errorf("stmt %d: got %T, want AlterQuery", i, p.Stmts[i])
					}
				}
			}
		})
	}
}
