package js

// Function definition
type Function struct {
	Name         string
	Definition   string
	Dependencies []*Function
}

var Functions = &Function{
	Name:         "functions",
	Definition:   "() => ({})",
	Dependencies: nil,
}
