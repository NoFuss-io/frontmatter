package main

import (
	"fmt"
	"strconv"
	"strings"
)

// splitOn splits args on the first occurrence of keyword, returning the two halves.
func splitOn(args []string, keyword string) (before, after []string, found bool) {
	for i, a := range args {
		if a == keyword {
			return args[:i], args[i+1:], true
		}
	}
	return args, nil, false
}

// splitCommas expands comma-separated tokens, handling "a,b", "a, b", and "a , b".
// Empty parts are discarded, so trailing commas are silently accepted.
func splitCommas(args []string) []string {
	var out []string
	for _, arg := range args {
		for _, part := range strings.Split(arg, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

func parseSortClause(present bool, tokens []string) ([]Field, bool, error) {
	if !present {
		return nil, false, nil
	}
	if len(tokens) == 0 || tokens[0] != "by" {
		return nil, false, fmt.Errorf("expected 'by' after 'sort'")
	}
	args := splitCommas(tokens[1:])
	desc := len(args) > 0 && args[len(args)-1] == "desc"
	if desc {
		args = args[:len(args)-1]
	}
	if len(args) == 0 {
		return nil, false, fmt.Errorf("no fields specified after 'sort by'")
	}
	var fields []Field
	for _, arg := range args {
		f, err := ParseField(arg)
		if err != nil {
			return nil, false, err
		}
		fields = append(fields, f)
	}
	return fields, desc, nil
}

func parseLimitClause(present bool, tokens []string) (int, error) {
	if !present {
		return 0, nil
	}
	if len(tokens) != 1 {
		return 0, fmt.Errorf("limit requires exactly one integer argument")
	}
	n, err := strconv.Atoi(tokens[0])
	if err != nil || n < 0 {
		return 0, fmt.Errorf("limit must be a non-negative integer, got %q", tokens[0])
	}
	return n, nil
}

type FieldType int

const (
	TypeAny FieldType = iota
	TypeString
	TypeBool
	TypeInt
	TypeNumber
	TypeDate
	TypeLink
	TypeList
)

var typeNames = map[string]FieldType{
	"any": TypeAny, "string": TypeString, "bool": TypeBool, "int": TypeInt,
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
