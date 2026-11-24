package internal

import (
	"strings"
	"testing"
)

func TestTypeCheckOrdering(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr string // substring; empty means the query must parse
	}{
		// Allowed: numeric and temporal pairs.
		{"int_vs_int", `select from * where n:int > 3`, ""},
		{"number_vs_bool", `select from * where n:numeric >= true`, ""},
		{"date_vs_date_string", `select from * where d:date > "2025-01-15"`, ""},
		{"datetime_vs_date_string", `select from * where d:datetime <= "2025-01-15T09:00:00"`, ""},
		{"date_vs_today", `select from * where d:date < today()`, ""},
		{"date_strings", `select from * where "2025-01-01" < "2025-02-01"`, ""},
		{"arith_vs_func", `select from * where n:int + 1 > length(title)`, ""},
		{"date_diff_vs_int", `select from * where date_diff(d, "2025-01-15") > 0`, ""},

		// Allowed: unknown operands are checked at runtime.
		{"untyped_vs_string", `select from * where title > "abc"`, ""},
		{"untyped_vs_date", `select from * where due > "2025-01-15"`, ""},
		{"string_func_vs_int", `select from * where regexp_extract(title, "\\d+") > 3`, ""},
		{"coalesce_vs_int", `select from * where coalesce(a, b) > 3`, ""},
		{"null_literal", `select from * where n:int > null`, ""},

		// Allowed: list membership.
		{"list_contains", `select from * where tags:list >= "x"`, ""},
		{"contained_in_list", `select from * where "x" <= tags:list`, ""},
		{"untyped_contains", `select from * where tags >= "x"`, ""},
		{"split_contains", `select from * where split(title, " ") >= "a"`, ""},

		// Errors.
		{"string_field_vs_int", `select from * where price:string > 3`, `field price:string > int 3`},
		{"string_vs_number_field", `select from * where price:string > cost:numeric`, `field price:string > field cost:numeric`},
		{"date_vs_int", `select from * where due:date > 5`, `field due:date > int 5`},
		{"date_vs_text", `select from * where due:date < "soon"`, `field due:date < string "soon"`},
		{"int_vs_date_string", `select from * where n:int < "2025-01-15"`, `field n:int < "2025-01-15"`},
		{"link_vs_int", `select from * where ref:link > 1`, `field ref:link > int 1`},
		{"list_lt", `select from * where tags:list < "x"`, `field tags:list < string "x"`},
		{"list_literal_vs_int", `select from * where [1, 2] < 3`, `list < int 3`},
		{"list_vs_list", `select from * where tags:list >= [1]`, `field tags:list >= list`},
		{"func_vs_text", `select from * where length(title) > "x"`, `length() > string "x"`},

		// Every clause is checked.
		{"in_select", `select n:int > "x" from *`, `field n:int > string "x"`},
		{"in_sort", `select from * sort by d:date > 1`, `field d:date > int 1`},
		{"in_set", `update * set ok = d:date > 1`, `field d:date > int 1`},
		{"in_func_arg", `select from * where coalesce(d:date > 1, false)`, `field d:date > int 1`},
		{"nested_not", `select from * where not (d:date > 1)`, `field d:date > int 1`},
		{"alter_where", `alter * drop x where d:date > 1`, `field d:date > int 1`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseProgram(strings.NewReader(tc.query))
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tc.wantErr != "" && err == nil:
				t.Fatalf("expected error containing %q, got none", tc.wantErr)
			case tc.wantErr != "" && !strings.Contains(err.Error(), tc.wantErr):
				t.Fatalf("error %q does not contain %q", err, tc.wantErr)
			}
		})
	}
}
