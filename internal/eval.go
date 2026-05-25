package internal

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Value is the runtime value produced by evaluating an expression.
// Null marks absence: a missing field or a cast that could not produce a value.
// Per Manual.md: null propagates through arithmetic; in boolean context it is falsey.
type Value struct {
	Kind FieldType
	Data any
	Null bool
}

// String returns a formatted representation of v like bool(true) or string("haha").
func (v Value) String() string {
	if v.Null {
		return "null"
	}
	switch v.Kind {
	case TypeBool, TypeAny:
		return fmt.Sprintf("%s(%v)", v.Kind.String(), v.Data)
	case TypeInt:
		return fmt.Sprintf("%s(%d)", v.Kind.String(), v.Data)
	case TypeNumber:
		return fmt.Sprintf("%s(%g)", v.Kind.String(), v.Data)
	case TypeString:
		return fmt.Sprintf("%s(%q)", v.Kind.String(), v.Data)
	case TypeDate:
		return fmt.Sprintf("%s(%q)", v.Kind.String(), v.Data.(time.Time).Format("2006-01-02"))
	case TypeDatetime:
		return fmt.Sprintf("%s(%q)", v.Kind.String(), v.Data.(time.Time).Format("2006-01-02T15:04:05"))
	case TypeLink, TypeMdLink:
		return fmt.Sprintf("%s(%q)", v.Kind.String(), v.Data)
	case TypeList:
		els := v.Data.([]Value)
		parts := make([]string, len(els))
		for i, e := range els {
			parts[i] = e.String()
		}
		return fmt.Sprintf("%s(%v)", v.Kind.String(), parts)
	}
	return "unknown"
}

// Cast converts v to target. Returns error if conversion is not possible.
func Cast(v Value, target FieldType) (Value, error) {
	if v.Null {
		return Value{}, fmt.Errorf("cannot cast null to %s", target)
	}
	if target == TypeAny {
		return v, nil
	}
	if target == TypeList {
		var elems []Value
		if v.Kind == TypeList {
			elems = v.Data.([]Value)
		} else {
			elems = []Value{v}
		}
		out := make([]Value, len(elems))
		for i, e := range elems {
			s, err := Cast(e, TypeString)
			if err != nil {
				return Value{}, fmt.Errorf("list element %d: %w", i, err)
			}
			out[i] = s
		}
		return Value{Kind: TypeList, Data: out}, nil
	}
	if target == v.Kind {
		return v, nil
	}
	if v.Kind == TypeList {
		els, ok := v.Data.([]Value)
		if !ok || len(els) != 1 {
			return Value{}, fmt.Errorf("cannot cast list with %d elements to %s", len(els), target)
		}
		return Cast(els[0], target)
	}
	switch target {
	case TypeBool:
		return castToBool(v)
	case TypeInt:
		return castToInt(v)
	case TypeNumber:
		return castToNumber(v)
	case TypeString:
		return castToString(v)
	case TypeDate:
		return castToDate(v)
	case TypeDatetime:
		return castToDatetime(v)
	case TypeLink:
		return castToLink(v)
	case TypeMdLink:
		return castToMdLink(v)
	}
	return Value{}, fmt.Errorf("unknown target type %v", target)
}

func castToBool(v Value) (Value, error) {
	switch v.Kind {
	case TypeInt:
		switch v.Data.(int64) {
		case 0:
			return Value{Kind: TypeBool, Data: false}, nil
		case 1:
			return Value{Kind: TypeBool, Data: true}, nil
		}
	case TypeNumber:
		switch v.Data.(float64) {
		case 0:
			return Value{Kind: TypeBool, Data: false}, nil
		case 1:
			return Value{Kind: TypeBool, Data: true}, nil
		}
	case TypeString:
		switch strings.ToLower(v.Data.(string)) {
		case "true":
			return Value{Kind: TypeBool, Data: true}, nil
		case "false":
			return Value{Kind: TypeBool, Data: false}, nil
		}
	}
	return Value{}, fmt.Errorf("cannot cast %s to bool", v.Kind)
}

