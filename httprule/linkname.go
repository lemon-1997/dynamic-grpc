package httprule

import (
	_ "unsafe"
)

// Template is a compiled representation of path templates.
type Template struct {
	// Version is the version number of the format.
	Version int
	// OpCodes is a sequence of operations.
	OpCodes []int
	// Pool is a constant pool
	Pool []string
	// Verb is a VERB part in the template.
	Verb string
	// Fields is a list of field paths bound in this template.
	Fields []string
	// Original template (example: /v1/a_bit_of_everything)
	Template string
}

// Compiler compiles utilities representation of path templates into marshallable operations.
// They can be unmarshalled by runtime.NewPattern.
type Compiler interface {
	Compile() Template
}

//go:linkname Parse github.com/grpc-ecosystem/grpc-gateway/v2/internal/httprule.Parse
func Parse(tmpl string) (Compiler, error)
