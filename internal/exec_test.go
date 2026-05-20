package internal

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

func vInt(n int64) Value   { return Value{Kind: TypeInt, Data: n} }
func vNum(n float64) Value { return Value{Kind: TypeNumber, Data: n} }
func vStr(s string) Value  { return Value{Kind: TypeString, Data: s} }
func vBool(b bool) Value   { return Value{Kind: TypeBool, Data: b} }
func vNull() Value         { return Value{Null: true} }
func vList(els ...Value) Value {
	return Value{Kind: TypeList, Data: els}
}
func vDate(y, m, d int) Value {
	return Value{Kind: TypeDate, Data: time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)}
}
func vDatetime(y, mo, d, h, mn, s int) Value {
	return Value{Kind: TypeDatetime, Data: time.Date(y, time.Month(mo), d, h, mn, s, 0, time.UTC)}
}
func vLink(s string) Value   { return Value{Kind: TypeLink, Data: s} }
func vMdLink(s string) Value { return Value{Kind: TypeMdLink, Data: s} }

func valEq(a, b Value) bool {
	if a.Null != b.Null {
		return false
	}
	if a.Null {
		return true
	}
	if a.Kind != b.Kind {
		return false
	}
	if a.Kind == TypeDate || a.Kind == TypeDatetime {
		return a.Data.(time.Time).Equal(b.Data.(time.Time))
	}
	if a.Kind == TypeList {
		as, bs := a.Data.([]Value), b.Data.([]Value)
		if len(as) != len(bs) {
			return false
		}
		for i := range as {
			if !valEq(as[i], bs[i]) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(a.Data, b.Data)
}

func parseQ(t *testing.T, src string) Query {
	t.Helper()
	q, err := ParseQuery(strings.NewReader(src))
	if err != nil {
		t.Fatalf("ParseQuery(%q): %v", src, err)
	}
	return q
}

// ── Row 2: LitExpr.Eval ───────────────────────────────────────────────────────

func TestLitExpr_Eval(t *testing.T) {
	tests := []struct {
		name string
		lit  LitExpr
		want Value
	}{
		{"int_dec", LitExpr{Kind: LitInt, Value: "42"}, vInt(42)},
		{"int_neg", LitExpr{Kind: LitInt, Value: "-7"}, vInt(-7)},
		{"int_hex", LitExpr{Kind: LitInt, Value: "0xFF"}, vInt(255)},
		{"numeric", LitExpr{Kind: LitNumeric, Value: "3.14"}, vNum(3.14)},
		{"numeric_exp", LitExpr{Kind: LitNumeric, Value: "1.0e2"}, vNum(100)},
		{"string", LitExpr{Kind: LitString, Value: "hello"}, vStr("hello")},
		{"bool_true", LitExpr{Kind: LitBool, Value: "true"}, vBool(true)},
		{"bool_false", LitExpr{Kind: LitBool, Value: "false"}, vBool(false)},
		{"bool_TRUE", LitExpr{Kind: LitBool, Value: "TRUE"}, vBool(true)},
		{"null", LitExpr{Kind: LitNull, Value: "null"}, vNull()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.lit.Eval(nil)
			if !valEq(got, tc.want) {
				t.Errorf("Eval() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// ── Row 3: FieldExpr.Eval ─────────────────────────────────────────────────────

func TestFieldExpr_Eval(t *testing.T) {
	fm := FrontMatter{
		"title":  "hello",
		"count":  int64(5),
		"price":  3.14,
		"active": true,
		"tags":   []any{"a", "b"},
	}
	tests := []struct {
		name string
		f    Field
		want Value
	}{
		{"any_string", Field{Name: "title", Type: TypeAny}, vStr("hello")},
		{"any_int", Field{Name: "count", Type: TypeAny}, vInt(5)},
		{"any_bool", Field{Name: "active", Type: TypeAny}, vBool(true)},
		{"missing", Field{Name: "nope", Type: TypeAny}, vNull()},
		{"typed_match", Field{Name: "count", Type: TypeInt}, vInt(5)},
		{"typed_relax_int_to_string", Field{Name: "count", Type: TypeString}, vStr("5")},
		{"typed_cast_fail_string_to_int", Field{Name: "title", Type: TypeInt}, vNull()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := FieldExpr{Field: tc.f}
			got := e.Eval(fm)
			if !valEq(got, tc.want) {
				t.Errorf("Eval(%+v) = %+v, want %+v", tc.f, got, tc.want)
			}
		})
	}
}

// ── Row 4: UnaryExpr.Eval ─────────────────────────────────────────────────────

func TestUnaryExpr_Eval(t *testing.T) {
	fm := FrontMatter{"n": int64(5)}
	missing := FieldExpr{Field: Field{Name: "missing"}}
	bl := func(s string) LitExpr { return LitExpr{Kind: LitBool, Value: s} }
	il := func(s string) LitExpr { return LitExpr{Kind: LitInt, Value: s} }

	tests := []struct {
		name string
		e    UnaryExpr
		want Value
	}{
		{"not_true", UnaryExpr{Op: UnaryNot, Operand: bl("true")}, vBool(false)},
		{"not_false", UnaryExpr{Op: UnaryNot, Operand: bl("false")}, vBool(true)},
		{"not_null", UnaryExpr{Op: UnaryNot, Operand: missing}, vBool(true)},
		{"neg_int", UnaryExpr{Op: UnaryNeg, Operand: il("5")}, vInt(-5)},
		{"neg_null", UnaryExpr{Op: UnaryNeg, Operand: missing}, vNull()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.e.Eval(fm)
			if !valEq(got, tc.want) {
				t.Errorf("Eval() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// ── Row 5: BinExpr (and/or) ───────────────────────────────────────────────────

func TestBinExpr_BoolOps(t *testing.T) {
	bl := func(s string) LitExpr { return LitExpr{Kind: LitBool, Value: s} }
	missing := FieldExpr{Field: Field{Name: "missing"}}
	fm := FrontMatter{}

	tests := []struct {
		name string
		e    BinExpr
		want Value
	}{
		{"and_tt", BinExpr{Op: BinAnd, Left: bl("true"), Right: bl("true")}, vBool(true)},
		{"and_tf", BinExpr{Op: BinAnd, Left: bl("true"), Right: bl("false")}, vBool(false)},
		{"and_ft", BinExpr{Op: BinAnd, Left: bl("false"), Right: bl("true")}, vBool(false)},
		{"and_null_lhs", BinExpr{Op: BinAnd, Left: missing, Right: bl("true")}, vBool(false)},
		{"and_null_rhs", BinExpr{Op: BinAnd, Left: bl("true"), Right: missing}, vBool(false)},
		{"or_tf", BinExpr{Op: BinOr, Left: bl("true"), Right: bl("false")}, vBool(true)},
		{"or_ff", BinExpr{Op: BinOr, Left: bl("false"), Right: bl("false")}, vBool(false)},
		{"or_null_null", BinExpr{Op: BinOr, Left: missing, Right: missing}, vBool(false)},
		{"or_null_true", BinExpr{Op: BinOr, Left: missing, Right: bl("true")}, vBool(true)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.e.Eval(fm)
			if !valEq(got, tc.want) {
				t.Errorf("Eval() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// ── Row 6: BinExpr (arith) ────────────────────────────────────────────────────

func TestBinExpr_Arith(t *testing.T) {
	il := func(s string) LitExpr { return LitExpr{Kind: LitInt, Value: s} }
	nl := func(s string) LitExpr { return LitExpr{Kind: LitNumeric, Value: s} }
	missing := FieldExpr{Field: Field{Name: "missing"}}
	fm := FrontMatter{}

	tests := []struct {
		name string
		e    BinExpr
		want Value
	}{
		{"add_ints", BinExpr{Op: BinAdd, Left: il("2"), Right: il("3")}, vInt(5)},
		{"sub_ints", BinExpr{Op: BinSub, Left: il("10"), Right: il("4")}, vInt(6)},
		{"mul_ints", BinExpr{Op: BinMul, Left: il("3"), Right: il("4")}, vInt(12)},
		{"div_ints_exact", BinExpr{Op: BinDiv, Left: il("10"), Right: il("2")}, vInt(5)},
		{"add_numerics", BinExpr{Op: BinAdd, Left: nl("1.5"), Right: nl("2.5")}, vNum(4.0)},
		{"mix_int_numeric", BinExpr{Op: BinAdd, Left: il("1"), Right: nl("2.5")}, vNum(3.5)},
		{"add_null_lhs", BinExpr{Op: BinAdd, Left: missing, Right: il("1")}, vNull()},
		{"add_null_rhs", BinExpr{Op: BinAdd, Left: il("1"), Right: missing}, vNull()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.e.Eval(fm)
			if !valEq(got, tc.want) {
				t.Errorf("Eval() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// ── Row 7: BinExpr (comparison) ───────────────────────────────────────────────

func TestBinExpr_Compare(t *testing.T) {
	il := func(s string) LitExpr { return LitExpr{Kind: LitInt, Value: s} }
	sl := func(s string) LitExpr { return LitExpr{Kind: LitString, Value: s} }
	missing := FieldExpr{Field: Field{Name: "missing"}}
	fm := FrontMatter{}

	tests := []struct {
		name string
		e    BinExpr
		want Value
	}{
		{"eq_ints_true", BinExpr{Op: BinEq, Left: il("1"), Right: il("1")}, vBool(true)},
		{"eq_ints_false", BinExpr{Op: BinEq, Left: il("1"), Right: il("2")}, vBool(false)},
		{"ne_ints", BinExpr{Op: BinNe, Left: il("1"), Right: il("2")}, vBool(true)},
		{"lt", BinExpr{Op: BinLt, Left: il("1"), Right: il("2")}, vBool(true)},
		{"le_eq", BinExpr{Op: BinLe, Left: il("2"), Right: il("2")}, vBool(true)},
		{"gt_false", BinExpr{Op: BinGt, Left: il("1"), Right: il("2")}, vBool(false)},
		{"ge_eq", BinExpr{Op: BinGe, Left: il("2"), Right: il("2")}, vBool(true)},
		{"eq_strings", BinExpr{Op: BinEq, Left: sl("a"), Right: sl("a")}, vBool(true)},
		{"eq_null_lhs", BinExpr{Op: BinEq, Left: missing, Right: il("1")}, vBool(false)},
		{"eq_null_rhs", BinExpr{Op: BinEq, Left: il("1"), Right: missing}, vBool(false)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.e.Eval(fm)
			if !valEq(got, tc.want) {
				t.Errorf("Eval() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// ── Row 8: Assign.Apply ───────────────────────────────────────────────────────

func TestAssign_Apply(t *testing.T) {
	t.Run("set_literal", func(t *testing.T) {
		fm := FrontMatter{}
		a := Assign{
			Field: Field{Name: "title"},
			Op:    OpSet,
			Value: LitExpr{Kind: LitString, Value: "hello"},
		}
		if err := a.Apply(fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if fm["title"] != "hello" {
			t.Errorf("title = %v, want hello", fm["title"])
		}
	})
	t.Run("set_typed_cast_ok", func(t *testing.T) {
		fm := FrontMatter{}
		a := Assign{
			Field: Field{Name: "n", Type: TypeInt},
			Op:    OpSet,
			Value: LitExpr{Kind: LitString, Value: "42"},
		}
		if err := a.Apply(fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if fm["n"] != int64(42) {
			t.Errorf("n = %v (%T), want int64(42)", fm["n"], fm["n"])
		}
	})
	t.Run("cast_only_creates_null", func(t *testing.T) {
		fm := FrontMatter{}
		a := Assign{
			Field: Field{Name: "foo", Type: TypeInt},
			Op:    OpSet,
			Value: nil,
		}
		if err := a.Apply(fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if _, ok := fm["foo"]; !ok {
			t.Error("foo not created")
		}
		if fm["foo"] != nil {
			t.Errorf("foo = %v, want nil", fm["foo"])
		}
	})
	t.Run("set_null", func(t *testing.T) {
		fm := FrontMatter{"title": "hello"}
		a := Assign{
			Field: Field{Name: "title"},
			Op:    OpSet,
			Value: LitExpr{Kind: LitNull, Value: "null"},
		}
		if err := a.Apply(fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if fm["title"] != nil {
			t.Errorf("title = %v, want nil", fm["title"])
		}
	})
	t.Run("add_to_list", func(t *testing.T) {
		fm := FrontMatter{"tags": []any{"a"}}
		a := Assign{
			Field: Field{Name: "tags"},
			Op:    OpAdd,
			Value: LitExpr{Kind: LitString, Value: "b"},
		}
		if err := a.Apply(fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		tags, ok := fm["tags"].([]any)
		if !ok || len(tags) != 2 || tags[1] != "b" {
			t.Errorf("tags = %+v, want [a b]", fm["tags"])
		}
	})
	t.Run("sub_from_list", func(t *testing.T) {
		fm := FrontMatter{"tags": []any{"a", "b", "c"}}
		a := Assign{
			Field: Field{Name: "tags"},
			Op:    OpSub,
			Value: LitExpr{Kind: LitString, Value: "b"},
		}
		if err := a.Apply(fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		tags, _ := fm["tags"].([]any)
		if len(tags) != 2 || tags[0] != "a" || tags[1] != "c" {
			t.Errorf("tags = %+v, want [a c]", tags)
		}
	})
	t.Run("add_to_null_field", func(t *testing.T) {
		fm := FrontMatter{"tags": nil}
		a := Assign{
			Field: Field{Name: "tags"},
			Op:    OpAdd,
			Value: LitExpr{Kind: LitString, Value: "x"},
		}
		if err := a.Apply(fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		tags, ok := fm["tags"].([]any)
		if !ok || len(tags) != 1 || tags[0] != "x" {
			t.Errorf("tags = %+v, want [x]", fm["tags"])
		}
	})
	t.Run("sub_from_null_field", func(t *testing.T) {
		fm := FrontMatter{"tags": nil}
		a := Assign{
			Field: Field{Name: "tags"},
			Op:    OpSub,
			Value: LitExpr{Kind: LitString, Value: "x"},
		}
		if err := a.Apply(fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if fm["tags"] != nil {
			t.Errorf("tags = %+v, want nil", fm["tags"])
		}
	})
	t.Run("cast_failure_errors", func(t *testing.T) {
		fm := FrontMatter{"src": "hello"}
		a := Assign{
			Field: Field{Name: "n", Type: TypeInt},
			Op:    OpSet,
			Value: FieldExpr{Field: Field{Name: "src", Type: TypeString}},
		}
		if err := a.Apply(fm); err == nil {
			t.Error("expected cast-failure error, got nil")
		}
	})
}

// ── Row 9: SortTerm.Eval ──────────────────────────────────────────────────────

func TestSortTerm_Eval(t *testing.T) {
	fm := FrontMatter{"title": "abc"}
	s := SortTerm{
		Expr: FieldExpr{Field: Field{Name: "title"}},
		Desc: false,
	}
	if got := s.Eval(fm); !valEq(got, vStr("abc")) {
		t.Errorf("Eval() = %+v, want %+v", got, vStr("abc"))
	}
}

// ── Row 10: AlterQuery.Apply ──────────────────────────────────────────────────

func TestAlterQuery_Apply(t *testing.T) {
	t.Run("drop_one", func(t *testing.T) {
		fm := FrontMatter{"title": "x", "date": "2026-01-01"}
		q := parseQ(t, "alter *.md drop title").(AlterQuery)
		if err := q.Apply(fm); err != nil {
			t.Fatal(err)
		}
		if _, ok := fm["title"]; ok {
			t.Error("title still present")
		}
		if _, ok := fm["date"]; !ok {
			t.Error("date should remain")
		}
	})
	t.Run("drop_multi", func(t *testing.T) {
		fm := FrontMatter{"a": int64(1), "b": int64(2), "c": int64(3)}
		q := parseQ(t, "alter *.md drop a, b").(AlterQuery)
		if err := q.Apply(fm); err != nil {
			t.Fatal(err)
		}
		if len(fm) != 1 || fm["c"] != int64(3) {
			t.Errorf("FrontMatter = %+v, want {c:3}", fm)
		}
	})
	t.Run("rename", func(t *testing.T) {
		fm := FrontMatter{"foo": "hello"}
		q := parseQ(t, "alter *.md rename foo to bar").(AlterQuery)
		if err := q.Apply(fm); err != nil {
			t.Fatal(err)
		}
		if _, ok := fm["foo"]; ok {
			t.Error("foo still present")
		}
		if fm["bar"] != "hello" {
			t.Errorf("bar = %v, want hello", fm["bar"])
		}
	})
	t.Run("where_skips", func(t *testing.T) {
		fm := FrontMatter{"published": false, "title": "x"}
		q := parseQ(t, "alter *.md drop title where published = true").(AlterQuery)
		if err := q.Apply(fm); err != nil {
			t.Fatal(err)
		}
		if _, ok := fm["title"]; !ok {
			t.Error("title should remain (where=false)")
		}
	})
}

// ── Row 11: UpdateQuery.Apply ─────────────────────────────────────────────────

func TestUpdateQuery_Apply(t *testing.T) {
	t.Run("set_one", func(t *testing.T) {
		fm := FrontMatter{}
		q := parseQ(t, `update *.md set title = "hello"`).(UpdateQuery)
		if err := q.Apply(fm); err != nil {
			t.Fatal(err)
		}
		if fm["title"] != "hello" {
			t.Errorf("title = %v", fm["title"])
		}
	})
	t.Run("set_multi", func(t *testing.T) {
		fm := FrontMatter{}
		q := parseQ(t, `update *.md set a = 1, b = 2`).(UpdateQuery)
		if err := q.Apply(fm); err != nil {
			t.Fatal(err)
		}
		if fm["a"] != int64(1) || fm["b"] != int64(2) {
			t.Errorf("FrontMatter = %+v", fm)
		}
	})
	t.Run("where_skips", func(t *testing.T) {
		fm := FrontMatter{"published": false}
		q := parseQ(t, `update *.md set title = "x" where published = true`).(UpdateQuery)
		if err := q.Apply(fm); err != nil {
			t.Fatal(err)
		}
		if _, ok := fm["title"]; ok {
			t.Error("title set despite where=false")
		}
	})
}

// ── Row 12: SelectQuery.Eval ──────────────────────────────────────────────────

func TestSelectQuery_Eval(t *testing.T) {
	t.Run("project_one", func(t *testing.T) {
		fm := FrontMatter{"title": "hello", "date": "2026-01-01"}
		q := parseQ(t, "select title from *.md").(SelectQuery)
		row, err := q.Eval(fm)
		if err != nil {
			t.Fatal(err)
		}
		if len(row) != 1 || !valEq(row[0], vStr("hello")) {
			t.Errorf("row = %+v, want [hello]", row)
		}
	})
	t.Run("project_multi", func(t *testing.T) {
		fm := FrontMatter{"a": "x", "b": int64(42)}
		q := parseQ(t, "select a, b from *.md").(SelectQuery)
		row, err := q.Eval(fm)
		if err != nil {
			t.Fatal(err)
		}
		if len(row) != 2 {
			t.Fatalf("row = %+v, want 2 cols", row)
		}
		if !valEq(row[0], vStr("x")) {
			t.Errorf("row[0] = %+v, want %+v", row[0], vStr("x"))
		}
		if !valEq(row[1], vInt(42)) {
			t.Errorf("row[1] = %+v, want %+v", row[1], vInt(42))
		}
	})
	t.Run("where_returns_nil", func(t *testing.T) {
		fm := FrontMatter{"published": false, "title": "x"}
		q := parseQ(t, "select title from *.md where published = true").(SelectQuery)
		row, err := q.Eval(fm)
		if err != nil {
			t.Fatal(err)
		}
		if row != nil {
			t.Errorf("row = %+v, want nil (where=false)", row)
		}
	})
	t.Run("missing_field_null", func(t *testing.T) {
		fm := FrontMatter{}
		q := parseQ(t, "select title from *.md").(SelectQuery)
		row, err := q.Eval(fm)
		if err != nil {
			t.Fatal(err)
		}
		if len(row) != 1 || !row[0].Null {
			t.Errorf("row = %+v, want [null]", row)
		}
	})
}

// ── Date/datetime types ───────────────────────────────────────────────────────

func TestFieldExpr_Eval_Date(t *testing.T) {
	fm := FrontMatter{
		"created":  time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC),
		"modified": time.Date(2026, 5, 14, 21, 2, 30, 0, time.UTC),
	}
	tests := []struct {
		name string
		f    Field
		want Value
	}{
		{"time_as_date", Field{Name: "created", Type: TypeDate}, vDate(2026, 5, 14)},
		{"time_as_datetime", Field{Name: "modified", Type: TypeDatetime}, vDatetime(2026, 5, 14, 21, 2, 30)},
		{"time_any_detects_date", Field{Name: "created", Type: TypeAny}, vDate(2026, 5, 14)},
		{"time_any_detects_datetime", Field{Name: "modified", Type: TypeAny}, vDatetime(2026, 5, 14, 21, 2, 30)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := FieldExpr{Field: tc.f}
			got := e.Eval(fm)
			if !valEq(got, tc.want) {
				t.Errorf("Eval(%+v) = %+v, want %+v", tc.f, got, tc.want)
			}
		})
	}
}

// ── Link/mdlink types ─────────────────────────────────────────────────────────

func TestFieldExpr_Eval_Link(t *testing.T) {
	fm := FrontMatter{
		"source": "[[ref|title]]",
		"url":    "[title](ref)",
	}
	tests := []struct {
		name string
		f    Field
		want Value
	}{
		{"string_as_link", Field{Name: "source", Type: TypeLink}, vLink("[[ref|title]]")},
		{"string_as_mdlink", Field{Name: "url", Type: TypeMdLink}, vMdLink("[title](ref)")},
		{"link_to_mdlink", Field{Name: "source", Type: TypeMdLink}, vMdLink("[title](ref)")},
		{"mdlink_to_link_via_field", Field{Name: "url", Type: TypeLink}, vLink("[[ref|title]]")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := FieldExpr{Field: tc.f}
			got := e.Eval(fm)
			if !valEq(got, tc.want) {
				t.Errorf("Eval(%+v) = %+v, want %+v", tc.f, got, tc.want)
			}
		})
	}
}

// ── List cast ─────────────────────────────────────────────────────────────────

// Casting to TypeList always normalises element values to TypeString so the
// list is uniformly list-of-string; scalars are wrapped and stringified.
func TestCastToList_StringElements(t *testing.T) {
	tests := []struct {
		name string
		v    Value
		want Value
	}{
		{"string_passthrough", vList(vStr("a"), vStr("b")), vList(vStr("a"), vStr("b"))},
		{"int_elements_stringified", vList(vInt(3), vInt(4)), vList(vStr("3"), vStr("4"))},
		{"mixed_stringified", vList(vInt(1), vStr("two"), vBool(true)), vList(vStr("1"), vStr("two"), vStr("true"))},
		{"scalar_wrapped", vInt(5), vList(vStr("5"))},
		{"scalar_string", vStr("hi"), vList(vStr("hi"))},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Cast(tc.v, TypeList)
			if err != nil {
				t.Fatalf("Cast(%+v, list) error: %s", tc.v, err)
			}
			if !valEq(got, tc.want) {
				t.Errorf("Cast(%+v, list) = %+v, want %+v", tc.v, got, tc.want)
			}
		})
	}
}

func TestFieldExpr_Eval_List(t *testing.T) {
	fm := FrontMatter{"nums": []any{int64(3), int64(4), int64(5)}}
	e := FieldExpr{Field: Field{Name: "nums", Type: TypeList}}
	got := e.Eval(fm)
	want := vList(vStr("3"), vStr("4"), vStr("5"))
	if !valEq(got, want) {
		t.Errorf("Eval() = %+v, want %+v", got, want)
	}
}
