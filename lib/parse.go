package lib

import (
	"errors"
	"io"
)

var errNotImplemented = errors.New("not implemented")

// ParseQuery reads a full query from r and returns a SelectQuery, UpdateQuery, or AlterQuery.
func ParseQuery(r io.Reader) (Query, error) {
	return nil, errNotImplemented
}

func (q *SelectQuery) Parse(r io.Reader) error {
	return errNotImplemented
}

func (q *UpdateQuery) Parse(r io.Reader) error {
	return errNotImplemented
}

func (q *AlterQuery) Parse(r io.Reader) error {
	return errNotImplemented
}

// ParseExpr reads an expression from r and returns the appropriate Expr node.
func ParseExpr(r io.Reader) (Expr, error) {
	return nil, errNotImplemented
}

func (e *BinExpr) Parse(r io.Reader) error {
	return errNotImplemented
}

func (e *UnaryExpr) Parse(r io.Reader) error {
	return errNotImplemented
}

func (e *FieldExpr) Parse(r io.Reader) error {
	return errNotImplemented
}

func (e *LitExpr) Parse(r io.Reader) error {
	return errNotImplemented
}

func (f *Field) Parse(r io.Reader) error {
	return errNotImplemented
}

func (a *Assign) Parse(r io.Reader) error {
	return errNotImplemented
}

func (s *SortTerm) Parse(r io.Reader) error {
	return errNotImplemented
}

func (p *RenamePair) Parse(r io.Reader) error {
	return errNotImplemented
}
