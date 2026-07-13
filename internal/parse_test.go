package internal

import (
	"strings"
	"testing"
)

// r wraps a string in a Reader for use with Parse methods.
func r(s string) *strings.Reader { return strings.NewReader(s) }

// ── Field ─────────────────────────────────────────────────────────────────────

func TestField_Parse(t *testing.T) {
	tests := []struct {
		in      string
		want    Field
		wantErr bool
	}{
		// Untyped identifier
		{in: "title", want: Field{Name: "title", Type: TypeAny}},
		{in: "_private", want: Field{Name: "_private", Type: TypeAny}},
		{in: "field_123", want: Field{Name: "field_123", Type: TypeAny}},

		// Typed identifiers
		{in: "title:string", want: Field{Name: "title", Type: TypeString}},
		{in: "active:bool", want: Field{Name: "active", Type: TypeBool}},
		{in: "count:int", want: Field{Name: "count", Type: TypeInt}},
		{in: "price:numeric", want: Field{Name: "price", Type: TypeNumber}},
		{in: "created:date", want: Field{Name: "created", Type: TypeDate}},
		{in: "modified:datetime", want: Field{Name: "modified", Type: TypeDatetime}},
		{in: "ref:link", want: Field{Name: "ref", Type: TypeLink}},
		{in: "ref:mdlink", want: Field{Name: "ref", Type: TypeMdLink}},

		// List type (list-of-string; no element-type annotation)
		{in: "tags:list", want: Field{Name: "tags", Type: TypeList}},

		// Quoted identifiers (backtick) — allow spaces, symbols, reserved words
		{in: "`created-at`", want: Field{Name: "created-at", Type: TypeAny}},
		{in: "`field with spaces`", want: Field{Name: "field with spaces", Type: TypeAny}},
		{in: "`from`", want: Field{Name: "from", Type: TypeAny}}, // reserved keyword
		{in: "`from`:date", want: Field{Name: "from", Type: TypeDate}},

		// Functions
		{in: `length("abc")`},
		{in: `length("abc", 123)`}, // wrong argument count
		{in: `coalesce("abc")`},
		{in: `coalesce("abc", 123, true)`},     // variadic
		{in: `bad_func("abc")`, wantErr: true}, // unknown function

		// Errors
		{in: "", wantErr: true},                 // empty
		{in: "1foo", wantErr: true},             // starts with digit
		{in: "foo:unknown", wantErr: true},      // unknown type
		{in: "``", wantErr: true},               // empty quoted identifier
		{in: "tags:list:string", wantErr: true}, // list element type no longer accepted
		{in: "tags:list:int", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			var f Field
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
		})
	}
}

// ── LitExpr ───────────────────────────────────────────────────────────────────

