package internal

import (
	"testing"
	"time"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func fe(name string, args ...Expr) FuncExpr { return FuncExpr{Name: name, Args: args} }
func lit(s string) LitExpr                  { return LitExpr{Kind: LitString, Value: s} }
func ilit(s string) LitExpr                 { return LitExpr{Kind: LitInt, Value: s} }
func flit(s string) LitExpr                 { return LitExpr{Kind: LitNumeric, Value: s} }

func evalF(fn FuncExpr) Value { return fn.Eval(nil) }

// ── String functions ──────────────────────────────────────────────────────────

func TestFn_Lower(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Hello World", "hello world"},
		{"CAFÉ", "café"},
		{"", ""},
	}
	for _, tc := range tests {
		got := evalF(fe("lower", lit(tc.in)))
		if !valEq(got, vStr(tc.want)) {
			t.Errorf("lower(%q) = %v, want %v", tc.in, got, vStr(tc.want))
		}
	}
}

func TestFn_Upper(t *testing.T) {
	got := evalF(fe("upper", lit("hello")))
	if !valEq(got, vStr("HELLO")) {
		t.Errorf("upper(%q) = %v, want %v", "hello", got, vStr("HELLO"))
	}
}

func TestFn_Length(t *testing.T) {
	tests := []struct {
		in   string
		want int64
	}{
		{"hello", 5},
		{"café", 4}, // 4 codepoints
		{"", 0},
	}
	for _, tc := range tests {
		got := evalF(fe("length", lit(tc.in)))
		if !valEq(got, vInt(tc.want)) {
			t.Errorf("length(%q) = %v, want %v", tc.in, got, vInt(tc.want))
		}
	}
}

