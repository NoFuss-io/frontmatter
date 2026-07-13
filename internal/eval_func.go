package internal

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

func evalFunc(e FuncExpr, fm FrontMatter) (Value, error) {
	switch e.Name {
	// ── String ────────────────────────────────────────────────────────────────
	case "lower":
		return fnUnaryStr(e.Args, fm, func(s string) Value {
			return fnStr(strings.ToLower(s))
		})
	case "upper":
		return fnUnaryStr(e.Args, fm, func(s string) Value {
			return fnStr(strings.ToUpper(s))
		})
	case "length":
		return fnUnaryStr(e.Args, fm, func(s string) Value {
			return fnInt(int64(utf8.RuneCountInString(s)))
		})
	case "substr":
		return fnSubstr(e.Args, fm)
	case "starts_with":
		return fnBinaryStr(e.Args, fm, func(s, t string) Value {
			return fnBool(strings.HasPrefix(s, t))
		})
	case "ends_with":
		return fnBinaryStr(e.Args, fm, func(s, t string) Value {
			return fnBool(strings.HasSuffix(s, t))
		})
	case "contains_substr":
		return fnBinaryStr(e.Args, fm, func(s, t string) Value {
			return fnBool(strings.Contains(s, t))
		})
	case "trim":
		return fnTrim(e.Args, fm, strings.TrimFunc, strings.Trim)
	case "ltrim":
		return fnTrim(e.Args, fm, strings.TrimLeftFunc, strings.TrimLeft)
	case "rtrim":
		return fnTrim(e.Args, fm, strings.TrimRightFunc, strings.TrimRight)
	case "replace":
		return fnReplace(e.Args, fm)
	case "split":
		return fnBinaryStr(e.Args, fm, func(s, sep string) Value {
			parts := strings.Split(s, sep)
			vals := make([]Value, len(parts))
			for i, p := range parts {
				vals[i] = fnStr(p)
			}
			return Value{Kind: TypeList, Data: vals}
		})
	case "concat":
		return fnConcat(e.Args, fm)
	case "regexp_contains":
		return fnBinaryStr(e.Args, fm, func(s, pat string) Value {
			re, err := regexp.Compile(pat)
			if err != nil {
				return Value{Null: true}
			}
			return fnBool(re.MatchString(s))
		})
	case "regexp_extract":
		return fnBinaryStr(e.Args, fm, func(s, pat string) Value {
			re, err := regexp.Compile(pat)
			if err != nil {
				return Value{Null: true}
			}
			m := re.FindStringSubmatch(s)
			if m == nil {
				return Value{Null: true}
			}
			if len(m) > 1 {
				return fnStr(m[1])
			}
			return fnStr(m[0])
		})
	case "to_string":
		return fnToString(e.Args, fm)

	// ── Numeric ───────────────────────────────────────────────────────────────
	case "abs":
		return fnUnaryNum(e.Args, fm, func(v Value) Value {
			if v.Kind == TypeInt {
				n := v.Data.(int64)
				if n < 0 {
					n = -n
				}
				return fnInt(n)
			}
			return Value{Kind: TypeNumber, Data: math.Abs(v.Data.(float64))}
		})
	case "ceil":
		return fnUnaryNum(e.Args, fm, func(v Value) Value {
			if v.Kind == TypeInt {
				return v
			}
			return fnInt(int64(math.Ceil(v.Data.(float64))))
		})
	case "floor":
		return fnUnaryNum(e.Args, fm, func(v Value) Value {
			if v.Kind == TypeInt {
				return v
			}
			return fnInt(int64(math.Floor(v.Data.(float64))))
		})
	case "round":
		return fnRound(e.Args, fm)
	case "mod":
		return fnMod(e.Args, fm)
	case "sqrt":
		return fnUnaryNum(e.Args, fm, func(v Value) Value {
			f, err := Cast(v, TypeNumber)
			if err != nil {
				return Value{Null: true}
			}
			return Value{Kind: TypeNumber, Data: math.Sqrt(f.Data.(float64))}
		})
	case "pow":
		return fnPow(e.Args, fm)
	case "greatest":
		return fnMinMax(e.Args, fm, false)
	case "least":
		return fnMinMax(e.Args, fm, true)
	case "coalesce":
		return fnCoalesce(e.Args, fm)

	// ── List ──────────────────────────────────────────────────────────────────
	case "array_length":
		return fnArrayLength(e.Args, fm)
	case "array_contains":
		return fnArrayContains(e.Args, fm)
	case "array_concat":
		return fnArrayConcat(e.Args, fm)
	case "distinct":
		return fnDistinct(e.Args, fm)
	case "array_to_string":
		return fnArrayToString(e.Args, fm)

	// ── Date ──────────────────────────────────────────────────────────────────
	case "today":
		return Value{Kind: TypeDate, Data: time.Now().Truncate(24 * time.Hour)}, nil
	case "year":
		return fnDatePart(e.Args, fm, func(t time.Time) int64 { return int64(t.Year()) })
	case "month":
		return fnDatePart(e.Args, fm, func(t time.Time) int64 { return int64(t.Month()) })
	case "day":
		return fnDatePart(e.Args, fm, func(t time.Time) int64 { return int64(t.Day()) })
	case "date_diff":
		return fnDateDiff(e.Args, fm)
	}
	return Value{Null: true}, fmt.Errorf("unknown function %q", e.Name)
}

