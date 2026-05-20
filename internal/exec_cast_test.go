package internal

import (
	"testing"
)

func TestCastReversible(t *testing.T) {
	testCases := [][]Value{
		[]Value{vBool(true), vInt(1), vNum(1.0)},
		[]Value{vBool(true), vStr("true")},
		[]Value{vInt(1), vNum(1.0)},
		[]Value{vInt(3), vStr("3")},
		[]Value{vNum(4.56), vStr("4.56")},

		[]Value{vLink("[[ref]]"), vStr("[[ref]]")},
		[]Value{vMdLink("[ref](ref)"), vStr("[ref](ref)")},
		[]Value{vLink("[[ref]]"), vMdLink("[ref](ref)")},

		[]Value{vStr("2026-05-14"), vDate(2026, 5, 14)},
		[]Value{vStr("2026-05-14T21:02:30"), vDatetime(2026, 5, 14, 21, 2, 30)},

		[]Value{vStr("hello")},
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
		{vBool(false), allBut(TypeInt, TypeNumber, TypeString, TypeList)},
		{vInt(2), allBut(TypeNumber, TypeString, TypeList)},
		{vNum(1.5), allBut(TypeString, TypeList)},

		{vLink("[[ref]]"), allBut(TypeMdLink, TypeString, TypeList)},
		{vMdLink("[title](ref)"), allBut(TypeLink, TypeString, TypeList)},

		{vStr("hello"), allBut(TypeList)},

		{vDate(2026, 5, 14), allBut(TypeString, TypeList)},
		{vDatetime(2026, 5, 14, 21, 2, 30), allBut(TypeString, TypeList)},

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
