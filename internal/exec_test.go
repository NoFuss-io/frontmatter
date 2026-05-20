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

