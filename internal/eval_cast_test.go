package internal

import (
	"testing"
)

func TestCastReversible(t *testing.T) {
	testCases := [][]Value{
		{vBool(true), vInt(1), vNum(1.0)},
		{vBool(true), vStr("true")},
		{vInt(1), vNum(1.0)},
		{vInt(3), vStr("3")},
		{vNum(4.56), vStr("4.56")},

		{vLink("[[ref]]"), vStr("[[ref]]")},
		{vMdLink("[ref](ref)"), vStr("[ref](ref)")},
		{vLink("[[ref]]"), vMdLink("[ref](ref)")},

		{vStr("2026-05-14"), vDate(2026, 5, 14)},
		{vStr("2026-05-14T21:02:30"), vDatetime(2026, 5, 14, 21, 2, 30)},

		{vStr("hello")},
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
