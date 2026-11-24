package internal

import (
	"fmt"
	"strconv"
	"time"
)

// Static type checking of ordering comparisons (<, <=, >, >=).
//
// Each operand gets a static type class where one is known from the query alone:
// a typed field, a literal, an arithmetic or boolean expression, or a function
// with a fixed return type. Untyped fields and functions whose result depends on
// their arguments are unknown and always pass; their values are checked at
// runtime, where a failed comparison evaluates as false.

type staticClass int

const (
	classUnknown  staticClass = iota
	classNumeric              // bool, int, number
	classTemporal             // date, datetime, or a date-formatted string literal
	classString               // string, link, mdlink, or a non-date string literal
	classList
)

// staticType describes an operand for checking and for error messages.
type staticType struct {
	class staticClass
	desc  string // e.g. `int`, `string "soon"`, `field title:string`
}

// functionClass is the result class of built-in functions with a fixed return type.
// String-returning functions are deliberately absent (unknown): their result may
// hold a number or date (e.g. regexp_extract), and there is no way to cast an
// expression, so ordering on them is left to runtime coercion.
var functionClass = map[string]staticClass{
	"length": classNumeric, "ceil": classNumeric, "floor": classNumeric,
	"round": classNumeric, "mod": classNumeric, "sqrt": classNumeric,
	"pow": classNumeric, "array_length": classNumeric, "year": classNumeric,
	"month": classNumeric, "day": classNumeric, "date_diff": classNumeric,
	"starts_with": classNumeric, "ends_with": classNumeric,
	"contains_substr": classNumeric, "regexp_contains": classNumeric,
	"array_contains": classNumeric,

	"today": classTemporal,

	"split": classList, "array_concat": classList, "distinct": classList,
}

func classOfType(t FieldType) staticClass {
	switch t {
	case TypeBool, TypeInt, TypeNumber:
		return classNumeric
	case TypeDate, TypeDatetime:
		return classTemporal
	case TypeString, TypeLink, TypeMdLink:
		return classString
	case TypeList:
		return classList
	}
	return classUnknown
}

func typeOf(e Expr) staticType {
	switch e := e.(type) {
	case FieldExpr:
		if e.Field.Type == TypeAny {
			return staticType{classUnknown, "field " + e.Field.Name}
		}
		return staticType{classOfType(e.Field.Type), fmt.Sprintf("field %s:%s", e.Field.Name, typeKeyword(e.Field.Type))}
	case LitExpr:
		switch e.Kind {
		case LitInt:
			return staticType{classNumeric, "int " + e.Value}
		case LitNumeric:
			return staticType{classNumeric, "number " + e.Value}
		case LitBool:
			return staticType{classNumeric, "bool " + e.Value}
		case LitString:
			if isDateString(e.Value) {
				return staticType{classTemporal, strconv.Quote(e.Value)}
			}
			return staticType{classString, "string " + strconv.Quote(e.Value)}
		}
		return staticType{classUnknown, "null"}
	case ListExpr:
		return staticType{classList, "list"}
	case UnaryExpr:
		if e.Op == UnaryNot {
			return staticType{classNumeric, "bool"}
		}
		return staticType{classNumeric, "number"}
	case BinExpr:
		switch e.Op {
		case BinAdd, BinSub, BinMul, BinDiv:
			return staticType{classNumeric, "number"}
		case BinIntersect, BinUnion:
			return staticType{classList, "list"}
		}
		return staticType{classNumeric, "bool"}
	case FuncExpr:
		if c, ok := functionClass[e.Name]; ok {
			return staticType{c, e.Name + "()"}
		}
		return staticType{classUnknown, e.Name + "()"}
	}
	return staticType{classUnknown, "expression"}
}

func isDateString(s string) bool {
	if _, err := time.Parse("2006-01-02", s); err == nil {
		return true
	}
	_, err := time.Parse("2006-01-02T15:04:05", s)
	return err == nil
}

var orderingSymbol = map[BinOp]string{BinLt: "<", BinLe: "<=", BinGt: ">", BinGe: ">="}

// checkOrdering reports an error if an ordering comparison can never be true
// for any input. List membership (list >= x, x <= list) is allowed.
func checkOrdering(e BinExpr) error {
	l, r := typeOf(e.Left), typeOf(e.Right)
	if (e.Op == BinGe && l.class == classList && r.class != classList) ||
		(e.Op == BinLe && r.class == classList && l.class != classList) {
		return nil
	}
	if l.class == classUnknown || r.class == classUnknown {
		return nil
	}
	if l.class == r.class && (l.class == classNumeric || l.class == classTemporal) {
		return nil
	}
	return fmt.Errorf("type error: cannot compare %s %s %s; ordering needs two numbers or two dates/datetimes",
		l.desc, orderingSymbol[e.Op], r.desc)
}

// checkExpr walks e and checks every ordering comparison in it.
func checkExpr(e Expr) error {
	switch e := e.(type) {
	case BinExpr:
		if _, ok := orderingSymbol[e.Op]; ok {
			if err := checkOrdering(e); err != nil {
				return err
			}
		}
		if err := checkExpr(e.Left); err != nil {
			return err
		}
		return checkExpr(e.Right)
	case UnaryExpr:
		return checkExpr(e.Operand)
	case ListExpr:
		for _, x := range e.Elems {
			if err := checkExpr(x); err != nil {
				return err
			}
		}
	case FuncExpr:
		for _, x := range e.Args {
			if err := checkExpr(x); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkQuery statically type-checks every expression in a parsed query.
func checkQuery(q Query) error {
	base := q.q()
	exprs := append([]Expr{}, base.Select...)
	if base.Where != nil {
		exprs = append(exprs, base.Where)
	}
	for _, s := range base.SortBy {
		exprs = append(exprs, s.Expr)
	}
	if u, ok := q.(UpdateQuery); ok {
		for _, a := range u.Set {
			if a.Value != nil {
				exprs = append(exprs, a.Value)
			}
		}
	}
	for _, e := range exprs {
		if err := checkExpr(e); err != nil {
			return err
		}
	}
	return nil
}

// typeKeyword returns the query-syntax name of t (e.g. "numeric", not "number").
func typeKeyword(t FieldType) string {
	for name, ft := range typeNames {
		if ft == t {
			return name
		}
	}
	return t.String()
}