// builtinFuncs is the set of recognized built-in function names (lowercase).
// Kept in sync with the switch in evalFunc.
var builtinFuncs = map[string]bool{
	"lower": true, "upper": true, "length": true, "substr": true,
	"starts_with": true, "ends_with": true, "contains_substr": true,
	"trim": true, "ltrim": true, "rtrim": true, "replace": true,
	"split": true, "concat": true, "regexp_contains": true,
	"regexp_extract": true, "to_string": true,
	"abs": true, "ceil": true, "floor": true, "round": true, "mod": true,
	"sqrt": true, "pow": true, "greatest": true, "least": true, "coalesce": true,
	"array_length": true, "array_contains": true, "array_concat": true,
	"distinct": true, "array_to_string": true,
	"today": true, "year": true, "month": true, "day": true, "date_diff": true,
}

// isKnownFunc reports whether name (case-insensitive) is a built-in function.
func isKnownFunc(name string) bool {
	return builtinFuncs[strings.ToLower(name)]
}

// ── helpers ───────────────────────────────────────────────────────────────────

func fnStr(s string) Value { return Value{Kind: TypeString, Data: s} }
func fnBool(b bool) Value  { return Value{Kind: TypeBool, Data: b} }
func fnInt(n int64) Value  { return Value{Kind: TypeInt, Data: n} }

// argStr evaluates args[i] and casts it to string. The bool reports whether a
// usable value was produced (present, non-null, castable); an eval error is
// returned separately so it can propagate and fail the file.
func argStr(args []Expr, i int, fm FrontMatter) (string, bool, error) {
	if i >= len(args) {
		return "", false, nil
	}
	v, err := args[i].Eval(fm)
	if err != nil {
		return "", false, fmt.Errorf("could not evaluate argument %d: %w", i+1, err)
	}
	s, cerr := Cast(v, TypeString)
	if cerr != nil || s.Null {
		return "", false, nil
	}
	return s.Data.(string), true, nil
}

func argInt(args []Expr, i int, fm FrontMatter) (int64, bool, error) {
	if i >= len(args) {
		return 0, false, nil
	}
	v, err := args[i].Eval(fm)
	if err != nil {
		return 0, false, fmt.Errorf("could not evaluate argument %d: %w", i+1, err)
	}
	n, cerr := Cast(v, TypeInt)
	if cerr != nil || n.Null {
		return 0, false, nil
	}
	return n.Data.(int64), true, nil
}

func argNum(args []Expr, i int, fm FrontMatter) (float64, bool, error) {
	if i >= len(args) {
		return 0, false, nil
	}
	v, err := args[i].Eval(fm)
	if err != nil {
		return 0, false, fmt.Errorf("could not evaluate argument %d: %w", i+1, err)
	}
	n, cerr := Cast(v, TypeNumber)
	if cerr != nil || n.Null {
		return 0, false, nil
	}
	return n.Data.(float64), true, nil
}

func fnUnaryStr(args []Expr, fm FrontMatter, f func(string) Value) (Value, error) {
	if len(args) != 1 {
		return Value{Null: true}, nil
	}
	s, ok, err := argStr(args, 0, fm)
	if err != nil {
		return Value{}, err
	}
	if !ok {
		return Value{Null: true}, nil
	}
	return f(s), nil
}

