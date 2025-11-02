package lsp

// Sig represents a method signature
type Sig struct {
	Method        string
	Detail        string
	Documentation string
	Frame         string
	Class         string
	IsStatic      bool
	FileName      string
	Row           int
}

// ClassNode represents a class in the inheritance hierarchy
type ClassNode struct {
	Frame string
	Class string
}