func castToInt(v Value) (Value, error) {
	switch v.Kind {
	case TypeBool:
		if v.Data.(bool) {
			return Value{Kind: TypeInt, Data: int64(1)}, nil
		}
		return Value{Kind: TypeInt, Data: int64(0)}, nil
	case TypeNumber:
		f := v.Data.(float64)
		if f != math.Trunc(f) {
			return Value{}, fmt.Errorf("cannot cast %g to int: not a whole number", f)
		}
		return Value{Kind: TypeInt, Data: int64(f)}, nil
	case TypeString:
		n, err := strconv.ParseInt(v.Data.(string), 0, 64)
		if err != nil {
			return Value{}, fmt.Errorf("cannot cast string %q to int: %w", v.Data.(string), err)
		}
		return Value{Kind: TypeInt, Data: n}, nil
	}
	return Value{}, fmt.Errorf("cannot cast %s to int", v.Kind)
}

func castToNumber(v Value) (Value, error) {
	switch v.Kind {
	case TypeInt:
		return Value{Kind: TypeNumber, Data: float64(v.Data.(int64))}, nil
	case TypeBool:
		if v.Data.(bool) {
			return Value{Kind: TypeNumber, Data: float64(1)}, nil
		}
		return Value{Kind: TypeNumber, Data: float64(0)}, nil
	case TypeString:
		f, err := strconv.ParseFloat(v.Data.(string), 64)
		if err != nil {
			return Value{}, fmt.Errorf("cannot cast string %q to number: %w", v.Data.(string), err)
		}
		return Value{Kind: TypeNumber, Data: f}, nil
	}
	return Value{}, fmt.Errorf("cannot cast %s to number", v.Kind)
}

func castToString(v Value) (Value, error) {
	switch v.Kind {
	case TypeInt:
		return Value{Kind: TypeString, Data: strconv.FormatInt(v.Data.(int64), 10)}, nil
	case TypeNumber:
		return Value{Kind: TypeString, Data: strconv.FormatFloat(v.Data.(float64), 'g', -1, 64)}, nil
	case TypeBool:
		if v.Data.(bool) {
			return Value{Kind: TypeString, Data: "true"}, nil
		}
		return Value{Kind: TypeString, Data: "false"}, nil
	case TypeDate:
		return Value{Kind: TypeString, Data: v.Data.(time.Time).Format("2006-01-02")}, nil
	case TypeDatetime:
		return Value{Kind: TypeString, Data: v.Data.(time.Time).Format("2006-01-02T15:04:05")}, nil
	case TypeLink, TypeMdLink:
		return Value{Kind: TypeString, Data: v.Data.(string)}, nil
	}
	return Value{}, fmt.Errorf("cannot cast %s to string", v.Kind)
}

func castToDate(v Value) (Value, error) {
	switch v.Kind {
	case TypeString:
		t, err := time.Parse("2006-01-02", v.Data.(string))
		if err != nil {
			return Value{}, fmt.Errorf("cannot cast string %q to date: %w", v.Data.(string), err)
		}
		return Value{Kind: TypeDate, Data: t}, nil
	}
	return Value{}, fmt.Errorf("cannot cast %s to date", v.Kind)
}

func castToDatetime(v Value) (Value, error) {
	switch v.Kind {
	case TypeString:
		t, err := time.Parse("2006-01-02T15:04:05", v.Data.(string))
		if err != nil {
			return Value{}, fmt.Errorf("cannot cast string %q to datetime: %w", v.Data.(string), err)
		}
		return Value{Kind: TypeDatetime, Data: t}, nil
	}
	return Value{}, fmt.Errorf("cannot cast %s to datetime", v.Kind)
}

// parseWikiLink parses [[ref]] or [[ref|title]], returns ref, title, ok.
func parseWikiLink(s string) (ref, title string, ok bool) {
	if !strings.HasPrefix(s, "[[") || !strings.HasSuffix(s, "]]") || len(s) < 5 {
		return "", "", false
	}
	inner := s[2 : len(s)-2]
	if inner == "" {
		return "", "", false
	}
	if i := strings.IndexByte(inner, '|'); i >= 0 {
		return inner[:i], inner[i+1:], true
	}
	return inner, "", true
}

