package internal

import (
	"math"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

func evalFunc(e FuncExpr, fm FrontMatter) Value {
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
		return Value{Kind: TypeDate, Data: time.Now().Truncate(24 * time.Hour)}
	case "year":
		return fnDatePart(e.Args, fm, func(t time.Time) int64 { return int64(t.Year()) })
	case "month":
		return fnDatePart(e.Args, fm, func(t time.Time) int64 { return int64(t.Month()) })
	case "day":
		return fnDatePart(e.Args, fm, func(t time.Time) int64 { return int64(t.Day()) })
	case "date_diff":
		return fnDateDiff(e.Args, fm)
	}
	return Value{Null: true}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func fnStr(s string) Value { return Value{Kind: TypeString, Data: s} }
func fnBool(b bool) Value  { return Value{Kind: TypeBool, Data: b} }
func fnInt(n int64) Value  { return Value{Kind: TypeInt, Data: n} }

func argStr(args []Expr, i int, fm FrontMatter) (string, bool) {
	if i >= len(args) {
		return "", false
	}
	v := args[i].Eval(fm)
	s, err := Cast(v, TypeString)
	if err != nil || s.Null {
		return "", false
	}
	return s.Data.(string), true
}

func argInt(args []Expr, i int, fm FrontMatter) (int64, bool) {
	if i >= len(args) {
		return 0, false
	}
	v := args[i].Eval(fm)
	n, err := Cast(v, TypeInt)
	if err != nil || n.Null {
		return 0, false
	}
	return n.Data.(int64), true
}

func argNum(args []Expr, i int, fm FrontMatter) (float64, bool) {
	if i >= len(args) {
		return 0, false
	}
	v := args[i].Eval(fm)
	n, err := Cast(v, TypeNumber)
	if err != nil || n.Null {
		return 0, false
	}
	return n.Data.(float64), true
}

func fnUnaryStr(args []Expr, fm FrontMatter, f func(string) Value) Value {
	if len(args) != 1 {
		return Value{Null: true}
	}
	s, ok := argStr(args, 0, fm)
	if !ok {
		return Value{Null: true}
	}
	return f(s)
}

func fnBinaryStr(args []Expr, fm FrontMatter, f func(string, string) Value) Value {
	if len(args) != 2 {
		return Value{Null: true}
	}
	s, ok1 := argStr(args, 0, fm)
	t, ok2 := argStr(args, 1, fm)
	if !ok1 || !ok2 {
		return Value{Null: true}
	}
	return f(s, t)
}

func fnUnaryNum(args []Expr, fm FrontMatter, f func(Value) Value) Value {
	if len(args) != 1 {
		return Value{Null: true}
	}
	v := args[0].Eval(fm)
	if v.Null {
		return Value{Null: true}
	}
	if v.Kind != TypeInt && v.Kind != TypeNumber {
		c, err := Cast(v, TypeNumber)
		if err != nil {
			return Value{Null: true}
		}
		v = c
	}
	return f(v)
}

func fnTrim(
	args []Expr, fm FrontMatter,
	trimFunc func(string, func(rune) bool) string,
	trimStr func(string, string) string,
) Value {
	if len(args) < 1 || len(args) > 2 {
		return Value{Null: true}
	}
	s, ok := argStr(args, 0, fm)
	if !ok {
		return Value{Null: true}
	}
	if len(args) == 1 {
		return fnStr(trimFunc(s, func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\n' || r == '\r'
		}))
	}
	chars, ok := argStr(args, 1, fm)
	if !ok {
		return Value{Null: true}
	}
	return fnStr(trimStr(s, chars))
}

func fnSubstr(args []Expr, fm FrontMatter) Value {
	if len(args) < 2 || len(args) > 3 {
		return Value{Null: true}
	}
	s, ok := argStr(args, 0, fm)
	if !ok {
		return Value{Null: true}
	}
	pos, ok := argInt(args, 1, fm)
	if !ok {
		return Value{Null: true}
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
		return fnStr("")
	}
	if len(args) == 2 {
		return fnStr(string(runes[start:]))
	}
	length, ok := argInt(args, 2, fm)
	if !ok {
		return Value{Null: true}
	}
	end := start + length
	if end > n {
		end = n
	}
	if end < start {
		return fnStr("")
	}
	return fnStr(string(runes[start:end]))
}

func fnReplace(args []Expr, fm FrontMatter) Value {
	if len(args) != 3 {
		return Value{Null: true}
	}
	s, ok1 := argStr(args, 0, fm)
	from, ok2 := argStr(args, 1, fm)
	to, ok3 := argStr(args, 2, fm)
	if !ok1 || !ok2 || !ok3 {
		return Value{Null: true}
	}
	return fnStr(strings.ReplaceAll(s, from, to))
}

func fnConcat(args []Expr, fm FrontMatter) Value {
	if len(args) < 2 {
		return Value{Null: true}
	}
	var sb strings.Builder
	for i := range args {
		s, ok := argStr(args, i, fm)
		if !ok {
			return Value{Null: true}
		}
		sb.WriteString(s)
	}
	return fnStr(sb.String())
}

func fnToString(args []Expr, fm FrontMatter) Value {
	if len(args) != 1 {
		return Value{Null: true}
	}
	v := args[0].Eval(fm)
	if v.Null {
		return Value{Null: true}
	}
	s, err := Cast(v, TypeString)
	if err != nil {
		return Value{Null: true}
	}
	return s
}

