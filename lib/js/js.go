package js

// Function definition
type Function struct {
	// Name must be unique and not conflict with the function names in "helper.js"
	Name string

	Definition string

	// Dependencies will be preloaded and assigned to the global js object "functions"
	Dependencies []*Function
}

var Functions = &Function{
	Name:         "functions",
	Definition:   "() => ({})",
	Dependencies: nil,
}
