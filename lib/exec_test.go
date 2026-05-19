package lib_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/backlin/frontmatter/lib"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

func vInt(n int64) lib.Value   { return lib.Value{Kind: lib.TypeInt, Data: n} }
func vNum(n float64) lib.Value { return lib.Value{Kind: lib.TypeNumber, Data: n} }
func vStr(s string) lib.Value  { return lib.Value{Kind: lib.TypeString, Data: s} }
func vBool(b bool) lib.Value   { return lib.Value{Kind: lib.TypeBool, Data: b} }
func vNull() lib.Value         { return lib.Value{Null: true} }
func vList(els ...lib.Value) lib.Value {
	return lib.Value{Kind: lib.TypeList, Data: els}
}
func vDate(y, m, d int) lib.Value {
	return lib.Value{Kind: lib.TypeDate, Data: time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)}
}
func vDatetime(y, mo, d, h, mn, s int) lib.Value {
	return lib.Value{Kind: lib.TypeDatetime, Data: time.Date(y, time.Month(mo), d, h, mn, s, 0, time.UTC)}
}
func vLink(s string) lib.Value   { return lib.Value{Kind: lib.TypeLink, Data: s} }
func vMdLink(s string) lib.Value { return lib.Value{Kind: lib.TypeMdLink, Data: s} }