// parseMdLink parses [title](ref), returns ref, title, ok.
func parseMdLink(s string) (ref, title string, ok bool) {
	if len(s) < 4 || s[0] != '[' {
		return "", "", false
	}
	i := strings.Index(s, "](")
	if i < 0 || s[len(s)-1] != ')' {
		return "", "", false
	}
	return s[i+2 : len(s)-1], s[1:i], true
}

func castToLink(v Value) (Value, error) {
	switch v.Kind {
	case TypeLink:
		return v, nil
	case TypeMdLink:
		ref, title, _ := parseMdLink(v.Data.(string))
		if title == "" || title == ref {
			return Value{Kind: TypeLink, Data: fmt.Sprintf("[[%s]]", ref)}, nil
		}
		return Value{Kind: TypeLink, Data: fmt.Sprintf("[[%s|%s]]", ref, title)}, nil
	case TypeString:
		s := v.Data.(string)
		if _, _, ok := parseWikiLink(s); ok {
			return Value{Kind: TypeLink, Data: s}, nil
		}
		ref, title, ok := parseMdLink(s)
		if !ok {
			return Value{}, fmt.Errorf("cannot cast string %q to link: invalid format", s)
		}
		if title == "" || title == ref {
			return Value{Kind: TypeLink, Data: fmt.Sprintf("[[%s]]", ref)}, nil
		}
		return Value{Kind: TypeLink, Data: fmt.Sprintf("[[%s|%s]]", ref, title)}, nil
	}
	return Value{}, fmt.Errorf("cannot cast %s to link", v.Kind)
}

func castToMdLink(v Value) (Value, error) {
	switch v.Kind {
	case TypeMdLink:
		return v, nil
	case TypeLink:
		ref, title, _ := parseWikiLink(v.Data.(string))
		if title == "" {
			return Value{Kind: TypeMdLink, Data: fmt.Sprintf("[%s](%s)", ref, ref)}, nil
		}
		return Value{Kind: TypeMdLink, Data: fmt.Sprintf("[%s](%s)", title, ref)}, nil
	case TypeString:
		s := v.Data.(string)
		if _, _, ok := parseMdLink(s); ok {
			return Value{Kind: TypeMdLink, Data: s}, nil
		}
		ref, title, ok := parseWikiLink(s)
		if !ok {
			return Value{}, fmt.Errorf("cannot cast string %q to mdlink: invalid format", s)
		}
		if title == "" {
			return Value{Kind: TypeMdLink, Data: fmt.Sprintf("[%s](%s)", ref, ref)}, nil
		}
		return Value{Kind: TypeMdLink, Data: fmt.Sprintf("[%s](%s)", title, ref)}, nil
	}
	return Value{}, fmt.Errorf("cannot cast %s to mdlink", v.Kind)
}

// ── Expressions ───────────────────────────────────────────────────────────────

func (e LitExpr) Eval(_ FrontMatter) Value {
	switch e.Kind {
	case LitInt:
		n, err := strconv.ParseInt(e.Value, 0, 64)
		if err != nil {
			return Value{Null: true}
		}
		return Value{Kind: TypeInt, Data: n}
	case LitNumeric:
		f, err := strconv.ParseFloat(e.Value, 64)
		if err != nil {
			return Value{Null: true}
		}
		return Value{Kind: TypeNumber, Data: f}
	case LitString:
		return Value{Kind: TypeString, Data: e.Value}
	case LitBool:
		return Value{Kind: TypeBool, Data: strings.ToLower(e.Value) == "true"}
	case LitNull:
		return Value{Null: true}
	}
	return Value{Null: true}
}
func (e ListExpr) Eval(fm FrontMatter) Value {
	els := make([]Value, len(e.Elems))
	for i, ex := range e.Elems {
		els[i] = ex.Eval(fm)
	}
	return Value{Kind: TypeList, Data: els}
}

