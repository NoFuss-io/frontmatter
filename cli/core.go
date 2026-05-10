package main

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
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

// dateVal preserves YAML date-only formatting (unquoted, no time component).
type dateVal string

func (d dateVal) MarshalYAML() (interface{}, error) {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!timestamp", Value: string(d)}, nil
}

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
	neg := strings.HasPrefix(s, "!")
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
	for _, orPart := range strings.Split(s, "||") {
		var group []Comparison
		for _, andPart := range strings.Split(orPart, "&&") {
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

// File represents a parsed markdown file with frontmatter.
type File struct {
	Path  string
	FM    map[string]interface{}
	Body  string
	hasFM bool
}

func normalizeFM(m map[string]interface{}) {
	for k, v := range m {
		m[k] = normalizeVal(v)
	}
}

func normalizeVal(v interface{}) interface{} {
	switch v := v.(type) {
	case time.Time:
		return dateVal(v.Format("2006-01-02"))
	case map[string]interface{}:
		normalizeFM(v)
	case []interface{}:
		for i, item := range v {
			v[i] = normalizeVal(item)
		}
	}
	return v
}

func ReadFile(path string) (*File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f := &File{Path: path, FM: make(map[string]interface{})}
	s := string(raw)

	if !strings.HasPrefix(s, "---\n") {
		f.Body = s
		return f, nil
	}
	rest := s[4:]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		if after, found := strings.CutSuffix(rest, "\n---"); found {
			if err := yaml.Unmarshal([]byte(after), &f.FM); err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			normalizeFM(f.FM)
			f.hasFM = true
			return f, nil
		}
		f.Body = s
		return f, nil
	}
	f.hasFM = true
	if err := yaml.Unmarshal([]byte(rest[:end]), &f.FM); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	normalizeFM(f.FM)
	f.Body = rest[end+5:]
	return f, nil
}

func (f *File) Write() error {
	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(f.FM); err != nil {
		return err
	}
	buf.WriteString("---\n")
	buf.WriteString(f.Body)
	return os.WriteFile(f.Path, buf.Bytes(), 0644)
}

func (f *File) Matches(expr Expression) bool {
	if len(expr) == 0 {
		return true
	}
	for _, group := range expr {
		if f.matchesAll(group) {
			return true
		}
	}
	return false
}

func (f *File) matchesAll(cmps []Comparison) bool {
	for _, cmp := range cmps {
		if !f.matchesCmp(cmp) {
			return false
		}
	}
	return true
}

func (f *File) matchesCmp(cmp Comparison) bool {
	v, ok := f.FM[cmp.Field.Name]
	var result bool
	if !ok {
		result = false
	} else if cmp.Field.Type != TypeAny && !isType(v, cmp.Field) {
		result = false
	} else if cmp.Value != nil {
		if arr, ok := v.([]interface{}); ok {
			for _, item := range arr {
				if fmtValue(item) == *cmp.Value {
					result = true
					break
				}
			}
		} else {
			result = fmtValue(v) == *cmp.Value
		}
	} else {
		result = true
	}
	if cmp.Neg {
		return !result
	}
	return result
}

func isType(v interface{}, f Field) bool {
	switch f.Type {
	case TypeAny:
		return true
	case TypeString:
		_, ok := v.(string)
		return ok
	case TypeInt:
		switch v.(type) {
		case int, int64, uint64:
			return true
		}
		return false
	case TypeNumber:
		switch v.(type) {
		case int, int64, uint64, float64, float32:
			return true
		}
		return false
	case TypeDate:
		switch v.(type) {
		case time.Time, dateVal:
			return true
		case string:
			_, err := time.Parse("2006-01-02", v.(string))
			return err == nil
		}
		return false
	case TypeLink:
		if s, ok := v.(string); ok {
			return strings.HasPrefix(s, "[[") || (strings.HasPrefix(s, "[") && strings.Contains(s, "]("))
		}
		return false
	case TypeList:
		arr, ok := v.([]interface{})
		if !ok {
			return false
		}
		if f.ElemType == nil {
			return true
		}
		elemField := Field{Type: *f.ElemType}
		for _, item := range arr {
			if !isType(item, elemField) {
				return false
			}
		}
		return true
	}
	return false
}

func fmtValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch v := v.(type) {
	case time.Time:
		return v.Format("2006-01-02")
	case dateVal:
		return string(v)
	case []interface{}:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = fmtValue(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (f *File) set(name, value string) {
	f.FM[name] = value
	f.hasFM = true
}

func (f *File) Remove(field Field) bool {
	v, ok := f.FM[field.Name]
	if !ok {
		return false
	}
	if field.Type != TypeAny && !isType(v, field) {
		return false
	}
	delete(f.FM, field.Name)
	return true
}

func (f *File) Apply(a Assignment) error {
	v, ok := f.FM[a.Field.Name]

	if a.Value == nil {
		// Cast form: no = present, field type is the target.
		if !ok || a.Field.Type == TypeAny {
			return nil
		}
		newVal, err := castValue(v, a.Field)
		if err != nil {
			return fmt.Errorf("casting %q: %w", a.Field.Name, err)
		}
		f.FM[a.Field.Name] = newVal
		f.hasFM = true
		return nil
	}

	switch a.Op {
	case OpSet:
		f.set(a.Field.Name, *a.Value)

	case OpAdd:
		if !ok {
			f.FM[a.Field.Name] = *a.Value
			f.hasFM = true
			return nil
		}
		switch cur := v.(type) {
		case int:
			n, err := strconv.ParseInt(*a.Value, 10, 64)
			if err != nil {
				return fmt.Errorf("field %q: += requires int value, got %q", a.Field.Name, *a.Value)
			}
			f.FM[a.Field.Name] = cur + int(n)
		case int64:
			n, err := strconv.ParseInt(*a.Value, 10, 64)
			if err != nil {
				return fmt.Errorf("field %q: += requires int value, got %q", a.Field.Name, *a.Value)
			}
			f.FM[a.Field.Name] = cur + n
		case float64:
			n, err := strconv.ParseFloat(*a.Value, 64)
			if err != nil {
				return fmt.Errorf("field %q: += requires number value, got %q", a.Field.Name, *a.Value)
			}
			f.FM[a.Field.Name] = cur + n
		case string:
			f.FM[a.Field.Name] = cur + *a.Value
		case []interface{}:
			for _, item := range cur {
				if fmtValue(item) == *a.Value {
					return nil // already present, treat as set
				}
			}
			f.FM[a.Field.Name] = append(cur, *a.Value)
		default:
			return fmt.Errorf("field %q: += not supported for this type", a.Field.Name)
		}
		f.hasFM = true

	case OpSub:
		if !ok {
			return nil
		}
		switch cur := v.(type) {
		case int:
			n, err := strconv.ParseInt(*a.Value, 10, 64)
			if err != nil {
				return fmt.Errorf("field %q: -= requires int value, got %q", a.Field.Name, *a.Value)
			}
			f.FM[a.Field.Name] = cur - int(n)
		case int64:
			n, err := strconv.ParseInt(*a.Value, 10, 64)
			if err != nil {
				return fmt.Errorf("field %q: -= requires int value, got %q", a.Field.Name, *a.Value)
			}
			f.FM[a.Field.Name] = cur - n
		case float64:
			n, err := strconv.ParseFloat(*a.Value, 64)
			if err != nil {
				return fmt.Errorf("field %q: -= requires number value, got %q", a.Field.Name, *a.Value)
			}
			f.FM[a.Field.Name] = cur - n
		case []interface{}:
			result := make([]interface{}, 0, len(cur))
			for _, item := range cur {
				if fmtValue(item) != *a.Value {
					result = append(result, item)
				}
			}
			f.FM[a.Field.Name] = result
		default:
			return fmt.Errorf("field %q: -= not supported for this type", a.Field.Name)
		}
		f.hasFM = true
	}
	return nil
}

func castValue(v interface{}, f Field) (interface{}, error) {
	s := fmtValue(v)
	switch f.Type {
	case TypeString:
		return s, nil
	case TypeInt:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, err
		}
		return int(n), nil
	case TypeNumber:
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, err
		}
		if n == math.Trunc(n) {
			return int(n), nil
		}
		return n, nil
	case TypeDate:
		for _, layout := range []string{"2006-01-02", "01/02/2006", "02.01.2006", time.RFC3339} {
			if t, err := time.Parse(layout, s); err == nil {
				return dateVal(t.Format("2006-01-02")), nil
			}
		}
		return nil, fmt.Errorf("cannot parse %q as date", s)
	case TypeLink:
		if strings.HasPrefix(s, "[[") {
			return s, nil
		}
		return "[[" + s + "]]", nil
	case TypeList:
		var arr []interface{}
		if existing, ok := v.([]interface{}); ok {
			arr = make([]interface{}, len(existing))
			copy(arr, existing)
		} else {
			arr = []interface{}{s}
		}
		if f.ElemType != nil {
			elemField := Field{Type: *f.ElemType}
			for i, item := range arr {
				casted, err := castValue(item, elemField)
				if err != nil {
					return nil, fmt.Errorf("element %d: %w", i, err)
				}
				arr[i] = casted
			}
		}
		return arr, nil
	case TypeAny:
		return v, nil
	}
	return nil, fmt.Errorf("unsupported target type")
}
