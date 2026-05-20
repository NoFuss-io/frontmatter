package internal

import (
	"testing"
	"time"
)

func TestCastReversible(t *testing.T) {
	testCases := [][]Value{
		[]Value{v(TypeBool, true), v(TypeInt, int64(1)), v(TypeNumber, 1.0)},
		[]Value{v(TypeBool, true), v(TypeString, "true")},
		[]Value{v(TypeInt, int64(1)), v(TypeNumber, 1.0)},
		[]Value{v(TypeInt, int64(3)), v(TypeString, "3")},
		[]Value{v(TypeNumber, 4.56), v(TypeString, "4.56")},

		[]Value{v(TypeLink, "[[ref]]"), v(TypeString, "[[ref]]")},
		[]Value{v(TypeMdLink, "[ref](ref)"), v(TypeString, "[ref](ref)")},
		[]Value{v(TypeLink, "[[ref]]"), v(TypeMdLink, "[ref](ref)")},

		[]Value{v(TypeString, "2026-05-14"), v(TypeDate, time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC))},
		[]Value{v(TypeString, "2026-05-14T21:02:30"), v(TypeDatetime, time.Date(2026, 5, 14, 21, 2, 30, 0, time.UTC))},

		[]Value{v(TypeString, "hello")},
	}

	for _, tc := range testCases {
		for _, x := range tc {
			for _, want := range tc {
				got, err := Cast(x, want.Kind)
				if err != nil {
					t.Errorf("got %s -> error: %s", x, err)
				}
				if got != want {
					t.Errorf("got %s -> %s, want %s", x, got, want)
				}
			}
		}
	}
}

func TestCastNull(t *testing.T) {
	for _, x := range allTypes {
		input := Value{x, nil, true}
		for _, y := range allTypes {
			got, err := Cast(input, y)
			if err == nil {
				t.Errorf("got %s -> %s, want error", x, got)
			}
		}
	}
}

func TestCastFailures(t *testing.T) {
	testCases := []struct {
		from Value
		to   []FieldType
	}{
		{v(TypeBool, false), allBut(TypeInt, TypeNumber, TypeString, TypeList)},
		{v(TypeInt, int64(2)), allBut(TypeNumber, TypeString, TypeList)},
		{v(TypeNumber, 1.5), allBut(TypeString, TypeList)},

		{v(TypeLink, "[[ref]]"), allBut(TypeMdLink, TypeString, TypeList)},
		{v(TypeMdLink, "[title](ref)"), allBut(TypeLink, TypeString, TypeList)},

		{v(TypeString, "hello"), allBut(TypeList)},

		{v(TypeDate, time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC)), allBut(TypeString, TypeList)},
		{v(TypeDatetime, time.Date(2026, 5, 14, 21, 2, 30, 0, time.UTC)), allBut(TypeString, TypeList)},

		{vList(vStr("a"), vStr("b")), allTypes},
	}

	for _, tc := range testCases {
		for _, to := range tc.to {
			if tc.from.Kind == to {
				continue
			}
			got, err := Cast(tc.from, to)
			if err == nil {
				t.Errorf("got %s -> %s, want error", tc.from, got)
			}
		}
	}
}

var allTypes = []FieldType{TypeBool, TypeInt, TypeNumber, TypeDate, TypeDatetime, TypeLink, TypeMdLink, TypeString, TypeList}

func allBut(tt ...FieldType) []FieldType {
	out := make([]FieldType, 0, len(allTypes)-len(tt))
	for _, t := range allTypes {
		for _, u := range tt {
			if t == u {
				goto next
			}
		}
		out = append(out, t)
	next:
	}
	return out
}

func v(ty FieldType, val any) Value {
	return Value{ty, val, false}
}
