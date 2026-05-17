package lib

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

// Value is the runtime value produced by evaluating an expression.
// Void marks absence: a missing field or a cast that could not produce a value.
// Per Manual.md: void propagates through arithmetic; in boolean context it is falsey.
type Value struct {
	Kind FieldType
	Data any
	Void bool
}

// Row is the projected output of SelectQuery.Eval for a single document.
// The slice index matches the order of SelectQuery.Fields.
type Row []Value

var errEvalNotImplemented = errors.New("eval not implemented")

// Cast converts v to target. Returns a Void Value if the conversion is not
// possible (e.g. string "hello" → int).
func Cast(v Value, target FieldType) Value {
	if v.Void {
		return Value{Void: true}
	}
	if target == TypeAny || target == v.Kind {
		return v
	}
	if target == TypeList {
		return Value{Kind: TypeList, Data: []Value{v}}
	}
	if v.Kind == TypeList {
		els, ok := v.Data.([]Value)
		if !ok || len(els) != 1 {
			return Value{Void: true}
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
	}
	return Value{Void: true}
}

func castToBool(v Value) Value {
	switch v.Kind {
	case TypeInt:
		switch v.Data.(int64) {
		case 0:
			return Value{Kind: TypeBool, Data: false}
		case 1:
			return Value{Kind: TypeBool, Data: true}
		}
	case TypeNumber:
		switch v.Data.(float64) {
		case 0:
			return Value{Kind: TypeBool, Data: false}
		case 1:
			return Value{Kind: TypeBool, Data: true}
		}
	case TypeString:
		switch strings.ToLower(v.Data.(string)) {
		case "true":
			return Value{Kind: TypeBool, Data: true}
		case "false":
			return Value{Kind: TypeBool, Data: false}
		}
	}
	return Value{Void: true}
}

func castToInt(v Value) Value {
	switch v.Kind {
	case TypeBool:
		if v.Data.(bool) {
			return Value{Kind: TypeInt, Data: int64(1)}
		}
		return Value{Kind: TypeInt, Data: int64(0)}
	case TypeNumber:
		f := v.Data.(float64)
		if f != math.Trunc(f) {
			return Value{Void: true}
		}
		return Value{Kind: TypeInt, Data: int64(f)}
	case TypeString:
		n, err := strconv.ParseInt(v.Data.(string), 0, 64)
		if err != nil {
			return Value{Void: true}
		}
		return Value{Kind: TypeInt, Data: n}
	}
	return Value{Void: true}
}

func castToNumber(v Value) Value {
	switch v.Kind {
	case TypeInt:
		return Value{Kind: TypeNumber, Data: float64(v.Data.(int64))}
	case TypeBool:
		if v.Data.(bool) {
			return Value{Kind: TypeNumber, Data: float64(1)}
		}
		return Value{Kind: TypeNumber, Data: float64(0)}
	case TypeString:
		f, err := strconv.ParseFloat(v.Data.(string), 64)
		if err != nil {
			return Value{Void: true}
		}
		return Value{Kind: TypeNumber, Data: f}
	}
	return Value{Void: true}
}

func castToString(v Value) Value {
	switch v.Kind {
	case TypeInt:
		return Value{Kind: TypeString, Data: strconv.FormatInt(v.Data.(int64), 10)}
	case TypeNumber:
		return Value{Kind: TypeString, Data: strconv.FormatFloat(v.Data.(float64), 'g', -1, 64)}
	case TypeBool:
		if v.Data.(bool) {
			return Value{Kind: TypeString, Data: "true"}
		}
		return Value{Kind: TypeString, Data: "false"}
	}
	return Value{Void: true}
}

// ── Expressions ───────────────────────────────────────────────────────────────

func (e LitExpr) Eval(_ *FrontMatter) Value {
	switch e.Kind {
	case LitInt:
		n, err := strconv.ParseInt(e.Value, 0, 64)
		if err != nil {
			return Value{Void: true}
		}
		return Value{Kind: TypeInt, Data: n}
	case LitNumeric:
		f, err := strconv.ParseFloat(e.Value, 64)
		if err != nil {
			return Value{Void: true}
		}
		return Value{Kind: TypeNumber, Data: f}
	case LitString:
		return Value{Kind: TypeString, Data: e.Value}
	case LitBool:
		return Value{Kind: TypeBool, Data: strings.ToLower(e.Value) == "true"}
	case LitNull:
		return Value{Void: true}
	}
	return Value{Void: true}
}
func (e FieldExpr) Eval(fm *FrontMatter) Value {
	if fm == nil {
		return Value{Void: true}
	}
	raw, ok := (*fm)[e.Field.Name]
	if !ok {
		return Value{Void: true}
	}
	v := valueFromAny(raw)
	if e.Field.Type == TypeAny {
		return v
	}
	return Cast(v, e.Field.Type)
}

// valueFromAny converts a raw Go value (from YAML-decoded frontmatter) into a
// typed Value. Unknown Go types fall through as TypeAny with Data preserved.
func valueFromAny(x any) Value {
	if x == nil {
		return Value{Void: true}
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
	}
	return Value{Kind: TypeAny, Data: x}
}
func (e UnaryExpr) Eval(fm *FrontMatter) Value {
	v := e.Operand.Eval(fm)
	switch e.Op {
	case UnaryNot:
		return Value{Kind: TypeBool, Data: !truthy(v)}
	case UnaryNeg:
		if v.Void {
			return Value{Void: true}
		}
		switch v.Kind {
		case TypeInt:
			return Value{Kind: TypeInt, Data: -v.Data.(int64)}
		case TypeNumber:
			return Value{Kind: TypeNumber, Data: -v.Data.(float64)}
		}
	}
	return Value{Void: true}
}

// truthy returns the boolean view of v. Void is falsey; non-bool values are
// passed through castToBool, with cast failure also treated as falsey.
func truthy(v Value) bool {
	if v.Void {
		return false
	}
	if v.Kind == TypeBool {
		return v.Data.(bool)
	}
	b := castToBool(v)
	if b.Void {
		return false
	}
	return b.Data.(bool)
}
func (BinExpr) Eval(*FrontMatter) Value   { return Value{Void: true} }

// SortTerm.Eval is a thin wrapper around the underlying expression.
func (s SortTerm) Eval(fm *FrontMatter) Value { return s.Expr.Eval(fm) }

// ── Assignment ────────────────────────────────────────────────────────────────

// Apply evaluates a.Value (if present), casts to a.Field.Type, and writes the
// result into doc.FrontMatter under a.Field.Name. A runtime cast failure is
// returned as an error so the caller can halt the current file per Manual.md.
func (Assign) Apply(*FrontMatter) error { return errEvalNotImplemented }

// ── Queries ───────────────────────────────────────────────────────────────────

// Eval evaluates the where clause; if truthy, projects Fields into a Row.
// Returns nil Row if where is falsey or evaluates to void.
func (SelectQuery) Eval(*FrontMatter) (Row, error) { return nil, errEvalNotImplemented }

// Eval evaluates the where clause; if truthy, applies each Assign to doc.
func (UpdateQuery) Eval(*FrontMatter) error { return errEvalNotImplemented }

// Eval evaluates the where clause; if truthy, drops or renames fields on doc.
func (AlterQuery) Eval(*FrontMatter) error { return errEvalNotImplemented }
