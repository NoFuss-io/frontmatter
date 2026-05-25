package internal

import (
	"strings"
	"testing"
)

func FuzzParseProgram(f *testing.F) {
	seeds := []string{
		"",
		";;;",
		"select",
		"select * from *",
		"select title from recipes/*",
		`select title, tags from * where tags >= "vegan"`,
		`select count(*) from a sort by title limit 10`,
		"update *.md set title:string",
		`update * set tags += "x" where tags`,
		"alter *.md drop foo",
		"alter * rename foo to bar",
		"select * from a; update b set x = 1; -- comment\nalter c drop d",
		"-- only a comment\n",
		"select `field with spaces` from *",
		`update * set tags:list where tags`,
		`select * from a where x = "é"`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		_, _ = ParseProgram(strings.NewReader(src))
	})
}

func FuzzParseExpr(f *testing.F) {
	seeds := []string{
		"",
		"a",
		"a + b",
		"a = 1",
		"a and b or c",
		"not a",
		"a:int = 1",
		"-1.5e3",
		"0xFF",
		`"hello"`,
		`'''multi
line'''`,
		"`field with spaces` = \"x\"",
		"a + b * (c - d)",
		"tags:list",
		"foo:date >= 2020-01-01",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		_, _ = ParseExpr(strings.NewReader(src))
	})
}