func (e FieldExpr) Eval(fm FrontMatter) Value {
	if fm == nil {
		return Value{Null: true}
	}
	raw, ok := (fm)[e.Field.Name]
	if !ok {
		return Value{Null: true}
	}
	v := valueFromAny(raw)
	if e.Field.Type == TypeAny {
		return v
	}
	c, err := Cast(v, e.Field.Type)
	if err != nil {
		return Value{Null: true}
	}
	return c
}

// valueFromAny converts a raw Go value (from YAML-decoded frontmatter) into a
// typed Value. Unknown Go types fall through as TypeAny with Data preserved.
func valueFromAny(x any) Value {
	if x == nil {
		return Value{Null: true}
	}
	switch v := x.(type) {
	case string:
		return Value{Kind: TypeString, Data: v}
	case bool:
		return Value{Kind: TypeBool, Data: v}
	case int:
		return Value{Kind: TypeInt, Data: int64(v)}
	case int64:
		return Value{Kind: TypeInt, Data: v}
	case float64:
		return Value{Kind: TypeNumber, Data: v}
	case float32:
		return Value{Kind: TypeNumber, Data: float64(v)}
	case []any:
		els := make([]Value, len(v))
		for i, e := range v {
			els[i] = valueFromAny(e)
		}
		return Value{Kind: TypeList, Data: els}
	case time.Time:
		if v.Hour() == 0 && v.Minute() == 0 && v.Second() == 0 && v.Nanosecond() == 0 {
			return Value{Kind: TypeDate, Data: v}
		}
		return Value{Kind: TypeDatetime, Data: v}
	}
	return Value{Kind: TypeAny, Data: x}
}
func (e UnaryExpr) Eval(fm FrontMatter) Value {
	v := e.Operand.Eval(fm)
	switch e.Op {
	case UnaryNot:
		return Value{Kind: TypeBool, Data: !truthy(v)}
	case UnaryNeg:
		if v.Null {
			return Value{Null: true}
		}
		switch v.Kind {
		case TypeInt:
			return Value{Kind: TypeInt, Data: -v.Data.(int64)}
		case TypeNumber:
			return Value{Kind: TypeNumber, Data: -v.Data.(float64)}
		}
	}
	return Value{Null: true}
}

// truthy returns the boolean view of v. Null is falsey; non-null uses type-native
// semantics rather than strict bool casting, so bare field references act as
// existence checks.
func truthy(v Value) bool {
	if v.Null {
		return false
	}
	switch v.Kind {
	case TypeBool:
		return v.Data.(bool)
	case TypeInt:
		return v.Data.(int64) != 0
	case TypeNumber:
		return v.Data.(float64) != 0
	case TypeString:
		s := v.Data.(string)
		return s != "" && strings.ToLower(s) != "false"
	default:
		return true
	}
}
func (e BinExpr) Eval(fm FrontMatter) Value {
	switch e.Op {
	case BinAnd:
		return Value{Kind: TypeBool, Data: truthy(e.Left.Eval(fm)) && truthy(e.Right.Eval(fm))}
	case BinOr:
		return Value{Kind: TypeBool, Data: truthy(e.Left.Eval(fm)) || truthy(e.Right.Eval(fm))}
	case BinAdd, BinSub, BinMul, BinDiv:
		return arith(e.Op, e.Left.Eval(fm), e.Right.Eval(fm))
	case BinEq, BinNe, BinLt, BinLe, BinGt, BinGe, BinOverlap:
		return compare(e.Op, e.Left.Eval(fm), e.Right.Eval(fm))
	}
	return Value{Null: true}
}

