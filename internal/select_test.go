package internal

import (
	"io"
	"testing"

	tablepkg "github.com/nofuss-io/frontmatter/internal/table"
)

// captureRenderer records the Table passed to Render so select tests can
// compare output without going through a text format.
type captureRenderer struct {
	got tablepkg.Table
}

func (r *captureRenderer) Render(t tablepkg.Table, _ io.Writer) error {
	r.got = t
	return nil
}

// testDoc is a (path, frontmatter) pair used as select test input.
type testDoc struct {
	path string
	fm   FrontMatter
}

// Four documents shared by all select cases below.
var selectDocs = []testDoc{
	{"a.md", FrontMatter{"title": "Alpha", "published": true, "rating": int64(5)}},
	{"b.md", FrontMatter{"title": "Beta", "published": false, "rating": int64(3)}},
	{"c.md", FrontMatter{"title": "Gamma", "published": true, "rating": int64(4)}},
	{"d.md", FrontMatter{"title": "Delta", "published": true, "rating": int64(2)}},
}

// runSelect drives q against docs through the Output pipeline and returns
// the Table the renderer received.
func runSelect(t *testing.T, q SelectQuery, docs []testDoc) tablepkg.Table {
	t.Helper()
	cap := &captureRenderer{}
	prog := Program{Stmts: []Query{q}}
	out := NewOutput(&prog, io.Discard, 20, cap)
	for _, d := range docs {
		if out.AllDone() {
			break
		}
		row, err := q.Eval(d.fm)
		if err != nil {
			t.Fatalf("Eval(%s): %v", d.path, err)
		}
		if row != nil {
			out.Append(d.path, 0, row)
		}
	}
	out.Finalize()
	out.Print(io.Discard)
	return cap.got
}

func tableEq(a, b tablepkg.Table) bool {
	if len(a.Headers) != len(b.Headers) {
		return false
	}
	for i, h := range a.Headers {
		if h != b.Headers[i] {
			return false
		}
	}
	if len(a.Rows) != len(b.Rows) {
		return false
	}
	for i, ar := range a.Rows {
		if len(ar) != len(b.Rows[i]) {
			return false
		}
		for j, c := range ar {
			if c != b.Rows[i][j] {
				return false
			}
		}
	}
	return true
}

func TestSelectQuery_Output(t *testing.T) {
	titleField := FieldExpr{Field: Field{Name: "title"}}
	ratingField := FieldExpr{Field: Field{Name: "rating"}}
	publishedTrue := BinExpr{
		Op:    BinEq,
		Left:  FieldExpr{Field: Field{Name: "published"}},
		Right: LitExpr{Kind: LitBool, Value: "true"},
	}

	cases := []struct {
		name string
		q    SelectQuery
		want tablepkg.Table
	}{
		{
			name: "all rows",
			q: SelectQuery{
				Select: []Expr{titleField},
				From:   []FilePath{"*.md"},
			},
			want: tablepkg.Table{
				Headers: []string{"filename", "title"},
				Rows: [][]string{
					{"a.md", "Alpha"},
					{"b.md", "Beta"},
					{"c.md", "Gamma"},
					{"d.md", "Delta"},
				},
			},
		},
		{
			name: "where published = true",
			q: SelectQuery{
				Select: []Expr{titleField},
				From:   []FilePath{"*.md"},
				Where:  publishedTrue,
			},
			want: tablepkg.Table{
				Headers: []string{"filename", "title"},
				Rows: [][]string{
					{"a.md", "Alpha"},
					{"c.md", "Gamma"},
					{"d.md", "Delta"},
				},
			},
		},
		{
			name: "limit 2",
			q: SelectQuery{
				Select: []Expr{titleField},
				From:   []FilePath{"*.md"},
				Limit:  2,
			},
			want: tablepkg.Table{
				Headers: []string{"filename", "title"},
				Rows: [][]string{
					{"a.md", "Alpha"},
					{"b.md", "Beta"},
				},
			},
		},
		{
			name: "sort by rating desc",
			q: SelectQuery{
				Select: []Expr{titleField},
				From:   []FilePath{"*.md"},
				SortBy: []SortTerm{{Expr: ratingField, Desc: true}},
			},
			want: tablepkg.Table{
				Headers: []string{"filename", "title"},
				Rows: [][]string{
					{"a.md", "Alpha"}, // rating 5
					{"c.md", "Gamma"}, // rating 4
					{"b.md", "Beta"},  // rating 3
					{"d.md", "Delta"}, // rating 2
				},
			},
		},
		{
			name: "sort by rating desc limit 2",
			q: SelectQuery{
				Select: []Expr{titleField},
				From:   []FilePath{"*.md"},
				SortBy: []SortTerm{{Expr: ratingField, Desc: true}},
				Limit:  2,
			},
			want: tablepkg.Table{
				Headers: []string{"filename", "title"},
				Rows: [][]string{
					{"a.md", "Alpha"},
					{"c.md", "Gamma"},
				},
			},
		},
		{
			name: "where published limit 2",
			q: SelectQuery{
				Select: []Expr{titleField},
				From:   []FilePath{"*.md"},
				Where:  publishedTrue,
				Limit:  2,
			},
			want: tablepkg.Table{
				Headers: []string{"filename", "title"},
				Rows: [][]string{
					{"a.md", "Alpha"},
					{"c.md", "Gamma"},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runSelect(t, tc.q, selectDocs)
			if !tableEq(got, tc.want) {
				t.Errorf("headers: got %v, want %v", got.Headers, tc.want.Headers)
				t.Errorf("rows got (%d):", len(got.Rows))
				for _, r := range got.Rows {
					t.Errorf("  %v", r)
				}
				t.Errorf("rows want (%d):", len(tc.want.Rows))
				for _, r := range tc.want.Rows {
					t.Errorf("  %v", r)
				}
			}
		})
	}
}
