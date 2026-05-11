package main

import (
	"fmt"
	"strings"
)

type FieldType int

const (
	TypeAny FieldType = iota
	TypeString
	TypeInt
	TypeNumber
	TypeDate
	TypeLink
	TypeList
)

var typeNames = map[string]FieldType{
	"any": TypeAny, "string": TypeString, "int": TypeInt,
	"number": TypeNumber, "date": TypeDate, "link": TypeLink, "list": TypeList,
}

type Field struct {
	Name     string
	Type     FieldType
	ElemType *FieldType // non-nil for list:<type>
}

// Comparison is used in `where` clauses.
type Comparison struct {
	Neg   bool
	Field Field
	Value *string // nil = type-only match
}

// AssignOp is the mutation operator used in assignments.
type AssignOp int

const (
	OpSet AssignOp = iota // =
	OpAdd                 // +=
	OpSub                 // -=
)

// Assignment is used in `update set` clauses.
type Assignment struct {
	Field Field
	Op    AssignOp
	Value *string // nil = cast to Field.Type
}

// Expression is an OR of AND-groups.
type Expression [][]Comparison

func parseTypeName(s string) (FieldType, error) {
	if strings.HasPrefix(s, "list:") {
		return TypeList, nil
	}
	t, ok := typeNames[s]
	if !ok {
		return TypeAny, fmt.Errorf("unknown type %q", s)
	}
	return t, nil
}

func ParseField(s string) (Field, error) {
	i := strings.IndexByte(s, ':')
	if i < 0 {
		return Field{Name: s}, nil
	}
	name, typePart := s[:i], s[i+1:]
	f := Field{Name: name}
	if strings.HasPrefix(typePart, "list:") {
		f.Type = TypeList
		elemName := typePart[5:]
		et, err := parseTypeName(elemName)
		if err != nil {
			return f, err
		}
		f.ElemType = &et
		return f, nil
	}
	t, err := parseTypeName(typePart)
	if err != nil {
		return f, err
	}
	f.Type = t
	return f, nil
}

func ParseComparison(s string) (Comparison, error) {
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	i := strings.IndexByte(s, '=')
	if i < 0 {
		f, err := ParseField(s)
		return Comparison{Neg: neg, Field: f}, err
	}
	f, err := ParseField(s[:i])
	if err != nil {
		return Comparison{}, err
	}
	v := s[i+1:]
	return Comparison{Neg: neg, Field: f, Value: &v}, nil
}

func ParseAssignment(s string) (Assignment, error) {
	if i := strings.Index(s, "+="); i >= 0 {
		f, err := ParseField(s[:i])
		if err != nil {
			return Assignment{}, err
		}
		v := s[i+2:]
		return Assignment{Field: f, Op: OpAdd, Value: &v}, nil
	}
	if i := strings.Index(s, "-="); i >= 0 {
		f, err := ParseField(s[:i])
		if err != nil {
			return Assignment{}, err
		}
		v := s[i+2:]
		return Assignment{Field: f, Op: OpSub, Value: &v}, nil
	}
	if i := strings.IndexByte(s, '='); i >= 0 {
		f, err := ParseField(s[:i])
		if err != nil {
			return Assignment{}, err
		}
		v := s[i+1:]
		return Assignment{Field: f, Op: OpSet, Value: &v}, nil
	}
	f, err := ParseField(s)
	if err != nil {
		return Assignment{}, err
	}
	return Assignment{Field: f, Op: OpSet}, nil // Value nil = cast
}

func ParseExpression(s string) (Expression, error) {
	s = strings.TrimSpace(s)
	if len(s) > 1 && s[0] == '(' && s[len(s)-1] == ')' {
		s = s[1 : len(s)-1]
	}
	var expr Expression
	for _, orPart := range strings.Split(s, " or ") {
		var group []Comparison
		for _, andPart := range strings.Split(orPart, " and ") {
			cmp, err := ParseComparison(strings.TrimSpace(andPart))
			if err != nil {
				return nil, fmt.Errorf("in expression: %w", err)
			}
			group = append(group, cmp)
		}
		expr = append(expr, group)
	}
	return expr, nil
}
