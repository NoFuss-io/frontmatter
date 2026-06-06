package internal

import (
	"testing"
)

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
		{"eq_null_null", BinExpr{Op: BinEq, Left: missing, Right: missing}, vBool(true)},
		{"ne_null_null", BinExpr{Op: BinNe, Left: missing, Right: missing}, vBool(false)},
		{"ne_null_value", BinExpr{Op: BinNe, Left: missing, Right: il("1")}, vBool(true)},
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

// ── Set intersection operator (<=>) ──────────────────────────────────────────

func TestSetOp_Intersect(t *testing.T) {
	tests := []struct {
		name string
		l, r Value
		want Value
	}{
		{"list_list_overlap", vList(vStr("a"), vStr("b"), vStr("c")), vList(vStr("b"), vStr("c"), vStr("d")), vList(vStr("b"), vStr("c"))},
		{"list_list_disjoint", vList(vStr("a"), vStr("b")), vList(vStr("c"), vStr("d")), vList()},
		{"list_list_identical", vList(vStr("x"), vStr("y")), vList(vStr("x"), vStr("y")), vList(vStr("x"), vStr("y"))},
		{"scalar_scalar_eq", vStr("a"), vStr("a"), vList(vStr("a"))},
		{"scalar_scalar_ne", vStr("a"), vStr("b"), vList()},
		{"null_lhs", vNull(), vList(vStr("a")), vNull()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := setOp(BinIntersect, tc.l, tc.r)
			if !valEq(got, tc.want) {
				t.Errorf("setOp(BinIntersect, %v, %v) = %v, want %v", tc.l, tc.r, got, tc.want)
			}
		})
	}
}

// ── Set union operator (>=<) ──────────────────────────────────────────────────

func TestSetOp_Union(t *testing.T) {
	tests := []struct {
		name string
		l, r Value
		want Value
	}{
		{"distinct_merge", vList(vStr("a"), vStr("b")), vList(vStr("b"), vStr("c")), vList(vStr("a"), vStr("b"), vStr("c"))},
		{"no_overlap", vList(vStr("a")), vList(vStr("b")), vList(vStr("a"), vStr("b"))},
		{"identical", vList(vStr("x")), vList(vStr("x")), vList(vStr("x"))},
		{"scalar_scalar", vStr("a"), vStr("b"), vList(vStr("a"), vStr("b"))},
		{"null_lhs", vNull(), vList(vStr("a")), vNull()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := setOp(BinUnion, tc.l, tc.r)
			if !valEq(got, tc.want) {
				t.Errorf("setOp(BinUnion, %v, %v) = %v, want %v", tc.l, tc.r, got, tc.want)
			}
		})
	}
}

// ── Match operators (LIKE / ILIKE / REGEXP) ───────────────────────────────────

func TestMatchOp(t *testing.T) {
	tests := []struct {
		name string
		op   BinOp
		l, r Value
		want Value
	}{
		// LIKE
		{"like_prefix", BinLike, vStr("foobar"), vStr("foo%"), vBool(true)},
		{"like_suffix", BinLike, vStr("foobar"), vStr("%bar"), vBool(true)},
		{"like_contains", BinLike, vStr("foobar"), vStr("%oba%"), vBool(true)},
		{"like_underscore", BinLike, vStr("foobar"), vStr("f_obar"), vBool(true)},
		{"like_miss", BinLike, vStr("foobar"), vStr("baz%"), vBool(false)},
		{"like_case_sensitive", BinLike, vStr("Foobar"), vStr("foo%"), vBool(false)},
		// NOT LIKE
		{"not_like_hit", BinNotLike, vStr("foobar"), vStr("baz%"), vBool(true)},
		{"not_like_miss", BinNotLike, vStr("foobar"), vStr("foo%"), vBool(false)},
		// ILIKE
		{"ilike_case", BinILike, vStr("Foobar"), vStr("foo%"), vBool(true)},
		{"ilike_miss", BinILike, vStr("Foobar"), vStr("baz%"), vBool(false)},
		// NOT ILIKE
		{"not_ilike_hit", BinNotILike, vStr("Foobar"), vStr("baz%"), vBool(true)},
		// REGEXP
		{"regexp_match", BinRegexp, vStr("abc123"), vStr(`[a-z]+\d+`), vBool(true)},
		{"regexp_miss", BinRegexp, vStr("abc"), vStr(`^\d+`), vBool(false)},
		// NOT REGEXP
		{"not_regexp_hit", BinNotRegexp, vStr("abc"), vStr(`^\d+`), vBool(true)},
		// null propagation
		{"null_subject", BinLike, vNull(), vStr("%foo%"), vNull()},
		{"null_pattern", BinLike, vStr("foo"), vNull(), vNull()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := matchOp(tc.op, tc.l, tc.r)
			if !valEq(got, tc.want) {
				t.Errorf("matchOp(%v, %v, %v) = %v, want %v", tc.op, tc.l, tc.r, got, tc.want)
			}
		})
	}
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