func fnBinaryStr(args []Expr, fm FrontMatter, f func(string, string) Value) (Value, error) {
	if len(args) != 2 {
		return Value{Null: true}, nil
	}
	s, ok1, err := argStr(args, 0, fm)
	if err != nil {
		return Value{}, err
	}
	t, ok2, err := argStr(args, 1, fm)
	if err != nil {
		return Value{}, err
	}
	if !ok1 || !ok2 {
		return Value{Null: true}, nil
	}
	return f(s, t), nil
}

func fnUnaryNum(args []Expr, fm FrontMatter, f func(Value) Value) (Value, error) {
	if len(args) != 1 {
		return Value{Null: true}, nil
	}
	v, err := args[0].Eval(fm)
	if err != nil {
		return Value{}, fmt.Errorf("could not evaluate argument 1: %w", err)
	}
	if v.Null {
		return Value{Null: true}, nil
	}
	if v.Kind != TypeInt && v.Kind != TypeNumber {
		c, cerr := Cast(v, TypeNumber)
		if cerr != nil {
			return Value{Null: true}, nil
		}
		v = c
	}
	return f(v), nil
}

func fnTrim(
	args []Expr, fm FrontMatter,
	trimFunc func(string, func(rune) bool) string,
	trimStr func(string, string) string,
) (Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return Value{Null: true}, nil
	}
	s, ok, err := argStr(args, 0, fm)
	if err != nil {
		return Value{}, err
	}
	if !ok {
		return Value{Null: true}, nil
	}
	if len(args) == 1 {
		return fnStr(trimFunc(s, func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\n' || r == '\r'
		})), nil
	}
	chars, ok, err := argStr(args, 1, fm)
	if err != nil {
		return Value{}, err
	}
	if !ok {
		return Value{Null: true}, nil
	}
	return fnStr(trimStr(s, chars)), nil
}

func fnSubstr(args []Expr, fm FrontMatter) (Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return Value{Null: true}, nil
	}
	s, ok, err := argStr(args, 0, fm)
	if err != nil {
		return Value{}, err
	}
	if !ok {
		return Value{Null: true}, nil
	}
	pos, ok, err := argInt(args, 1, fm)
	if err != nil {
		return Value{}, err
	}
	if !ok {
		return Value{Null: true}, nil
	}
	runes := []rune(s)
	n := int64(len(runes))
	// 1-based; negative counts from end
	if pos < 0 {
		pos = n + pos + 1
	}
	if pos < 1 {
		pos = 1
	}
	start := pos - 1
	if start >= n {
		return fnStr(""), nil
	}
	if len(args) == 2 {
		return fnStr(string(runes[start:])), nil
	}
	length, ok, err := argInt(args, 2, fm)
	if err != nil {
		return Value{}, err
	}
	if !ok {
		return Value{Null: true}, nil
	}
	end := start + length
	if end > n {
		end = n
	}
	if end < start {
		return fnStr(""), nil
	}
	return fnStr(string(runes[start:end])), nil
}

func fnReplace(args []Expr, fm FrontMatter) (Value, error) {
	if len(args) != 3 {
		return Value{Null: true}, nil
	}
	s, ok1, err := argStr(args, 0, fm)
	if err != nil {
		return Value{}, err
	}
	from, ok2, err := argStr(args, 1, fm)
	if err != nil {
		return Value{}, err
	}
	to, ok3, err := argStr(args, 2, fm)
	if err != nil {
		return Value{}, err
	}
	if !ok1 || !ok2 || !ok3 {
		return Value{Null: true}, nil
	}
	return fnStr(strings.ReplaceAll(s, from, to)), nil
}

func fnConcat(args []Expr, fm FrontMatter) (Value, error) {
	if len(args) < 2 {
		return Value{Null: true}, nil
	}
	var sb strings.Builder
	for i := range args {
		s, ok, err := argStr(args, i, fm)
		if err != nil {
			return Value{}, err
		}
		if !ok {
			return Value{Null: true}, nil
		}
		sb.WriteString(s)
	}
	return fnStr(sb.String()), nil
}

