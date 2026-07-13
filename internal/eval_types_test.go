package internal

import (
	"testing"
	"time"
)

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
			got, _ := e.Eval(fm)
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
			got, _ := e.Eval(fm)
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
	got, _ := e.Eval(fm)
	want := vList(vStr("3"), vStr("4"), vStr("5"))
	if !valEq(got, want) {
		t.Errorf("Eval() = %+v, want %+v", got, want)
	}
}
