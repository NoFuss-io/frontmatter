package internal

// Row is the projected output of SelectQuery.Eval for a single document.
// The slice index matches the order of SelectQuery.Fields.
type Row []Value
