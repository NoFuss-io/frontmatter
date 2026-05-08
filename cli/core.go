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

type Comparison struct {
	Field Field
	Value *string // nil = field must exist, no value constraint
}

// Expression is an OR of AND-groups
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
	i := strings.IndexByte(s, '=')
	if i < 0 {
		f, err := ParseField(s)
		return Comparison{Field: f}, err
	}
	f, err := ParseField(s[:i])
	if err != nil {
		return Comparison{}, err
	}
	v := s[i+1:]
	return Comparison{Field: f, Value: &v}, nil
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
		// Handle frontmatter ending at EOF without trailing newline
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
	if !ok {
		return false
	}
	if cmp.Field.Type != TypeAny && !isType(v, cmp.Field.Type) {
		return false
	}
	if cmp.Value != nil {
		return fmtValue(v) == *cmp.Value
	}
	return true
}

func isType(v interface{}, t FieldType) bool {
	switch t {
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
		_, ok := v.([]interface{})
		return ok
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

func (f *File) Set(name, value string) {
	f.FM[name] = value
	f.hasFM = true
}

func (f *File) Remove(field Field) bool {
	v, ok := f.FM[field.Name]
	if !ok {
		return false
	}
	if field.Type != TypeAny && !isType(v, field.Type) {
		return false
	}
	delete(f.FM, field.Name)
	return true
}

func (f *File) Cast(field Field, targetType FieldType) error {
	v, ok := f.FM[field.Name]
	if !ok {
		return fmt.Errorf("field %q not found", field.Name)
	}
	if field.Type != TypeAny && !isType(v, field.Type) {
		return fmt.Errorf("field %q is not of type %v", field.Name, field.Type)
	}
	newVal, err := castValue(v, targetType)
	if err != nil {
		return fmt.Errorf("casting %q: %w", field.Name, err)
	}
	f.FM[field.Name] = newVal
	return nil
}

func castValue(v interface{}, t FieldType) (interface{}, error) {
	s := fmtValue(v)
	switch t {
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
		if arr, ok := v.([]interface{}); ok {
			return arr, nil
		}
		return []interface{}{s}, nil
	case TypeAny:
		return v, nil
	}
	return nil, fmt.Errorf("unsupported target type")
}

func (f *File) CheckType(field Field) bool {
	v, ok := f.FM[field.Name]
	if !ok {
		return false
	}
	return isType(v, field.Type)
}