func valEq(a, b lib.Value) bool {
	if a.Null != b.Null {
		return false
	}
	if a.Null {
		return true
	}
	if a.Kind != b.Kind {
		return false
	}
	if a.Kind == lib.TypeDate || a.Kind == lib.TypeDatetime {
		return a.Data.(time.Time).Equal(b.Data.(time.Time))
	}
	if a.Kind == lib.TypeList {
		as, bs := a.Data.([]lib.Value), b.Data.([]lib.Value)
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

func parseQ(t *testing.T, src string) lib.Query {
	t.Helper()
	q, err := lib.ParseQuery(strings.NewReader(src))
	if err != nil {
		t.Fatalf("ParseQuery(%q): %v", src, err)
	}
	return q
}

// ── Row 2: LitExpr.Eval ───────────────────────────────────────────────────────

func TestLitExpr_Eval(t *testing.T) {
	tests := []struct {
		name string
		lit  lib.LitExpr
		want lib.Value
	}{
		{"int_dec", lib.LitExpr{Kind: lib.LitInt, Value: "42"}, vInt(42)},
		{"int_neg", lib.LitExpr{Kind: lib.LitInt, Value: "-7"}, vInt(-7)},
		{"int_hex", lib.LitExpr{Kind: lib.LitInt, Value: "0xFF"}, vInt(255)},
		{"numeric", lib.LitExpr{Kind: lib.LitNumeric, Value: "3.14"}, vNum(3.14)},
		{"numeric_exp", lib.LitExpr{Kind: lib.LitNumeric, Value: "1.0e2"}, vNum(100)},
		{"string", lib.LitExpr{Kind: lib.LitString, Value: "hello"}, vStr("hello")},
		{"bool_true", lib.LitExpr{Kind: lib.LitBool, Value: "true"}, vBool(true)},
		{"bool_false", lib.LitExpr{Kind: lib.LitBool, Value: "false"}, vBool(false)},
		{"bool_TRUE", lib.LitExpr{Kind: lib.LitBool, Value: "TRUE"}, vBool(true)},
		{"null", lib.LitExpr{Kind: lib.LitNull, Value: "null"}, vNull()},
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
	fm := lib.FrontMatter{
		"title":  "hello",
		"count":  int64(5),
		"price":  3.14,
		"active": true,
		"tags":   []any{"a", "b"},
	}
	tests := []struct {
		name string
		f    lib.Field
		want lib.Value
	}{
		{"any_string", lib.Field{Name: "title", Type: lib.TypeAny}, vStr("hello")},
		{"any_int", lib.Field{Name: "count", Type: lib.TypeAny}, vInt(5)},
		{"any_bool", lib.Field{Name: "active", Type: lib.TypeAny}, vBool(true)},
		{"missing", lib.Field{Name: "nope", Type: lib.TypeAny}, vNull()},
		{"typed_match", lib.Field{Name: "count", Type: lib.TypeInt}, vInt(5)},
		{"typed_relax_int_to_string", lib.Field{Name: "count", Type: lib.TypeString}, vStr("5")},
		{"typed_cast_fail_string_to_int", lib.Field{Name: "title", Type: lib.TypeInt}, vNull()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := lib.FieldExpr{Field: tc.f}
			got := e.Eval(&fm)
			if !valEq(got, tc.want) {
				t.Errorf("Eval(%+v) = %+v, want %+v", tc.f, got, tc.want)
			}
		})
	}
}

// ── Row 4: UnaryExpr.Eval ─────────────────────────────────────────────────────

func TestUnaryExpr_Eval(t *testing.T) {
	fm := lib.FrontMatter{"n": int64(5)}
	missing := lib.FieldExpr{Field: lib.Field{Name: "missing"}}
	bl := func(s string) lib.LitExpr { return lib.LitExpr{Kind: lib.LitBool, Value: s} }
	il := func(s string) lib.LitExpr { return lib.LitExpr{Kind: lib.LitInt, Value: s} }

	tests := []struct {
		name string
		e    lib.UnaryExpr
		want lib.Value
	}{
		{"not_true", lib.UnaryExpr{Op: lib.UnaryNot, Operand: bl("true")}, vBool(false)},
		{"not_false", lib.UnaryExpr{Op: lib.UnaryNot, Operand: bl("false")}, vBool(true)},
		{"not_null", lib.UnaryExpr{Op: lib.UnaryNot, Operand: missing}, vBool(true)},
		{"neg_int", lib.UnaryExpr{Op: lib.UnaryNeg, Operand: il("5")}, vInt(-5)},
		{"neg_null", lib.UnaryExpr{Op: lib.UnaryNeg, Operand: missing}, vNull()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.e.Eval(&fm)
			if !valEq(got, tc.want) {
				t.Errorf("Eval() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// ── Row 5: BinExpr (and/or) ───────────────────────────────────────────────────

func TestBinExpr_BoolOps(t *testing.T) {
	bl := func(s string) lib.LitExpr { return lib.LitExpr{Kind: lib.LitBool, Value: s} }
	missing := lib.FieldExpr{Field: lib.Field{Name: "missing"}}
	fm := lib.FrontMatter{}

	tests := []struct {
		name string
		e    lib.BinExpr
		want lib.Value
	}{
		{"and_tt", lib.BinExpr{Op: lib.BinAnd, Left: bl("true"), Right: bl("true")}, vBool(true)},
		{"and_tf", lib.BinExpr{Op: lib.BinAnd, Left: bl("true"), Right: bl("false")}, vBool(false)},
		{"and_ft", lib.BinExpr{Op: lib.BinAnd, Left: bl("false"), Right: bl("true")}, vBool(false)},
		{"and_null_lhs", lib.BinExpr{Op: lib.BinAnd, Left: missing, Right: bl("true")}, vBool(false)},
		{"and_null_rhs", lib.BinExpr{Op: lib.BinAnd, Left: bl("true"), Right: missing}, vBool(false)},
		{"or_tf", lib.BinExpr{Op: lib.BinOr, Left: bl("true"), Right: bl("false")}, vBool(true)},
		{"or_ff", lib.BinExpr{Op: lib.BinOr, Left: bl("false"), Right: bl("false")}, vBool(false)},
		{"or_null_null", lib.BinExpr{Op: lib.BinOr, Left: missing, Right: missing}, vBool(false)},
		{"or_null_true", lib.BinExpr{Op: lib.BinOr, Left: missing, Right: bl("true")}, vBool(true)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.e.Eval(&fm)
			if !valEq(got, tc.want) {
				t.Errorf("Eval() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// ── Row 6: BinExpr (arith) ────────────────────────────────────────────────────

func TestBinExpr_Arith(t *testing.T) {
	il := func(s string) lib.LitExpr { return lib.LitExpr{Kind: lib.LitInt, Value: s} }
	nl := func(s string) lib.LitExpr { return lib.LitExpr{Kind: lib.LitNumeric, Value: s} }
	missing := lib.FieldExpr{Field: lib.Field{Name: "missing"}}
	fm := lib.FrontMatter{}

	tests := []struct {
		name string
		e    lib.BinExpr
		want lib.Value
	}{
		{"add_ints", lib.BinExpr{Op: lib.BinAdd, Left: il("2"), Right: il("3")}, vInt(5)},
		{"sub_ints", lib.BinExpr{Op: lib.BinSub, Left: il("10"), Right: il("4")}, vInt(6)},
		{"mul_ints", lib.BinExpr{Op: lib.BinMul, Left: il("3"), Right: il("4")}, vInt(12)},
		{"div_ints_exact", lib.BinExpr{Op: lib.BinDiv, Left: il("10"), Right: il("2")}, vInt(5)},
		{"add_numerics", lib.BinExpr{Op: lib.BinAdd, Left: nl("1.5"), Right: nl("2.5")}, vNum(4.0)},
		{"mix_int_numeric", lib.BinExpr{Op: lib.BinAdd, Left: il("1"), Right: nl("2.5")}, vNum(3.5)},
		{"add_null_lhs", lib.BinExpr{Op: lib.BinAdd, Left: missing, Right: il("1")}, vNull()},
		{"add_null_rhs", lib.BinExpr{Op: lib.BinAdd, Left: il("1"), Right: missing}, vNull()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.e.Eval(&fm)
			if !valEq(got, tc.want) {
				t.Errorf("Eval() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// ── Row 7: BinExpr (comparison) ───────────────────────────────────────────────

func TestBinExpr_Compare(t *testing.T) {
	il := func(s string) lib.LitExpr { return lib.LitExpr{Kind: lib.LitInt, Value: s} }
	sl := func(s string) lib.LitExpr { return lib.LitExpr{Kind: lib.LitString, Value: s} }
	missing := lib.FieldExpr{Field: lib.Field{Name: "missing"}}
	fm := lib.FrontMatter{}

	tests := []struct {
		name string
		e    lib.BinExpr
		want lib.Value
	}{
		{"eq_ints_true", lib.BinExpr{Op: lib.BinEq, Left: il("1"), Right: il("1")}, vBool(true)},
		{"eq_ints_false", lib.BinExpr{Op: lib.BinEq, Left: il("1"), Right: il("2")}, vBool(false)},
		{"ne_ints", lib.BinExpr{Op: lib.BinNe, Left: il("1"), Right: il("2")}, vBool(true)},
		{"lt", lib.BinExpr{Op: lib.BinLt, Left: il("1"), Right: il("2")}, vBool(true)},
		{"le_eq", lib.BinExpr{Op: lib.BinLe, Left: il("2"), Right: il("2")}, vBool(true)},
		{"gt_false", lib.BinExpr{Op: lib.BinGt, Left: il("1"), Right: il("2")}, vBool(false)},
		{"ge_eq", lib.BinExpr{Op: lib.BinGe, Left: il("2"), Right: il("2")}, vBool(true)},
		{"eq_strings", lib.BinExpr{Op: lib.BinEq, Left: sl("a"), Right: sl("a")}, vBool(true)},
		{"eq_null_lhs", lib.BinExpr{Op: lib.BinEq, Left: missing, Right: il("1")}, vBool(false)},
		{"eq_null_rhs", lib.BinExpr{Op: lib.BinEq, Left: il("1"), Right: missing}, vBool(false)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.e.Eval(&fm)
			if !valEq(got, tc.want) {
				t.Errorf("Eval() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// ── Row 8: Assign.Apply ───────────────────────────────────────────────────────

func TestAssign_Apply(t *testing.T) {
	t.Run("set_literal", func(t *testing.T) {
		fm := lib.FrontMatter{}
		a := lib.Assign{
			Field: lib.Field{Name: "title"},
			Op:    lib.OpSet,
			Value: lib.LitExpr{Kind: lib.LitString, Value: "hello"},
		}
		if err := a.Apply(&fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if fm["title"] != "hello" {
			t.Errorf("title = %v, want hello", fm["title"])
		}
	})
	t.Run("set_typed_cast_ok", func(t *testing.T) {
		fm := lib.FrontMatter{}
		a := lib.Assign{
			Field: lib.Field{Name: "n", Type: lib.TypeInt},
			Op:    lib.OpSet,
			Value: lib.LitExpr{Kind: lib.LitString, Value: "42"},
		}
		if err := a.Apply(&fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if fm["n"] != int64(42) {
			t.Errorf("n = %v (%T), want int64(42)", fm["n"], fm["n"])
		}
	})
	t.Run("cast_only_creates_null", func(t *testing.T) {
		fm := lib.FrontMatter{}
		a := lib.Assign{
			Field: lib.Field{Name: "foo", Type: lib.TypeInt},
			Op:    lib.OpSet,
			Value: nil,
		}
		if err := a.Apply(&fm); err != nil {
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
		fm := lib.FrontMatter{"title": "hello"}
		a := lib.Assign{
			Field: lib.Field{Name: "title"},
			Op:    lib.OpSet,
			Value: lib.LitExpr{Kind: lib.LitNull, Value: "null"},
		}
		if err := a.Apply(&fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if fm["title"] != nil {
			t.Errorf("title = %v, want nil", fm["title"])
		}
	})
	t.Run("add_to_list", func(t *testing.T) {
		fm := lib.FrontMatter{"tags": []any{"a"}}
		a := lib.Assign{
			Field: lib.Field{Name: "tags"},
			Op:    lib.OpAdd,
			Value: lib.LitExpr{Kind: lib.LitString, Value: "b"},
		}
		if err := a.Apply(&fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		tags, ok := fm["tags"].([]any)
		if !ok || len(tags) != 2 || tags[1] != "b" {
			t.Errorf("tags = %+v, want [a b]", fm["tags"])
		}
	})
	t.Run("sub_from_list", func(t *testing.T) {
		fm := lib.FrontMatter{"tags": []any{"a", "b", "c"}}
		a := lib.Assign{
			Field: lib.Field{Name: "tags"},
			Op:    lib.OpSub,
			Value: lib.LitExpr{Kind: lib.LitString, Value: "b"},
		}
		if err := a.Apply(&fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		tags, _ := fm["tags"].([]any)
		if len(tags) != 2 || tags[0] != "a" || tags[1] != "c" {
			t.Errorf("tags = %+v, want [a c]", tags)
		}
	})
	t.Run("add_to_null_field", func(t *testing.T) {
		fm := lib.FrontMatter{"tags": nil}
		a := lib.Assign{
			Field: lib.Field{Name: "tags"},
			Op:    lib.OpAdd,
			Value: lib.LitExpr{Kind: lib.LitString, Value: "x"},
		}
		if err := a.Apply(&fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		tags, ok := fm["tags"].([]any)
		if !ok || len(tags) != 1 || tags[0] != "x" {
			t.Errorf("tags = %+v, want [x]", fm["tags"])
		}
	})
	t.Run("sub_from_null_field", func(t *testing.T) {
		fm := lib.FrontMatter{"tags": nil}
		a := lib.Assign{
			Field: lib.Field{Name: "tags"},
			Op:    lib.OpSub,
			Value: lib.LitExpr{Kind: lib.LitString, Value: "x"},
		}
		if err := a.Apply(&fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if fm["tags"] != nil {
			t.Errorf("tags = %+v, want nil", fm["tags"])
		}
	})
	t.Run("cast_failure_errors", func(t *testing.T) {
		fm := lib.FrontMatter{"src": "hello"}
		a := lib.Assign{
			Field: lib.Field{Name: "n", Type: lib.TypeInt},
			Op:    lib.OpSet,
			Value: lib.FieldExpr{Field: lib.Field{Name: "src", Type: lib.TypeString}},
		}
		if err := a.Apply(&fm); err == nil {
			t.Error("expected cast-failure error, got nil")
		}
	})
}

// ── Row 9: SortTerm.Eval ──────────────────────────────────────────────────────

func TestSortTerm_Eval(t *testing.T) {
	fm := lib.FrontMatter{"title": "abc"}
	s := lib.SortTerm{
		Expr: lib.FieldExpr{Field: lib.Field{Name: "title"}},
		Desc: false,
	}
	if got := s.Eval(&fm); !valEq(got, vStr("abc")) {
		t.Errorf("Eval() = %+v, want %+v", got, vStr("abc"))
	}
}

// ── Row 10: AlterQuery.Apply ──────────────────────────────────────────────────

func TestAlterQuery_Apply(t *testing.T) {
	t.Run("drop_one", func(t *testing.T) {
		fm := lib.FrontMatter{"title": "x", "date": "2026-01-01"}
		q := parseQ(t, "alter *.md drop title").(lib.AlterQuery)
		if err := q.Apply(&fm); err != nil {
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
		fm := lib.FrontMatter{"a": int64(1), "b": int64(2), "c": int64(3)}
		q := parseQ(t, "alter *.md drop a, b").(lib.AlterQuery)
		if err := q.Apply(&fm); err != nil {
			t.Fatal(err)
		}
		if len(fm) != 1 || fm["c"] != int64(3) {
			t.Errorf("FrontMatter = %+v, want {c:3}", fm)
		}
	})
	t.Run("rename", func(t *testing.T) {
		fm := lib.FrontMatter{"foo": "hello"}
		q := parseQ(t, "alter *.md rename foo to bar").(lib.AlterQuery)
		if err := q.Apply(&fm); err != nil {
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
		fm := lib.FrontMatter{"published": false, "title": "x"}
		q := parseQ(t, "alter *.md drop title where published = true").(lib.AlterQuery)
		if err := q.Apply(&fm); err != nil {
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
		fm := lib.FrontMatter{}
		q := parseQ(t, `update *.md set title = "hello"`).(lib.UpdateQuery)
		if err := q.Apply(&fm); err != nil {
			t.Fatal(err)
		}
		if fm["title"] != "hello" {
			t.Errorf("title = %v", fm["title"])
		}
	})
	t.Run("set_multi", func(t *testing.T) {
		fm := lib.FrontMatter{}
		q := parseQ(t, `update *.md set a = 1, b = 2`).(lib.UpdateQuery)
		if err := q.Apply(&fm); err != nil {
			t.Fatal(err)
		}
		if fm["a"] != int64(1) || fm["b"] != int64(2) {
			t.Errorf("FrontMatter = %+v", fm)
		}
	})
	t.Run("where_skips", func(t *testing.T) {
		fm := lib.FrontMatter{"published": false}
		q := parseQ(t, `update *.md set title = "x" where published = true`).(lib.UpdateQuery)
		if err := q.Apply(&fm); err != nil {
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
		fm := lib.FrontMatter{"title": "hello", "date": "2026-01-01"}
		q := parseQ(t, "select title from *.md").(lib.SelectQuery)
		row, err := q.Eval(&fm)
		if err != nil {
			t.Fatal(err)
		}
		if len(row) != 1 || !valEq(row[0], vStr("hello")) {
			t.Errorf("row = %+v, want [hello]", row)
		}
	})
	t.Run("project_multi", func(t *testing.T) {
		fm := lib.FrontMatter{"a": "x", "b": int64(42)}
		q := parseQ(t, "select a, b from *.md").(lib.SelectQuery)
		row, err := q.Eval(&fm)
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
		fm := lib.FrontMatter{"published": false, "title": "x"}
		q := parseQ(t, "select title from *.md where published = true").(lib.SelectQuery)
		row, err := q.Eval(&fm)
		if err != nil {
			t.Fatal(err)
		}
		if row != nil {
			t.Errorf("row = %+v, want nil (where=false)", row)
		}
	})
	t.Run("missing_field_null", func(t *testing.T) {
		fm := lib.FrontMatter{}
		q := parseQ(t, "select title from *.md").(lib.SelectQuery)
		row, err := q.Eval(&fm)
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
	fm := lib.FrontMatter{
		"created":  time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC),
		"modified": time.Date(2026, 5, 14, 21, 2, 30, 0, time.UTC),
	}
	tests := []struct {
		name string
		f    lib.Field
		want lib.Value
	}{
		{"time_as_date", lib.Field{Name: "created", Type: lib.TypeDate}, vDate(2026, 5, 14)},
		{"time_as_datetime", lib.Field{Name: "modified", Type: lib.TypeDatetime}, vDatetime(2026, 5, 14, 21, 2, 30)},
		{"time_any_detects_date", lib.Field{Name: "created", Type: lib.TypeAny}, vDate(2026, 5, 14)},
		{"time_any_detects_datetime", lib.Field{Name: "modified", Type: lib.TypeAny}, vDatetime(2026, 5, 14, 21, 2, 30)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := lib.FieldExpr{Field: tc.f}
			got := e.Eval(&fm)
			if !valEq(got, tc.want) {
				t.Errorf("Eval(%+v) = %+v, want %+v", tc.f, got, tc.want)
			}
		})
	}
}

// ── Link/mdlink types ─────────────────────────────────────────────────────────

func TestFieldExpr_Eval_Link(t *testing.T) {
	fm := lib.FrontMatter{
		"source": "[[ref|title]]",
		"url":    "[title](ref)",
	}
	tests := []struct {
		name string
		f    lib.Field
		want lib.Value
	}{
		{"string_as_link", lib.Field{Name: "source", Type: lib.TypeLink}, vLink("[[ref|title]]")},
		{"string_as_mdlink", lib.Field{Name: "url", Type: lib.TypeMdLink}, vMdLink("[title](ref)")},
		{"link_to_mdlink", lib.Field{Name: "source", Type: lib.TypeMdLink}, vMdLink("[title](ref)")},
		{"mdlink_to_link_via_field", lib.Field{Name: "url", Type: lib.TypeLink}, vLink("[[ref|title]]")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := lib.FieldExpr{Field: tc.f}
			got := e.Eval(&fm)
			if !valEq(got, tc.want) {
				t.Errorf("Eval(%+v) = %+v, want %+v", tc.f, got, tc.want)
			}
		})
	}
}

// ── List element type ─────────────────────────────────────────────────────────

func TestCastField_ListElemType(t *testing.T) {
	strType := lib.TypeString
	intType := lib.TypeInt

	tests := []struct {
		name  string
		v     lib.Value
		field lib.Field
		want  lib.Value
	}{
		{
			"no_elemtype_passthrough",
			vList(vStr("a")),
			lib.Field{Type: lib.TypeList},
			vList(vStr("a")),
		},
		{
			"elemtype_string_noop",
			vList(vStr("a"), vStr("b")),
			lib.Field{Type: lib.TypeList, ElemType: &strType},
			vList(vStr("a"), vStr("b")),
		},
		{
			"elemtype_int_cast_ok",
			vList(vStr("3"), vStr("4")),
			lib.Field{Type: lib.TypeList, ElemType: &intType},
			vList(vInt(3), vInt(4)),
		},
		{
			"elemtype_int_cast_fail",
			vList(vStr("hello"), vStr("4")),
			lib.Field{Type: lib.TypeList, ElemType: &intType},
			vNull(),
		},
		{
			"scalar_wrapped_and_cast",
			vInt(5),
			lib.Field{Type: lib.TypeList, ElemType: &intType},
			vList(vInt(5)),
		},
		{
			"non_list_delegates_to_cast",
			vStr("42"),
			lib.Field{Type: lib.TypeInt},
			vInt(42),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantErr := tc.want.Null
			got, err := lib.CastField(tc.v, tc.field)
			if wantErr {
				if err == nil {
					t.Errorf("CastField(%+v, %+v) = %+v, want error", tc.v, tc.field, got)
				}
			} else {
				if err != nil {
					t.Errorf("CastField(%+v, %+v) error: %s", tc.v, tc.field, err)
				} else if !valEq(got, tc.want) {
					t.Errorf("CastField(%+v, %+v) = %+v, want %+v", tc.v, tc.field, got, tc.want)
				}
			}
		})
	}
}

func TestFieldExpr_Eval_ListElemType(t *testing.T) {
	et := lib.TypeInt
	fm := lib.FrontMatter{"nums": []any{"3", "4", "5"}}
	e := lib.FieldExpr{Field: lib.Field{Name: "nums", Type: lib.TypeList, ElemType: &et}}
	got := e.Eval(&fm)
	want := vList(vInt(3), vInt(4), vInt(5))
	if !valEq(got, want) {
		t.Errorf("Eval() = %+v, want %+v", got, want)
	}
}