func TestLitExpr_Parse(t *testing.T) {
	tests := []struct {
		in      string
		want    LitExpr
		wantErr bool
	}{
		// Integer literals
		{in: "42", want: LitExpr{Kind: LitInt, Value: "42"}},
		{in: "-7", want: LitExpr{Kind: LitInt, Value: "-7"}},
		{in: "0", want: LitExpr{Kind: LitInt, Value: "0"}},
		{in: "0xFF", want: LitExpr{Kind: LitInt, Value: "0xFF"}},
		{in: "0x1A2B", want: LitExpr{Kind: LitInt, Value: "0x1A2B"}},

		// Numeric (float) literals
		{in: "3.14", want: LitExpr{Kind: LitNumeric, Value: "3.14"}},
		{in: "-2.5", want: LitExpr{Kind: LitNumeric, Value: "-2.5"}},
		{in: "1.0e10", want: LitExpr{Kind: LitNumeric, Value: "1.0e10"}},
		{in: "6.022E-23", want: LitExpr{Kind: LitNumeric, Value: "6.022E-23"}},
		{in: ".5", want: LitExpr{Kind: LitNumeric, Value: ".5"}},

		// Boolean literals (case insensitive per spec)
		{in: "true", want: LitExpr{Kind: LitBool, Value: "true"}},
		{in: "false", want: LitExpr{Kind: LitBool, Value: "false"}},
		{in: "TRUE", want: LitExpr{Kind: LitBool, Value: "TRUE"}},
		{in: "False", want: LitExpr{Kind: LitBool, Value: "False"}},

		// Null literal (case insensitive per spec)
		{in: "null", want: LitExpr{Kind: LitNull, Value: "null"}},
		{in: "NULL", want: LitExpr{Kind: LitNull, Value: "NULL"}},

		// Double-quoted strings
		{in: `"hello"`, want: LitExpr{Kind: LitString, Value: "hello"}},
		{in: `"hello world"`, want: LitExpr{Kind: LitString, Value: "hello world"}},
		{in: `""`, want: LitExpr{Kind: LitString, Value: ""}},
		{in: `"it's a \"test\""`, want: LitExpr{Kind: LitString, Value: `it's a "test"`}},

		// Single-quoted strings
		{in: `'hello'`, want: LitExpr{Kind: LitString, Value: "hello"}},
		{in: `'it\'s fine'`, want: LitExpr{Kind: LitString, Value: "it's fine"}},

		// Triple-quoted strings (multi-line)
		{in: `"""hello"""`, want: LitExpr{Kind: LitString, Value: "hello"}},
		{in: "'''line1\nline2'''", want: LitExpr{Kind: LitString, Value: "line1\nline2"}},

		// Raw strings (backslash not interpreted)
		{in: `r"C:\Users\name"`, want: LitExpr{Kind: LitString, Value: `C:\Users\name`}},
		{in: `r'no\escape'`, want: LitExpr{Kind: LitString, Value: `no\escape`}},

		// Escape sequences in regular strings
		{in: `"\n"`, want: LitExpr{Kind: LitString, Value: "\n"}},
		{in: `"\t"`, want: LitExpr{Kind: LitString, Value: "\t"}},
		{in: `"\\"`, want: LitExpr{Kind: LitString, Value: `\`}},

		// Errors
		{in: "", wantErr: true},              // empty
		{in: `"unterminated`, wantErr: true}, // missing closing quote
		{in: `'unterminated`, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			var lit LitExpr
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
		check func(t *testing.T, e Expr)
	}{
		// Atoms
		{
			in: "title",
			check: func(t *testing.T, e Expr) {
				fe, ok := e.(FieldExpr)
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
			check: func(t *testing.T, e Expr) {
				lit, ok := e.(LitExpr)
				if !ok {
					t.Fatalf("got %T, want LitExpr", e)
				}
				if lit.Kind != LitInt {
					t.Errorf("Kind = %v, want LitInt", lit.Kind)
				}
			},
		},

		// Comparison
		{
			in: "price:numeric > 10",
			check: func(t *testing.T, e Expr) {
				bin, ok := e.(BinExpr)
				if !ok {
					t.Fatalf("got %T, want BinExpr", e)
				}
				if bin.Op != BinGt {
					t.Errorf("Op = %v, want BinGt", bin.Op)
				}
				lhs, ok := bin.Left.(FieldExpr)
				if !ok {
					t.Fatalf("Left: got %T, want FieldExpr", bin.Left)
				}
				if lhs.Field.Name != "price" || lhs.Field.Type != TypeNumber {
					t.Errorf("Left = %+v, want {price:numeric}", lhs.Field)
				}
				rhs, ok := bin.Right.(LitExpr)
				if !ok {
					t.Fatalf("Right: got %T, want LitExpr", bin.Right)
				}
				if rhs.Kind != LitInt || rhs.Value != "10" {
					t.Errorf("Right = %+v, want LitInt 10", rhs)
				}
			},
		},
		{
			in: `url:string = "https://example.com"`,
			check: func(t *testing.T, e Expr) {
				bin, ok := e.(BinExpr)
				if !ok {
					t.Fatalf("got %T, want BinExpr", e)
				}
				if bin.Op != BinEq {
					t.Errorf("Op = %v, want BinEq", bin.Op)
				}
			},
		},

		// Unary not
		{
			in: "not active",
			check: func(t *testing.T, e Expr) {
				un, ok := e.(UnaryExpr)
				if !ok {
					t.Fatalf("got %T, want UnaryExpr", e)
				}
				if un.Op != UnaryNot {
					t.Errorf("Op = %v, want UnaryNot", un.Op)
				}
			},
		},

		// and binds tighter than or: "a and b or c" == "(a and b) or c"
		{
			in: "a and b or c",
			check: func(t *testing.T, e Expr) {
				top, ok := e.(BinExpr)
				if !ok {
					t.Fatalf("got %T, want BinExpr", e)
				}
				if top.Op != BinOr {
					t.Errorf("top Op = %v, want BinOr", top.Op)
				}
				left, ok := top.Left.(BinExpr)
				if !ok {
					t.Fatalf("Left: got %T, want BinExpr", top.Left)
				}
				if left.Op != BinAnd {
					t.Errorf("left Op = %v, want BinAnd", left.Op)
				}
			},
		},

		// Parentheses override precedence: "(a or b) and c"
		{
			in: "(a or b) and c",
			check: func(t *testing.T, e Expr) {
				top, ok := e.(BinExpr)
				if !ok {
					t.Fatalf("got %T, want BinExpr", e)
				}
				if top.Op != BinAnd {
					t.Errorf("top Op = %v, want BinAnd", top.Op)
				}
				left, ok := top.Left.(BinExpr)
				if !ok {
					t.Fatalf("Left: got %T, want BinExpr", top.Left)
				}
				if left.Op != BinOr {
					t.Errorf("left Op = %v, want BinOr", left.Op)
				}
			},
		},

		// Arithmetic: * binds tighter than +
		{
			in: "a + b * c",
			check: func(t *testing.T, e Expr) {
				top, ok := e.(BinExpr)
				if !ok {
					t.Fatalf("got %T, want BinExpr", e)
				}
				if top.Op != BinAdd {
					t.Errorf("top Op = %v, want BinAdd", top.Op)
				}
				right, ok := top.Right.(BinExpr)
				if !ok {
					t.Fatalf("Right: got %T, want BinExpr", top.Right)
				}
				if right.Op != BinMul {
					t.Errorf("right Op = %v, want BinMul", right.Op)
				}
			},
		},

		// Unary arithmetic negation
		{
			in: "-price",
			check: func(t *testing.T, e Expr) {
				un, ok := e.(UnaryExpr)
				if !ok {
					t.Fatalf("got %T, want UnaryExpr", e)
				}
				if un.Op != UnaryNeg {
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
			e, err := ParseExpr(r(tc.in))
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
		check   func(t *testing.T, a Assign)
	}{
		// Cast-only form: no operator, just field with type
		{
			in: "foo:int",
			check: func(t *testing.T, a Assign) {
				if a.Field.Name != "foo" || a.Field.Type != TypeInt {
					t.Errorf("Field = %+v, want {foo:int}", a.Field)
				}
				if a.Op != OpSet {
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
			check: func(t *testing.T, a Assign) {
				if a.Field.Name != "title" {
					t.Errorf("Field.Name = %q, want title", a.Field.Name)
				}
				if a.Op != OpSet {
					t.Errorf("Op = %v, want OpSet", a.Op)
				}
				lit, ok := a.Value.(LitExpr)
				if !ok {
					t.Fatalf("Value: got %T, want LitExpr", a.Value)
				}
				if lit.Kind != LitString || lit.Value != "hello" {
					t.Errorf("Value = %+v, want LitString hello", lit)
				}
			},
		},
		// Set null
		{
			in: "title = null",
			check: func(t *testing.T, a Assign) {
				if a.Op != OpSet {
					t.Errorf("Op = %v, want OpSet", a.Op)
				}
				lit, ok := a.Value.(LitExpr)
				if !ok {
					t.Fatalf("Value: got %T, want LitExpr", a.Value)
				}
				if lit.Kind != LitNull {
					t.Errorf("Kind = %v, want LitNull", lit.Kind)
				}
			},
		},
		// Set from field reference
		{
			in: "title = source",
			check: func(t *testing.T, a Assign) {
				_, ok := a.Value.(FieldExpr)
				if !ok {
					t.Fatalf("Value: got %T, want FieldExpr", a.Value)
				}
			},
		},
		// List addition
		{
			in: `tags += "recipe"`,
			check: func(t *testing.T, a Assign) {
				if a.Op != OpAdd {
					t.Errorf("Op = %v, want OpAdd", a.Op)
				}
			},
		},
		// List subtraction
		{
			in: `tags -= "recipe"`,
			check: func(t *testing.T, a Assign) {
				if a.Op != OpSub {
					t.Errorf("Op = %v, want OpSub", a.Op)
				}
			},
		},
		// Numeric increment via field ref
		{
			in: "count += 1",
			check: func(t *testing.T, a Assign) {
				if a.Op != OpAdd {
					t.Errorf("Op = %v, want OpAdd", a.Op)
				}
				lit, ok := a.Value.(LitExpr)
				if !ok {
					t.Fatalf("Value: got %T, want LitExpr", a.Value)
				}
				if lit.Kind != LitInt {
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
			var a Assign
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
		check   func(t *testing.T, s SortTerm)
	}{
		{
			in: "title",
			check: func(t *testing.T, s SortTerm) {
				if s.Desc {
					t.Error("Desc = true, want false (default is asc)")
				}
				fe, ok := s.Expr.(FieldExpr)
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
			check: func(t *testing.T, s SortTerm) {
				if !s.Desc {
					t.Error("Desc = false, want true")
				}
			},
		},
		{
			in: "price:numeric asc",
			check: func(t *testing.T, s SortTerm) {
				if s.Desc {
					t.Error("Desc = true, want false")
				}
				fe, ok := s.Expr.(FieldExpr)
				if !ok {
					t.Fatalf("Expr: got %T, want FieldExpr", s.Expr)
				}
				if fe.Field.Type != TypeNumber {
					t.Errorf("Type = %v, want TypeNumber", fe.Field.Type)
				}
			},
		},
		{in: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			var s SortTerm
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
		want    RenamePair
		wantErr bool
	}{
		{in: "foo to bar", want: RenamePair{From: "foo", To: "bar"}},
		{in: "old_name to new_name", want: RenamePair{From: "old_name", To: "new_name"}},
		{in: "`created-at` to created_at", want: RenamePair{From: "created-at", To: "created_at"}},
		// Errors
		{in: "foo", wantErr: true},    // missing "to"
		{in: "foo to", wantErr: true}, // missing target
		{in: "to bar", wantErr: true}, // missing source
		{in: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			var p RenamePair
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
		check   func(t *testing.T, q SelectQuery)
	}{
		{
			in: "select title from *.md",
			check: func(t *testing.T, q SelectQuery) {
				if len(q.Select) != 1 {
					t.Fatalf("len(Fields) = %d, want 1", len(q.Select))
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
			check: func(t *testing.T, q SelectQuery) {
				if len(q.Select) != 2 {
					t.Errorf("len(Fields) = %d, want 2", len(q.Select))
				}
			},
		},
		{
			in: "select title from *.md where published = true",
			check: func(t *testing.T, q SelectQuery) {
				if q.Where == nil {
					t.Fatal("Where = nil, want expression")
				}
			},
		},
		{
			in: "select title from *.md sort by date desc",
			check: func(t *testing.T, q SelectQuery) {
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
			check: func(t *testing.T, q SelectQuery) {
				if len(q.SortBy) != 2 {
					t.Fatalf("len(SortBy) = %d, want 2", len(q.SortBy))
				}
			},
		},
		{
			in: "select title from *.md limit 5",
			check: func(t *testing.T, q SelectQuery) {
				if q.Limit != 5 {
					t.Errorf("Limit = %d, want 5", q.Limit)
				}
			},
		},
		{
			in: "select title from *.md sort by date desc limit 10",
			check: func(t *testing.T, q SelectQuery) {
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
			check: func(t *testing.T, q SelectQuery) {
				if len(q.From) != 2 {
					t.Errorf("From = %v, want 2 globs", q.From)
				}
			},
		},
		// Zero fields (just filename)
		{
			in: "select from *.md",
			check: func(t *testing.T, q SelectQuery) {
				if len(q.Select) != 0 {
					t.Errorf("len(Fields) = %d, want 0", len(q.Select))
				}
				if len(q.From) != 1 || q.From[0] != "*.md" {
					t.Errorf("From = %v, want [*.md]", q.From)
				}
			},
		},
		// Errors
		{in: "select title", wantErr: true},                     // missing from clause
		{in: "select title from", wantErr: true},                // missing glob
		{in: "select title from *.md limit -1", wantErr: true},  // negative limit
		{in: "select title from *.md limit foo", wantErr: true}, // non-integer limit
		{in: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			var q SelectQuery
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
		check   func(t *testing.T, q UpdateQuery)
	}{
		{
			in: `update *.md set title = "hello"`,
			check: func(t *testing.T, q UpdateQuery) {
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
			check: func(t *testing.T, q UpdateQuery) {
				if len(q.Set) != 2 {
					t.Errorf("len(Set) = %d, want 2", len(q.Set))
				}
			},
		},
		{
			in: "update *.md set foo:int where active",
			check: func(t *testing.T, q UpdateQuery) {
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
			var q UpdateQuery
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
		check   func(t *testing.T, q AlterQuery)
	}{
		{
			in: "alter *.md drop title",
			check: func(t *testing.T, q AlterQuery) {
				if q.Op != AlterDrop {
					t.Errorf("Op = %v, want AlterDrop", q.Op)
				}
				if len(q.Drop) != 1 || q.Drop[0].Name != "title" {
					t.Errorf("Drop = %v, want [title]", q.Drop)
				}
			},
		},
		{
			in: "alter *.md drop title, date",
			check: func(t *testing.T, q AlterQuery) {
				if len(q.Drop) != 2 {
					t.Errorf("len(Drop) = %d, want 2", len(q.Drop))
				}
			},
		},
		{
			in: "alter *.md rename foo to bar",
			check: func(t *testing.T, q AlterQuery) {
				if q.Op != AlterRename {
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
			check: func(t *testing.T, q AlterQuery) {
				if len(q.Rename) != 2 {
					t.Errorf("len(Rename) = %d, want 2", len(q.Rename))
				}
			},
		},
		{
			in: "alter *.md drop title where published = false",
			check: func(t *testing.T, q AlterQuery) {
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
			var q AlterQuery
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
			q, err := ParseQuery(r(tc.in))
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
				if _, ok := q.(SelectQuery); !ok {
					t.Errorf("got %T, want SelectQuery", q)
				}
			case "update":
				if _, ok := q.(UpdateQuery); !ok {
					t.Errorf("got %T, want UpdateQuery", q)
				}
			case "alter":
				if _, ok := q.(AlterQuery); !ok {
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
			p, err := ParseProgram(r(tc.in))
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
					if _, ok := p.Stmts[i].(SelectQuery); !ok {
						t.Errorf("stmt %d: got %T, want SelectQuery", i, p.Stmts[i])
					}
				case kUpdate:
					if _, ok := p.Stmts[i].(UpdateQuery); !ok {
						t.Errorf("stmt %d: got %T, want UpdateQuery", i, p.Stmts[i])
					}
				case kAlter:
					if _, ok := p.Stmts[i].(AlterQuery); !ok {
						t.Errorf("stmt %d: got %T, want AlterQuery", i, p.Stmts[i])
					}
				}
			}
		})
	}
}

// ── FuncExpr parse ────────────────────────────────────────────────────────────

func TestFuncExpr_Parse(t *testing.T) {
	tests := []struct {
		in      string
		name    string
		nArgs   int
		wantErr bool
	}{
		{in: `LOWER("hello")`, name: "lower", nArgs: 1},
		{in: `lower("hello")`, name: "lower", nArgs: 1},
		{in: `TODAY()`, name: "today", nArgs: 0},
		{in: `CONCAT("a", "b", "c")`, name: "concat", nArgs: 3},
		{in: `SUBSTR(title, 1, 5)`, name: "substr", nArgs: 3},
		{in: `ARRAY_LENGTH(tags)`, name: "array_length", nArgs: 1},
		{in: `COALESCE(a, b, null)`, name: "coalesce", nArgs: 3},
		{in: `UPPER(LOWER("X"))`, name: "upper", nArgs: 1},
		{in: `f(`, wantErr: true},
		{in: `f(1,`, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			e, err := ParseExpr(r(tc.in))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseExpr(%q): expected error, got nil", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseExpr(%q): unexpected error: %v", tc.in, err)
			}
			fn, ok := e.(FuncExpr)
			if !ok {
				t.Fatalf("ParseExpr(%q): got %T, want FuncExpr", tc.in, e)
			}
			if fn.Name != tc.name {
				t.Errorf("Name = %q, want %q", fn.Name, tc.name)
			}
			if len(fn.Args) != tc.nArgs {
				t.Errorf("len(Args) = %d, want %d", len(fn.Args), tc.nArgs)
			}
		})
	}
}

// ── New operator parse ────────────────────────────────────────────────────────

func TestNewOperators_Parse(t *testing.T) {
	tests := []struct {
		in string
		op BinOp
	}{
		{in: `a <=> b`, op: BinIntersect},
		{in: `a >=< b`, op: BinUnion},
		{in: `title LIKE "%foo%"`, op: BinLike},
		{in: `title NOT LIKE "%foo%"`, op: BinNotLike},
		{in: `title ILIKE "%foo%"`, op: BinILike},
		{in: `title NOT ILIKE "%foo%"`, op: BinNotILike},
		{in: `title REGEXP r"^\d+"`, op: BinRegexp},
		{in: `title NOT REGEXP r"^\d+"`, op: BinNotRegexp},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			e, err := ParseExpr(r(tc.in))
			if err != nil {
				t.Fatalf("ParseExpr(%q): unexpected error: %v", tc.in, err)
			}
			bin, ok := e.(BinExpr)
			if !ok {
				t.Fatalf("ParseExpr(%q): got %T, want BinExpr", tc.in, e)
			}
			if bin.Op != tc.op {
				t.Errorf("Op = %v, want %v", bin.Op, tc.op)
			}
		})
	}
}