func fnRound(args []Expr, fm FrontMatter) Value {
	if len(args) < 1 || len(args) > 2 {
		return Value{Null: true}
	}
	f, ok := argNum(args, 0, fm)
	if !ok {
		return Value{Null: true}
	}
	digits := int64(0)
	if len(args) == 2 {
		d, ok := argInt(args, 1, fm)
		if !ok {
			return Value{Null: true}
		}
		digits = d
	}
	factor := math.Pow(10, float64(digits))
	return Value{Kind: TypeNumber, Data: math.Round(f*factor) / factor}
}

func fnMod(args []Expr, fm FrontMatter) Value {
	if len(args) != 2 {
		return Value{Null: true}
	}
	x, ok1 := argInt(args, 0, fm)
	y, ok2 := argInt(args, 1, fm)
	if !ok1 || !ok2 || y == 0 {
		return Value{Null: true}
	}
	return fnInt(x % y)
}

func fnPow(args []Expr, fm FrontMatter) Value {
	if len(args) != 2 {
		return Value{Null: true}
	}
	x, ok1 := argNum(args, 0, fm)
	y, ok2 := argNum(args, 1, fm)
	if !ok1 || !ok2 {
		return Value{Null: true}
	}
	return Value{Kind: TypeNumber, Data: math.Pow(x, y)}
}

func fnMinMax(args []Expr, fm FrontMatter, wantMin bool) Value {
	if len(args) == 0 {
		return Value{Null: true}
	}
	best := Value{Null: true}
	for _, a := range args {
		v := a.Eval(fm)
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
	return best
}

func fnCoalesce(args []Expr, fm FrontMatter) Value {
	for _, a := range args {
		v := a.Eval(fm)
		if !v.Null {
			return v
		}
	}
	return Value{Null: true}
}

func fnArrayLength(args []Expr, fm FrontMatter) Value {
	if len(args) != 1 {
		return Value{Null: true}
	}
	v := args[0].Eval(fm)
	if v.Null {
		return Value{Null: true}
	}
	if v.Kind != TypeList {
		return fnInt(1)
	}
	return fnInt(int64(len(v.Data.([]Value))))
}

func fnArrayContains(args []Expr, fm FrontMatter) Value {
	if len(args) != 2 {
		return Value{Null: true}
	}
	list := args[0].Eval(fm)
	elem := args[1].Eval(fm)
	if list.Null || elem.Null {
		return Value{Null: true}
	}
	if list.Kind != TypeList {
		return fnBool(scalarEq(list, elem))
	}
	return fnBool(listContains(list.Data.([]Value), elem))
}

func fnArrayConcat(args []Expr, fm FrontMatter) Value {
	if len(args) == 0 {
		return Value{Null: true}
	}
	var out []Value
	for _, a := range args {
		v := a.Eval(fm)
		if v.Null {
			return Value{Null: true}
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
	return Value{Kind: TypeList, Data: out}
}

func fnDistinct(args []Expr, fm FrontMatter) Value {
	if len(args) != 1 {
		return Value{Null: true}
	}
	v := args[0].Eval(fm)
	if v.Null {
		return Value{Null: true}
	}
	if v.Kind != TypeList {
		return Value{Kind: TypeList, Data: []Value{v}}
	}
	src := v.Data.([]Value)
	out := make([]Value, 0, len(src))
	for _, x := range src {
		if !listContains(out, x) {
			out = append(out, x)
		}
	}
	return Value{Kind: TypeList, Data: out}
}

func fnArrayToString(args []Expr, fm FrontMatter) Value {
	if len(args) != 2 {
		return Value{Null: true}
	}
	v := args[0].Eval(fm)
	sep, ok := argStr(args, 1, fm)
	if !ok || v.Null {
		return Value{Null: true}
	}
	var elems []Value
	if v.Kind == TypeList {
		elems = v.Data.([]Value)
	} else {
		elems = []Value{v}
	}
	parts := make([]string, 0, len(elems))
	for _, e := range elems {
		s, err := Cast(e, TypeString)
		if err != nil || s.Null {
			return Value{Null: true}
		}
		parts = append(parts, s.Data.(string))
	}
	return fnStr(strings.Join(parts, sep))
}

func fnDatePart(args []Expr, fm FrontMatter, f func(time.Time) int64) Value {
	if len(args) != 1 {
		return Value{Null: true}
	}
	v := args[0].Eval(fm)
	if v.Null {
		return Value{Null: true}
	}
	if v.Kind != TypeDate && v.Kind != TypeDatetime {
		c, err := Cast(v, TypeDate)
		if err != nil {
			c2, err2 := Cast(v, TypeDatetime)
			if err2 != nil {
				return Value{Null: true}
			}
			c = c2
		}
		v = c
	}
	return fnInt(f(v.Data.(time.Time)))
}

func fnDateDiff(args []Expr, fm FrontMatter) Value {
	if len(args) != 2 {
		return Value{Null: true}
	}
	coerce := func(i int) (time.Time, bool) {
		v := args[i].Eval(fm)
		if v.Null {
			return time.Time{}, false
		}
		if v.Kind != TypeDate && v.Kind != TypeDatetime {
			c, err := Cast(v, TypeDate)
			if err != nil {
				return time.Time{}, false
			}
			v = c
		}
		return v.Data.(time.Time), true
	}
	a, ok1 := coerce(0)
	b, ok2 := coerce(1)
	if !ok1 || !ok2 {
		return Value{Null: true}
	}
	days := int64(a.Sub(b).Hours() / 24)
	return fnInt(days)
}
