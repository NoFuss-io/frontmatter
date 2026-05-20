package internal

type Program struct {
	Stmts []Query
}

// Query is the top-level parsed result.
type Query interface{ query() }

// SelectQuery represents: select <exprs> from <globs> [where <expr>] [sort by <terms>] [limit <n>]
type SelectQuery struct {
	Fields []Expr
	From   []string
	Where  Expr // nil if absent
	SortBy []SortTerm
	Limit  int // 0 = no limit
}

func (SelectQuery) query() {}

// UpdateQuery represents: update <globs> set <assigns> [where <expr>]
type UpdateQuery struct {
	From  []string
	Set   []Assign
	Where Expr
}

func (UpdateQuery) query() {}

// AlterQuery represents: alter <globs> drop/rename ... [where <expr>]
type AlterQuery struct {
	From   []string
	Op     AlterOp
	Drop   []Field      // populated when Op == AlterDrop
	Rename []RenamePair // populated when Op == AlterRename
	Where  Expr
}

func (AlterQuery) query() {}

// AlterOp distinguishes the two forms of alter query.
type AlterOp int

const (
	AlterDrop AlterOp = iota
	AlterRename
)

// FieldType is the declared type of a frontmatter field.
type FieldType int

const (
	TypeAny      FieldType = iota // default when no type annotation is given
	TypeString                    // string
	TypeBool                      // bool
	TypeInt                       // int
	TypeNumber                    // numeric (float)
	TypeDate                      // date (YYYY-MM-DD)
	TypeDatetime                  // datetime (ISO 8601 local time)
	TypeLink                      // [[ref]] or [[ref|title]]
	TypeMdLink                    // [title](ref)
	TypeList                      // list[:elemtype]
)

func (f FieldType) String() string {
	switch f {
	case TypeBool:
		return "bool"
	case TypeInt:
		return "int"
	case TypeNumber:
		return "number"
	case TypeString:
		return "string"
	case TypeDate:
		return "date"
	case TypeDatetime:
		return "datetime"
	case TypeLink:
		return "link"
	case TypeMdLink:
		return "mdlink"
	case TypeList:
		return "list"
	case TypeAny:
		return "any"
	}
	return "unknown"
}

// Field is a field reference with an optional type annotation: identifier[:type].
type Field struct {
	Name string
	Type FieldType
}

// AssignOp is the operator used in an update...set assignment.
type AssignOp int

const (
	OpSet AssignOp = iota // =
	OpAdd                 // +=
	OpSub                 // -=
)

// RenamePair is one rename in an alter...rename clause.
type RenamePair struct {
	From string
	To   string
}

// SortTerm is one key in a sort-by clause.
type SortTerm struct {
	Expr Expr
	Desc bool
}

// Assign is one assignment in an update...set clause.
// Value is nil for the cast-only form (e.g. `set foo:int`).
type Assign struct {
	Field Field
	Op    AssignOp
	Value Expr
}

// --- Expression nodes ---

// Expr is the expression tree node interface.
type Expr interface {
	expr()
	Eval(fm *FrontMatter) Value
}

// BinExpr is a binary operation: Left Op Right.
type BinExpr struct {
	Op    BinOp
	Left  Expr
	Right Expr
}

func (BinExpr) expr() {}

// UnaryExpr is a unary operation: Op Operand.
type UnaryExpr struct {
	Op      UnaryOp
	Operand Expr
}

func (UnaryExpr) expr() {}

// FieldExpr is a field reference atom.
// Evaluates to the field's value in the current file's frontmatter, or void if absent.
type FieldExpr struct {
	Field Field
}

func (FieldExpr) expr() {}

// LitExpr is a literal constant atom.
// Value holds the raw source text: unquoted string content, digit string, "true", "false", or "null".
type LitExpr struct {
	Kind  LiteralKind
	Value string
}

func (LitExpr) expr() {}

// --- Operators ---

// BinOp is a binary operator, ordered by precedence group (lowest first).
type BinOp int

const (
	// Boolean connectives (lowest precedence).
	BinOr  BinOp = iota // or
	BinAnd              // and

	// Comparison operators.
	BinEq // =
	BinNe // !=
	BinLt // <
	BinLe // <=
	BinGt // >
	BinGe // >=

	// Arithmetic operators.
	BinAdd // +
	BinSub // -
	BinMul // *
	BinDiv // /
)

// UnaryOp is a unary operator.
type UnaryOp int

const (
	UnaryNot UnaryOp = iota // not  (boolean negation)
	UnaryNeg                // -    (arithmetic negation)
)

// LiteralKind identifies the kind of a LitExpr.
type LiteralKind int

const (
	LitString  LiteralKind = iota // quoted or raw string
	LitInt                        // decimal or 0x… integer
	LitNumeric                    // floating-point
	LitBool                       // true | false
	LitNull                       // null
)