func fnToString(args []Expr, fm FrontMatter) (Value, error) {
	if len(args) != 1 {
		return Value{Null: true}, nil
	}
	v, err := args[0].Eval(fm)
	if err != nil {
		return Value{}, fmt.Errorf("could not evaluate argument 1: %w", err)
	}
	if v.Null {
		return Value{Null: true}, nil
	}
	s, cerr := Cast(v, TypeString)
	if cerr != nil {
		return Value{Null: true}, nil
	}
	return s, nil
}

func fnRound(args []Expr, fm FrontMatter) (Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return Value{Null: true}, nil
	}
	f, ok, err := argNum(args, 0, fm)
	if err != nil {
		return Value{}, err
	}
	if !ok {
		return Value{Null: true}, nil
	}
	digits := int64(0)
	if len(args) == 2 {
		d, ok, err := argInt(args, 1, fm)
		if err != nil {
			return Value{}, err
		}
		if !ok {
			return Value{Null: true}, nil
		}
		digits = d
	}
	factor := math.Pow(10, float64(digits))
	return Value{Kind: TypeNumber, Data: math.Round(f*factor) / factor}, nil
}

func fnMod(args []Expr, fm FrontMatter) (Value, error) {
	if len(args) != 2 {
		return Value{Null: true}, nil
	}
	x, ok1, err := argInt(args, 0, fm)
	if err != nil {
		return Value{}, err
	}
	y, ok2, err := argInt(args, 1, fm)
	if err != nil {
		return Value{}, err
	}
	if !ok1 || !ok2 || y == 0 {
		return Value{Null: true}, nil
	}
	return fnInt(x % y), nil
}

func fnPow(args []Expr, fm FrontMatter) (Value, error) {
	if len(args) != 2 {
		return Value{Null: true}, nil
	}
	x, ok1, err := argNum(args, 0, fm)
	if err != nil {
		return Value{}, err
	}
	y, ok2, err := argNum(args, 1, fm)
	if err != nil {
		return Value{}, err
	}
	if !ok1 || !ok2 {
		return Value{Null: true}, nil
	}
	return Value{Kind: TypeNumber, Data: math.Pow(x, y)}, nil
}

func fnMinMax(args []Expr, fm FrontMatter, wantMin bool) (Value, error) {
	if len(args) == 0 {
		return Value{Null: true}, nil
	}
	best := Value{Null: true}
	for i, a := range args {
		v, err := a.Eval(fm)
		if err != nil {
			return Value{}, fmt.Errorf("could not evaluate argument %d: %w", i+1, err)
		}
		if v.Null {
			continue
		}
		if best.Null {
			best = v
			continue
		}
		cmp := compare(BinLt, v, best)
		less := !cmp.Null && cmp.Data.(bool)
		if wantMin && less {
			best = v
		} else if !wantMin && !less {
			// v >= best; keep v only if strictly greater
			gt := compare(BinGt, v, best)
			if !gt.Null && gt.Data.(bool) {
				best = v
			}
		}
	}
	return best, nil
}

func fnCoalesce(args []Expr, fm FrontMatter) (Value, error) {
	for i, a := range args {
		v, err := a.Eval(fm)
		if err != nil {
			return Value{}, fmt.Errorf("could not evaluate argument %d: %w", i+1, err)
		}
		if !v.Null {
			return v, nil
		}
	}
	return Value{Null: true}, nil
}

func fnArrayLength(args []Expr, fm FrontMatter) (Value, error) {
	if len(args) != 1 {
		return Value{Null: true}, nil
	}
	v, err := args[0].Eval(fm)
	if err != nil {
		return Value{}, fmt.Errorf("could not evaluate argument 1: %w", err)
	}
	if v.Null {
		return Value{Null: true}, nil
	}
	if v.Kind != TypeList {
		return fnInt(1), nil
	}
	return fnInt(int64(len(v.Data.([]Value)))), nil
}

func fnArrayContains(args []Expr, fm FrontMatter) (Value, error) {
	if len(args) != 2 {
		return Value{Null: true}, nil
	}
	list, err := args[0].Eval(fm)
	if err != nil {
		return Value{}, fmt.Errorf("could not evaluate argument 1: %w", err)
	}
	elem, err := args[1].Eval(fm)
	if err != nil {
		return Value{}, fmt.Errorf("could not evaluate argument 2: %w", err)
	}
	if list.Null || elem.Null {
		return Value{Null: true}, nil
	}
	if list.Kind != TypeList {
		return fnBool(scalarEq(list, elem)), nil
	}
	return fnBool(listContains(list.Data.([]Value), elem)), nil
}

