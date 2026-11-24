package docs

import _ "embed"

//go:embed manual.md
var ManualMD []byte

//go:embed functions.md
var FunctionsMD []byte
