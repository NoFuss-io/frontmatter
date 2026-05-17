package lib

import "errors"

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
	_ = v
	_ = target
	return Value{Void: true}
}

// ── Expressions ───────────────────────────────────────────────────────────────

func (LitExpr) Eval(*FrontMatter) Value   { return Value{Void: true} }
func (FieldExpr) Eval(*FrontMatter) Value { return Value{Void: true} }
func (UnaryExpr) Eval(*FrontMatter) Value { return Value{Void: true} }
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