// compare implements =, !=, <, <=, >, >=.
// null = null → true; null != null → false; null = non-null → false; null != non-null → true.
// Cast failures also follow null semantics.
// Lists participate in set equality (= / !=) or set membership (list >= scalar,
// scalar <= list); ordering operators require numeric coercion of both sides.
func compare(op BinOp, l, r Value) Value {
	if l.Null && r.Null {
		return Value{Kind: TypeBool, Data: op == BinEq}
	}
	if l.Null || r.Null {
		return Value{Kind: TypeBool, Data: op == BinNe}
	}
	if l.Kind == TypeList || r.Kind == TypeList {
		return compareList(op, l, r)
	}
	switch op {
	case BinEq:
		return Value{Kind: TypeBool, Data: scalarEq(l, r)}
	case BinNe:
		return Value{Kind: TypeBool, Data: !scalarEq(l, r)}
	case BinOverlap:
		return Value{Kind: TypeBool, Data: scalarEq(l, r)}
	}
	lf, errL := Cast(l, TypeNumber)
	rf, errR := Cast(r, TypeNumber)
	if errL != nil || errR != nil || lf.Null || rf.Null {
		return Value{Kind: TypeBool, Data: false}
	}
	a, b := lf.Data.(float64), rf.Data.(float64)
	var result bool
	switch op {
	case BinLt:
		result = a < b
	case BinLe:
		result = a <= b
	case BinGt:
		result = a > b
	case BinGe:
		result = a >= b
	}
	return Value{Kind: TypeBool, Data: result}
}

func scalarEq(l, r Value) bool {
	if l.Kind == r.Kind {
		if l.Kind == TypeDate || l.Kind == TypeDatetime {
			return l.Data.(time.Time).Equal(r.Data.(time.Time))
		}
		return l.Data == r.Data
	}
	lf, errL := Cast(l, TypeNumber)
	rf, errR := Cast(r, TypeNumber)
	if errL == nil && errR == nil && !lf.Null && !rf.Null {
		return lf.Data.(float64) == rf.Data.(float64)
	}
	ls, errL := Cast(l, TypeString)
	rs, errR := Cast(r, TypeString)
	if errL == nil && errR == nil && !ls.Null && !rs.Null {
		return ls.Data.(string) == rs.Data.(string)
	}
	return false
}

func compareList(op BinOp, l, r Value) Value {
	switch {
	case l.Kind == TypeList && r.Kind == TypeList:
		la, rb := l.Data.([]Value), r.Data.([]Value)
		switch op {
		case BinEq:
			return Value{Kind: TypeBool, Data: listSetEq(la, rb)}
		case BinNe:
			return Value{Kind: TypeBool, Data: !listSetEq(la, rb)}
		case BinOverlap:
			return Value{Kind: TypeBool, Data: listOverlap(la, rb)}
		}
	case l.Kind == TypeList && op == BinGe:
		return Value{Kind: TypeBool, Data: listContains(l.Data.([]Value), r)}
	case r.Kind == TypeList && op == BinLe:
		return Value{Kind: TypeBool, Data: listContains(r.Data.([]Value), l)}
	case l.Kind == TypeList && op == BinOverlap:
		return Value{Kind: TypeBool, Data: listContains(l.Data.([]Value), r)}
	case r.Kind == TypeList && op == BinOverlap:
		return Value{Kind: TypeBool, Data: listContains(r.Data.([]Value), l)}
	}
	return Value{Kind: TypeBool, Data: false}
}

func listOverlap(a, b []Value) bool {
	for _, x := range a {
		for _, y := range b {
			if scalarEq(x, y) {
				return true
			}
		}
	}
	return false
}

