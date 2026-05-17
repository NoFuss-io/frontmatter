# Parse implementation order

| # | Test                    | Complexity | Why                                                                 |
|--:|-------------------------|------------|---------------------------------------------------------------------|
| 1 | TestField_Parse         | Trivial    | Lexer only: identifier chars, backtick quoting, colon+type lookup   |
| 2 | TestRenamePair_Parse    | Trivial    | Two Field.Parse calls + "to" keyword check                          |
| 3 | TestLitExpr_Parse       | Simple     | First-char dispatch; string escaping adds edge cases, no recursion  |
| 4 | TestParseExpr           | Medium     | Recursive descent with precedence levels; core grammar challenge    |
| 5 | TestSortTerm_Parse      | Medium     | ParseExpr + optional asc/desc keyword; depends on #4               |
| 6 | TestAssign_Parse        | Medium     | Field + op + ParseExpr + static type check; depends on #1 and #4   |
| 7 | TestAlterQuery_Parse    | Hard       | Globs, drop/rename dispatch, Field list or RenamePair list, where   |
| 8 | TestUpdateQuery_Parse   | Hard       | Globs, set clause with Assign list, optional where                  |
| 9 | TestSelectQuery_Parse   | Hard       | Most clauses: field list, globs, where, sort-by list, limit         |
|10 | TestParseQuery          | Hard       | Trivial own logic (keyword dispatch), but requires all 9 above done |

Build order: 1 → 3 → 2 → 4 → 5+6 → 7+8+9 → 10
