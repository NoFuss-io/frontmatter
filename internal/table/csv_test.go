package table

import (
	"strings"
	"testing"
)

func TestCSVRender(t *testing.T) {
	cases := []struct {
		name string
		tbl  Table
		want string
	}{
		{
			name: "zero rows",
			tbl:  Table{Headers: []string{"name", "age"}, Rows: nil},
			want: `name,age
`,
		},
		{
			name: "one row",
			tbl:  Table{Headers: []string{"name", "age"}, Rows: [][]string{{"alice", "30"}}},
			want: `name,age
alice,30
`,
		},
		{
			name: "two rows",
			tbl: Table{
				Headers: []string{"name", "age"},
				Rows:    [][]string{{"alice", "30"}, {"bob", "25"}},
			},
			want: `name,age
alice,30
bob,25
`,
		},
		{
			name: "escapes commas and quotes",
			tbl: Table{
				Headers: []string{"note"},
				Rows:    [][]string{{"say \"hi\""}, {"a,b"}},
			},
			want: `note
"say ""hi"""
"a,b"
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf strings.Builder
			err := CSV{}.Render(tc.tbl, &buf)
			if err != nil {
				t.Fatalf("Render error: %v", err)
			}
			got := buf.String()
			if got != tc.want {
				t.Errorf("got:\n%q\nwant:\n%q", got, tc.want)
			}
		})
	}
}
