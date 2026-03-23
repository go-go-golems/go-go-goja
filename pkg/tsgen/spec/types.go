package spec

// TypeKind represents a TypeScript type node kind.
type TypeKind string

const (
	TypeKindString  TypeKind = "string"
	TypeKindNumber  TypeKind = "number"
	TypeKindBoolean TypeKind = "boolean"
	TypeKindAny     TypeKind = "any"
	TypeKindUnknown TypeKind = "unknown"
	TypeKindVoid    TypeKind = "void"
	TypeKindNever   TypeKind = "never"
	TypeKindNamed   TypeKind = "named"
	TypeKindArray   TypeKind = "array"
	TypeKindUnion   TypeKind = "union"
	TypeKindObject  TypeKind = "object"
)

// Bundle contains all declaration modules rendered into a single output file.
type Bundle struct {
	HeaderComment string
	Modules       []*Module
}

// Module describes a single `declare module "<name>"` block.
type Module struct {
	Name        string
	Description string
	Functions   []Function
	RawDTS      []string
}

// Function describes a JS-exported function in a module.
type Function struct {
	Name        string
	Description string
	Params      []Param
	Returns     TypeRef
}

// Param describes a function parameter.
type Param struct {
	Name        string
	Type        TypeRef
	Optional    bool
	Variadic    bool
	Description string
}

// TypeRef describes a TypeScript type.
type TypeRef struct {
	Kind  TypeKind
	Name  string
	Item  *TypeRef
	Union []TypeRef
	// Fields are used when Kind == TypeKindObject.
	Fields []Field
}

// Field describes a TypeScript object field.
type Field struct {
	Name     string
	Type     TypeRef
	Optional bool
}