func fnArrayConcat(args []Expr, fm FrontMatter) (Value, error) {
	if len(args) == 0 {
		return Value{Null: true}, nil
	}
	var out []Value
	for i, a := range args {
		v, err := a.Eval(fm)
		if err != nil {
			return Value{}, fmt.Errorf("could not evaluate argument %d: %w", i+1, err)
		}
		if v.Null {
			return Value{Null: true}, nil
		}
		if v.Kind == TypeList {
			out = append(out, v.Data.([]Value)...)
		} else {
			out = append(out, v)
		}
	}
	if out == nil {
		out = []Value{}
	}
	return Value{Kind: TypeList, Data: out}, nil
}

func fnDistinct(args []Expr, fm FrontMatter) (Value, error) {
	if len(args) != 1 {
		return Value{Null: true}, nil
	}
	v, err := args[0].Eval(fm)
	if err != nil {
		return Value{}, fmt.Errorf("could not evaluate argument 1: %w", err)
	}
	if v.Null {
		return Value{Null: true}, nil
	}
	if v.Kind != TypeList {
		return Value{Kind: TypeList, Data: []Value{v}}, nil
	}
	src := v.Data.([]Value)
	out := make([]Value, 0, len(src))
	for _, x := range src {
		if !listContains(out, x) {
			out = append(out, x)
		}
	}
	return Value{Kind: TypeList, Data: out}, nil
}

func fnArrayToString(args []Expr, fm FrontMatter) (Value, error) {
	if len(args) != 2 {
		return Value{Null: true}, nil
	}
	v, err := args[0].Eval(fm)
	if err != nil {
		return Value{}, fmt.Errorf("could not evaluate argument 1: %w", err)
	}
	sep, ok, err := argStr(args, 1, fm)
	if err != nil {
		return Value{}, err
	}
	if !ok || v.Null {
		return Value{Null: true}, nil
	}
	var elems []Value
	if v.Kind == TypeList {
		elems = v.Data.([]Value)
	} else {
		elems = []Value{v}
	}
	parts := make([]string, 0, len(elems))
	for _, e := range elems {
		s, cerr := Cast(e, TypeString)
		if cerr != nil || s.Null {
			return Value{Null: true}, nil
		}
		parts = append(parts, s.Data.(string))
	}
	return fnStr(strings.Join(parts, sep)), nil
}

func fnDatePart(args []Expr, fm FrontMatter, f func(time.Time) int64) (Value, error) {
	if len(args) != 1 {
		return Value{Null: true}, nil
	}
	v, err := args[0].Eval(fm)
	if err != nil {
		return Value{}, fmt.Errorf("could not evaluate argument 1: %w", err)
	}
	if v.Null {
		return Value{Null: true}, nil
	}
	if v.Kind != TypeDate && v.Kind != TypeDatetime {
		c, cerr := Cast(v, TypeDate)
		if cerr != nil {
			c2, cerr2 := Cast(v, TypeDatetime)
			if cerr2 != nil {
				return Value{Null: true}, nil
			}
			c = c2
		}
		v = c
	}
	return fnInt(f(v.Data.(time.Time))), nil
}

func fnDateDiff(args []Expr, fm FrontMatter) (Value, error) {
	if len(args) != 2 {
		return Value{Null: true}, nil
	}
	coerce := func(i int) (time.Time, bool, error) {
		v, err := args[i].Eval(fm)
		if err != nil {
			return time.Time{}, false, fmt.Errorf("could not evaluate argument %d: %w", i+1, err)
		}
		if v.Null {
			return time.Time{}, false, nil
		}
		if v.Kind != TypeDate && v.Kind != TypeDatetime {
			c, cerr := Cast(v, TypeDate)
			if cerr != nil {
				return time.Time{}, false, nil
			}
			v = c
		}
		return v.Data.(time.Time), true, nil
	}
	a, ok1, err := coerce(0)
	if err != nil {
		return Value{}, err
	}
	b, ok2, err := coerce(1)
	if err != nil {
		return Value{}, err
	}
	if !ok1 || !ok2 {
		return Value{Null: true}, nil
	}
	days := int64(a.Sub(b).Hours() / 24)
	return fnInt(days), nil
}