func listSetEq(a, b []Value) bool {
	if len(a) != len(b) {
		return false
	}
	used := make([]bool, len(b))
	for _, x := range a {
		found := false
		for i, y := range b {
			if !used[i] && scalarEq(x, y) {
				used[i] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func listContains(list []Value, v Value) bool {
	for _, x := range list {
		if scalarEq(x, v) {
			return true
		}
	}
	return false
}

// arith performs +, -, *, / with numeric coercion. Both ints stays int unless
// division has a non-zero remainder, in which case it promotes to number.
// Any null operand, cast failure, or division by zero → null.
func arith(op BinOp, l, r Value) Value {
	if l.Null || r.Null {
		return Value{Null: true}
	}
	if l.Kind == TypeInt && r.Kind == TypeInt {
		a, b := l.Data.(int64), r.Data.(int64)
		var n int64
		switch op {
		case BinAdd:
			n = a + b
		case BinSub:
			n = a - b
		case BinMul:
			n = a * b
		case BinDiv:
			if b == 0 {
				return Value{Null: true}
			}
			if a%b != 0 {
				return Value{Kind: TypeNumber, Data: float64(a) / float64(b)}
			}
			n = a / b
		}
		return Value{Kind: TypeInt, Data: n}
	}
	lf, errL := Cast(l, TypeNumber)
	rf, errR := Cast(r, TypeNumber)
	if errL != nil || errR != nil || lf.Null || rf.Null {
		return Value{Null: true}
	}
	a, b := lf.Data.(float64), rf.Data.(float64)
	var f float64
	switch op {
	case BinAdd:
		f = a + b
	case BinSub:
		f = a - b
	case BinMul:
		f = a * b
	case BinDiv:
		if b == 0 {
			return Value{Null: true}
		}
		f = a / b
	}
	return Value{Kind: TypeNumber, Data: f}
}

// SortTerm.Eval is a thin wrapper around the underlying expression.
func (s SortTerm) Eval(fm FrontMatter) Value { return s.Expr.Eval(fm) }

// ── Assignment ────────────────────────────────────────────────────────────────

// Apply evaluates a.Value (if present), casts to a.Field.Type, and writes the
// result into fm under a.Field.Name. A runtime cast failure is returned as
// an error so the caller can halt the current file per Manual.md.
func (a Assign) Apply(fm FrontMatter) error {
	if fm == nil {
		return errors.New("nil frontmatter")
	}
	name := a.Field.Name

	if a.Value == nil {
		// Cast-only form: ensure field exists; cast if it does.
		cur, ok := (fm)[name]
		if !ok {
			(fm)[name] = nil
			return nil
		}
		if a.Field.Type == TypeAny {
			return nil
		}
		v := valueFromAny(cur)
		c, err := Cast(v, a.Field.Type)
		if err != nil {
			return fmt.Errorf("cannot cast field %q: %w", name, err)
		}
		(fm)[name] = anyFromValue(c)
		return nil
	}

	v := a.Value.Eval(fm)
	if a.Field.Type != TypeAny {
		c, err := Cast(v, a.Field.Type)
		if err != nil {
			return fmt.Errorf("cannot cast value to %v for field %q: %w", a.Field.Type, name, err)
		}
		v = c
	}

	switch a.Op {
	case OpSet:
		(fm)[name] = anyFromValue(v)
	case OpAdd:
		return applyListAdd(fm, name, v)
	case OpSub:
		return applyListSub(fm, name, v)
	}
	return nil
}

// anyFromValue lowers a Value into the raw Go value used in FrontMatter maps.
func anyFromValue(v Value) any {
	if v.Null {
		return nil
	}
	if v.Kind == TypeList {
		els := v.Data.([]Value)
		out := make([]any, len(els))
		for i, e := range els {
			out[i] = anyFromValue(e)
		}
		return out
	}
	return v.Data
}

func applyListAdd(fm FrontMatter, name string, v Value) error {
	cur, ok := (fm)[name]
	var list []any
	if ok && cur != nil {
		if l, isList := cur.([]any); isList {
			list = l
		} else {
			list = []any{cur}
		}
	}
	var toAdd []Value
	if v.Kind == TypeList {
		toAdd = v.Data.([]Value)
	} else {
		toAdd = []Value{v}
	}
	for _, e := range toAdd {
		dup := false
		for _, x := range list {
			if scalarEq(valueFromAny(x), e) {
				dup = true
				break
			}
		}
		if !dup {
			list = append(list, anyFromValue(e))
		}
	}
	(fm)[name] = list
	return nil
}

func applyListSub(fm FrontMatter, name string, v Value) error {
	cur, ok := (fm)[name]
	if !ok || cur == nil {
		return nil
	}
	list, isList := cur.([]any)
	if !isList {
		return nil
	}
	var toRemove []Value
	if v.Kind == TypeList {
		toRemove = v.Data.([]Value)
	} else {
		toRemove = []Value{v}
	}
	out := make([]any, 0, len(list))
	for _, x := range list {
		xv := valueFromAny(x)
		drop := false
		for _, rm := range toRemove {
			if scalarEq(xv, rm) {
				drop = true
				break
			}
		}
		if !drop {
			out = append(out, x)
		}
	}
	(fm)[name] = out
	return nil
}