func TestFn_Substr(t *testing.T) {
	tests := []struct {
		name string
		fn   FuncExpr
		want Value
	}{
		{"from1", fe("substr", lit("hello"), ilit("1")), vStr("hello")},
		{"from2", fe("substr", lit("hello"), ilit("2")), vStr("ello")},
		{"from2len3", fe("substr", lit("hello"), ilit("2"), ilit("3")), vStr("ell")},
		{"neg_pos", fe("substr", lit("hello"), ilit("-2")), vStr("lo")},
		{"overflow_len", fe("substr", lit("hi"), ilit("1"), ilit("99")), vStr("hi")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := evalF(tc.fn)
			if !valEq(got, tc.want) {
				t.Errorf("= %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFn_StartsWith(t *testing.T) {
	if !valEq(evalF(fe("starts_with", lit("foobar"), lit("foo"))), vBool(true)) {
		t.Error("starts_with foobar foo: want true")
	}
	if !valEq(evalF(fe("starts_with", lit("foobar"), lit("bar"))), vBool(false)) {
		t.Error("starts_with foobar bar: want false")
	}
}

func TestFn_EndsWith(t *testing.T) {
	if !valEq(evalF(fe("ends_with", lit("foobar"), lit("bar"))), vBool(true)) {
		t.Error("ends_with foobar bar: want true")
	}
}

func TestFn_ContainsSubstr(t *testing.T) {
	if !valEq(evalF(fe("contains_substr", lit("foobar"), lit("oba"))), vBool(true)) {
		t.Error("contains_substr foobar oba: want true")
	}
}

func TestFn_Trim(t *testing.T) {
	tests := []struct {
		name string
		fn   FuncExpr
		want string
	}{
		{"both", fe("trim", lit("  hello  ")), "hello"},
		{"left", fe("ltrim", lit("  hello  ")), "hello  "},
		{"right", fe("rtrim", lit("  hello  ")), "  hello"},
		{"chars", fe("trim", lit("xxhelloxx"), lit("x")), "hello"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !valEq(evalF(tc.fn), vStr(tc.want)) {
				t.Errorf("= %v, want %v", evalF(tc.fn), vStr(tc.want))
			}
		})
	}
}

func TestFn_Replace(t *testing.T) {
	got := evalF(fe("replace", lit("aabbcc"), lit("b"), lit("X")))
	if !valEq(got, vStr("aaXXcc")) {
		t.Errorf("replace = %v, want aaXXcc", got)
	}
}

func TestFn_Split(t *testing.T) {
	got := evalF(fe("split", lit("a,b,c"), lit(",")))
	want := vList(vStr("a"), vStr("b"), vStr("c"))
	if !valEq(got, want) {
		t.Errorf("split = %v, want %v", got, want)
	}
}

func TestFn_Concat(t *testing.T) {
	got := evalF(fe("concat", lit("foo"), lit("bar"), lit("!")))
	if !valEq(got, vStr("foobar!")) {
		t.Errorf("concat = %v, want foobar!", got)
	}
	// Too few args → null
	if !evalF(fe("concat", lit("x"))).Null {
		t.Error("concat(1 arg) should be null")
	}
}

func TestFn_RegexpContains(t *testing.T) {
	if !valEq(evalF(fe("regexp_contains", lit("abc123"), lit(`\d+`))), vBool(true)) {
		t.Error("regexp_contains: want true")
	}
	if !valEq(evalF(fe("regexp_contains", lit("abc"), lit(`^\d+`))), vBool(false)) {
		t.Error("regexp_contains miss: want false")
	}
}

func TestFn_RegexpExtract(t *testing.T) {
	// No capture group → return full match
	got := evalF(fe("regexp_extract", lit("abc123"), lit(`\d+`)))
	if !valEq(got, vStr("123")) {
		t.Errorf("regexp_extract (no group) = %v, want 123", got)
	}
	// With capture group
	got = evalF(fe("regexp_extract", lit("date: 2024-01-15"), lit(`(\d{4}-\d{2}-\d{2})`)))
	if !valEq(got, vStr("2024-01-15")) {
		t.Errorf("regexp_extract (group) = %v, want 2024-01-15", got)
	}
	// No match → null
	if !evalF(fe("regexp_extract", lit("abc"), lit(`\d+`))).Null {
		t.Error("regexp_extract (no match): want null")
	}
}

func TestFn_ToString(t *testing.T) {
	if !valEq(evalF(fe("to_string", ilit("42"))), vStr("42")) {
		t.Error("to_string(42) want '42'")
	}
}

// ── Numeric functions ─────────────────────────────────────────────────────────

func TestFn_Abs(t *testing.T) {
	if !valEq(evalF(fe("abs", ilit("-5"))), vInt(5)) {
		t.Error("abs(-5) want 5")
	}
	if !valEq(evalF(fe("abs", ilit("3"))), vInt(3)) {
		t.Error("abs(3) want 3")
	}
}

func TestFn_Ceil(t *testing.T) {
	if !valEq(evalF(fe("ceil", flit("2.1"))), vInt(3)) {
		t.Error("ceil(2.1) want 3")
	}
	if !valEq(evalF(fe("ceil", ilit("4"))), vInt(4)) {
		t.Error("ceil(int 4) want 4")
	}
}

func TestFn_Floor(t *testing.T) {
	if !valEq(evalF(fe("floor", flit("2.9"))), vInt(2)) {
		t.Error("floor(2.9) want 2")
	}
}

func TestFn_Round(t *testing.T) {
	if !valEq(evalF(fe("round", flit("2.567"), ilit("2"))), Value{Kind: TypeNumber, Data: 2.57}) {
		t.Errorf("round(2.567,2) = %v, want 2.57", evalF(fe("round", flit("2.567"), ilit("2"))))
	}
	if !valEq(evalF(fe("round", flit("2.5"))), Value{Kind: TypeNumber, Data: float64(3)}) {
		t.Errorf("round(2.5) = %v, want 3", evalF(fe("round", flit("2.5"))))
	}
}

func TestFn_Mod(t *testing.T) {
	if !valEq(evalF(fe("mod", ilit("10"), ilit("3"))), vInt(1)) {
		t.Error("mod(10,3) want 1")
	}
	if !evalF(fe("mod", ilit("1"), ilit("0"))).Null {
		t.Error("mod(1,0) want null")
	}
}

func TestFn_Sqrt(t *testing.T) {
	got := evalF(fe("sqrt", ilit("4")))
	if !valEq(got, Value{Kind: TypeNumber, Data: float64(2)}) {
		t.Errorf("sqrt(4) = %v, want 2.0", got)
	}
}

func TestFn_Pow(t *testing.T) {
	got := evalF(fe("pow", flit("2"), flit("10")))
	if !valEq(got, Value{Kind: TypeNumber, Data: float64(1024)}) {
		t.Errorf("pow(2,10) = %v, want 1024", got)
	}
}

func TestFn_Greatest(t *testing.T) {
	got := evalF(fe("greatest", ilit("3"), ilit("1"), ilit("5"), ilit("2")))
	if !valEq(got, vInt(5)) {
		t.Errorf("greatest = %v, want 5", got)
	}
}

func TestFn_Least(t *testing.T) {
	got := evalF(fe("least", ilit("3"), ilit("1"), ilit("5")))
	if !valEq(got, vInt(1)) {
		t.Errorf("least = %v, want 1", got)
	}
}

func TestFn_Coalesce(t *testing.T) {
	null := LitExpr{Kind: LitNull, Value: "null"}
	got := evalF(fe("coalesce", null, null, lit("found")))
	if !valEq(got, vStr("found")) {
		t.Errorf("coalesce = %v, want 'found'", got)
	}
	if !evalF(fe("coalesce", null, null)).Null {
		t.Error("coalesce(null,null) want null")
	}
}

// ── List functions ────────────────────────────────────────────────────────────

func listExpr(ss ...string) ListExpr {
	elems := make([]Expr, len(ss))
	for i, s := range ss {
		elems[i] = lit(s)
	}
	return ListExpr{Elems: elems}
}

func TestFn_ArrayLength(t *testing.T) {
	got := evalF(fe("array_length", listExpr("a", "b", "c")))
	if !valEq(got, vInt(3)) {
		t.Errorf("array_length = %v, want 3", got)
	}
	// Scalar counts as 1
	if !valEq(evalF(fe("array_length", lit("x"))), vInt(1)) {
		t.Error("array_length(scalar) want 1")
	}
}

func TestFn_ArrayContains(t *testing.T) {
	fn := fe("array_contains", listExpr("a", "b", "c"), lit("b"))
	if !valEq(evalF(fn), vBool(true)) {
		t.Error("array_contains hit: want true")
	}
	fn2 := fe("array_contains", listExpr("a", "b"), lit("z"))
	if !valEq(evalF(fn2), vBool(false)) {
		t.Error("array_contains miss: want false")
	}
}

func TestFn_ArrayConcat(t *testing.T) {
	got := evalF(fe("array_concat", listExpr("a", "b"), listExpr("b", "c")))
	want := vList(vStr("a"), vStr("b"), vStr("b"), vStr("c"))
	if !valEq(got, want) {
		t.Errorf("array_concat = %v, want %v", got, want)
	}
}

func TestFn_Distinct(t *testing.T) {
	got := evalF(fe("distinct", listExpr("a", "b", "a", "c", "b")))
	want := vList(vStr("a"), vStr("b"), vStr("c"))
	if !valEq(got, want) {
		t.Errorf("distinct = %v, want %v", got, want)
	}
}

func TestFn_ArrayToString(t *testing.T) {
	got := evalF(fe("array_to_string", listExpr("a", "b", "c"), lit(", ")))
	if !valEq(got, vStr("a, b, c")) {
		t.Errorf("array_to_string = %v, want 'a, b, c'", got)
	}
}

// ── Date functions ────────────────────────────────────────────────────────────

func TestFn_Today(t *testing.T) {
	got := evalF(fe("today"))
	if got.Null || got.Kind != TypeDate {
		t.Errorf("today() = %v, want TypeDate", got)
	}
	now := time.Now()
	d := got.Data.(time.Time)
	if d.Year() != now.Year() || d.Month() != now.Month() || d.Day() != now.Day() {
		t.Errorf("today() date = %v, want today", d)
	}
}

func TestFn_YearMonthDay(t *testing.T) {
	dateVal := Value{Kind: TypeDate, Data: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)}
	dateExpr := LitExpr{Kind: LitString, Value: "2024-03-15"}

	fm := FrontMatter{"d": "2024-03-15"}
	fieldExpr := FieldExpr{Field: Field{Name: "d", Type: TypeDate}}
	_ = dateVal

	tests := []struct {
		fn   string
		want int64
	}{
		{"year", 2024},
		{"month", 3},
		{"day", 15},
	}
	for _, tc := range tests {
		got := FuncExpr{Name: tc.fn, Args: []Expr{dateExpr}}.Eval(fm)
		_ = fieldExpr
		if !valEq(got, vInt(tc.want)) {
			t.Errorf("%s(2024-03-15) = %v, want %d", tc.fn, got, tc.want)
		}
	}
}

func TestFn_DateDiff(t *testing.T) {
	// DATE_DIFF("2024-01-10", "2024-01-01") → 9
	got := FuncExpr{Name: "date_diff", Args: []Expr{
		LitExpr{Kind: LitString, Value: "2024-01-10"},
		LitExpr{Kind: LitString, Value: "2024-01-01"},
	}}.Eval(nil)
	if !valEq(got, vInt(9)) {
		t.Errorf("date_diff = %v, want 9", got)
	}
}

// ── Null propagation ──────────────────────────────────────────────────────────

func TestFn_NullPropagation(t *testing.T) {
	null := LitExpr{Kind: LitNull, Value: "null"}
	cases := []struct {
		name string
		fn   FuncExpr
	}{
		{"lower(null)", fe("lower", null)},
		{"length(null)", fe("length", null)},
		{"starts_with(null,x)", fe("starts_with", null, lit("x"))},
		{"concat(a,null)", fe("concat", lit("a"), null)},
		{"abs(null)", fe("abs", null)},
		{"mod(null,1)", fe("mod", null, ilit("1"))},
		{"array_length(null)", fe("array_length", null)},
		{"array_to_string(null,x)", fe("array_to_string", null, lit(","))},
		{"date_diff(null,x)", fe("date_diff", null, lit("2024-01-01"))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := evalF(tc.fn)
			if !got.Null {
				t.Errorf("%s = %v, want null", tc.name, got)
			}
		})
	}
}

// ── Arity errors ──────────────────────────────────────────────────────────────

func TestFn_ArityErrors(t *testing.T) {
	cases := []FuncExpr{
		fe("lower"),                       // 0 args, want 1
		fe("lower", lit("a"), lit("b")),   // 2 args, want 1
		fe("replace", lit("a"), lit("b")), // 2 args, want 3
		fe("mod", ilit("1")),              // 1 arg, want 2
	}
	for _, fn := range cases {
		got := evalF(fn)
		if !got.Null {
			t.Errorf("%s(%d args) = %v, want null", fn.Name, len(fn.Args), got)
		}
	}
}
