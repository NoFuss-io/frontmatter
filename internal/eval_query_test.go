package internal

import (
	"testing"
)

// ── Row 8: Assign.Apply ───────────────────────────────────────────────────────

func TestAssign_Apply(t *testing.T) {
	t.Run("set_literal", func(t *testing.T) {
		fm := FrontMatter{}
		a := Assign{
			Field: Field{Name: "title"},
			Op:    OpSet,
			Value: LitExpr{Kind: LitString, Value: "hello"},
		}
		if err := a.Apply(fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if fm["title"] != "hello" {
			t.Errorf("title = %v, want hello", fm["title"])
		}
	})
	t.Run("set_typed_cast_ok", func(t *testing.T) {
		fm := FrontMatter{}
		a := Assign{
			Field: Field{Name: "n", Type: TypeInt},
			Op:    OpSet,
			Value: LitExpr{Kind: LitString, Value: "42"},
		}
		if err := a.Apply(fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if fm["n"] != int64(42) {
			t.Errorf("n = %v (%T), want int64(42)", fm["n"], fm["n"])
		}
	})
	t.Run("cast_only_creates_null", func(t *testing.T) {
		fm := FrontMatter{}
		a := Assign{
			Field: Field{Name: "foo", Type: TypeInt},
			Op:    OpSet,
			Value: nil,
		}
		if err := a.Apply(fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if _, ok := fm["foo"]; !ok {
			t.Error("foo not created")
		}
		if fm["foo"] != nil {
			t.Errorf("foo = %v, want nil", fm["foo"])
		}
	})
	t.Run("set_null", func(t *testing.T) {
		fm := FrontMatter{"title": "hello"}
		a := Assign{
			Field: Field{Name: "title"},
			Op:    OpSet,
			Value: LitExpr{Kind: LitNull, Value: "null"},
		}
		if err := a.Apply(fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if fm["title"] != nil {
			t.Errorf("title = %v, want nil", fm["title"])
		}
	})
	t.Run("add_to_list", func(t *testing.T) {
		fm := FrontMatter{"tags": []any{"a"}}
		a := Assign{
			Field: Field{Name: "tags"},
			Op:    OpAdd,
			Value: LitExpr{Kind: LitString, Value: "b"},
		}
		if err := a.Apply(fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		tags, ok := fm["tags"].([]any)
		if !ok || len(tags) != 2 || tags[1] != "b" {
			t.Errorf("tags = %+v, want [a b]", fm["tags"])
		}
	})
	t.Run("sub_from_list", func(t *testing.T) {
		fm := FrontMatter{"tags": []any{"a", "b", "c"}}
		a := Assign{
			Field: Field{Name: "tags"},
			Op:    OpSub,
			Value: LitExpr{Kind: LitString, Value: "b"},
		}
		if err := a.Apply(fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		tags, _ := fm["tags"].([]any)
		if len(tags) != 2 || tags[0] != "a" || tags[1] != "c" {
			t.Errorf("tags = %+v, want [a c]", tags)
		}
	})
	t.Run("add_to_null_field", func(t *testing.T) {
		fm := FrontMatter{"tags": nil}
		a := Assign{
			Field: Field{Name: "tags"},
			Op:    OpAdd,
			Value: LitExpr{Kind: LitString, Value: "x"},
		}
		if err := a.Apply(fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		tags, ok := fm["tags"].([]any)
		if !ok || len(tags) != 1 || tags[0] != "x" {
			t.Errorf("tags = %+v, want [x]", fm["tags"])
		}
	})
	t.Run("sub_from_null_field", func(t *testing.T) {
		fm := FrontMatter{"tags": nil}
		a := Assign{
			Field: Field{Name: "tags"},
			Op:    OpSub,
			Value: LitExpr{Kind: LitString, Value: "x"},
		}
		if err := a.Apply(fm); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if fm["tags"] != nil {
			t.Errorf("tags = %+v, want nil", fm["tags"])
		}
	})
	t.Run("cast_failure_errors", func(t *testing.T) {
		fm := FrontMatter{"src": "hello"}
		a := Assign{
			Field: Field{Name: "n", Type: TypeInt},
			Op:    OpSet,
			Value: FieldExpr{Field: Field{Name: "src", Type: TypeString}},
		}
		if err := a.Apply(fm); err == nil {
			t.Error("expected cast-failure error, got nil")
		}
	})
}

// ── Row 10: AlterQuery.Apply ──────────────────────────────────────────────────

func TestAlterQuery_Apply(t *testing.T) {
	t.Run("drop_one", func(t *testing.T) {
		fm := FrontMatter{"title": "x", "date": "2026-01-01"}
		q := parseQ(t, "alter *.md drop title").(AlterQuery)
		if _, err := q.Eval(fm); err != nil {
			t.Fatal(err)
		}
		if _, ok := fm["title"]; ok {
			t.Error("title still present")
		}
		if _, ok := fm["date"]; !ok {
			t.Error("date should remain")
		}
	})
	t.Run("drop_multi", func(t *testing.T) {
		fm := FrontMatter{"a": int64(1), "b": int64(2), "c": int64(3)}
		q := parseQ(t, "alter *.md drop a, b").(AlterQuery)
		if _, err := q.Eval(fm); err != nil {
			t.Fatal(err)
		}
		if len(fm) != 1 || fm["c"] != int64(3) {
			t.Errorf("FrontMatter = %+v, want {c:3}", fm)
		}
	})
	t.Run("rename", func(t *testing.T) {
		fm := FrontMatter{"foo": "hello"}
		q := parseQ(t, "alter *.md rename foo to bar").(AlterQuery)
		if _, err := q.Eval(fm); err != nil {
			t.Fatal(err)
		}
		if _, ok := fm["foo"]; ok {
			t.Error("foo still present")
		}
		if fm["bar"] != "hello" {
			t.Errorf("bar = %v, want hello", fm["bar"])
		}
	})
	t.Run("where_skips", func(t *testing.T) {
		fm := FrontMatter{"published": false, "title": "x"}
		q := parseQ(t, "alter *.md drop title where published = true").(AlterQuery)
		if _, err := q.Eval(fm); err != nil {
			t.Fatal(err)
		}
		if _, ok := fm["title"]; !ok {
			t.Error("title should remain (where=false)")
		}
	})
}

// ── Row 11: UpdateQuery.Apply ─────────────────────────────────────────────────

func TestUpdateQuery_Apply(t *testing.T) {
	t.Run("set_one", func(t *testing.T) {
		fm := FrontMatter{}
		q := parseQ(t, `update *.md set title = "hello"`).(UpdateQuery)
		if _, err := q.Eval(fm); err != nil {
			t.Fatal(err)
		}
		if fm["title"] != "hello" {
			t.Errorf("title = %v", fm["title"])
		}
	})
	t.Run("set_multi", func(t *testing.T) {
		fm := FrontMatter{}
		q := parseQ(t, `update *.md set a = 1, b = 2`).(UpdateQuery)
		if _, err := q.Eval(fm); err != nil {
			t.Fatal(err)
		}
		if fm["a"] != int64(1) || fm["b"] != int64(2) {
			t.Errorf("FrontMatter = %+v", fm)
		}
	})
	t.Run("where_skips", func(t *testing.T) {
		fm := FrontMatter{"published": false}
		q := parseQ(t, `update *.md set title = "x" where published = true`).(UpdateQuery)
		if _, err := q.Eval(fm); err != nil {
			t.Fatal(err)
		}
		if _, ok := fm["title"]; ok {
			t.Error("title set despite where=false")
		}
	})
}

// ── Row 12: SelectQuery.Eval ──────────────────────────────────────────────────

func TestSelectQuery_Eval(t *testing.T) {
	t.Run("project_one", func(t *testing.T) {
		fm := FrontMatter{"title": "hello", "date": "2026-01-01"}
		q := parseQ(t, "select title from *.md").(SelectQuery)
		row, err := q.Eval(fm)
		if err != nil {
			t.Fatal(err)
		}
		if row == nil || len(row.print) != 1 || !valEq(row.print[0], vStr("hello")) {
			t.Errorf("row = %+v, want [hello]", row)
		}
	})
	t.Run("project_multi", func(t *testing.T) {
		fm := FrontMatter{"a": "x", "b": int64(42)}
		q := parseQ(t, "select a, b from *.md").(SelectQuery)
		row, err := q.Eval(fm)
		if err != nil {
			t.Fatal(err)
		}
		if row == nil || len(row.print) != 2 {
			t.Fatalf("row = %+v, want 2 cols", row)
		}
		if !valEq(row.print[0], vStr("x")) {
			t.Errorf("row[0] = %+v, want %+v", row.print[0], vStr("x"))
		}
		if !valEq(row.print[1], vInt(42)) {
			t.Errorf("row[1] = %+v, want %+v", row.print[1], vInt(42))
		}
	})
	t.Run("where_returns_nil", func(t *testing.T) {
		fm := FrontMatter{"published": false, "title": "x"}
		q := parseQ(t, "select title from *.md where published = true").(SelectQuery)
		row, err := q.Eval(fm)
		if err != nil {
			t.Fatal(err)
		}
		if row != nil {
			t.Errorf("row = %+v, want nil (where=false)", row)
		}
	})
	t.Run("missing_field_null", func(t *testing.T) {
		fm := FrontMatter{}
		q := parseQ(t, "select title from *.md").(SelectQuery)
		row, err := q.Eval(fm)
		if err != nil {
			t.Fatal(err)
		}
		if row == nil || len(row.print) != 1 || !row.print[0].Null {
			t.Errorf("row = %+v, want [null]", row)
		}
	})
	t.Run("zero_fields", func(t *testing.T) {
		fm := FrontMatter{"title": "hello"}
		q := parseQ(t, "select from *.md").(SelectQuery)
		row, err := q.Eval(fm)
		if err != nil {
			t.Fatal(err)
		}
		if row == nil || len(row.print) != 0 {
			t.Errorf("row = %+v, want empty slice", row)
		}
	})
}
