package spec

func String() TypeRef  { return TypeRef{Kind: TypeKindString} }
func Number() TypeRef  { return TypeRef{Kind: TypeKindNumber} }
func Boolean() TypeRef { return TypeRef{Kind: TypeKindBoolean} }
func Any() TypeRef     { return TypeRef{Kind: TypeKindAny} }
func Unknown() TypeRef { return TypeRef{Kind: TypeKindUnknown} }
func Void() TypeRef    { return TypeRef{Kind: TypeKindVoid} }
func Never() TypeRef   { return TypeRef{Kind: TypeKindNever} }

func Named(name string) TypeRef {
	return TypeRef{
		Kind: TypeKindNamed,
		Name: name,
	}
}

func Array(item TypeRef) TypeRef {
	return TypeRef{
		Kind: TypeKindArray,
		Item: &item,
	}
}

func Union(items ...TypeRef) TypeRef {
	return TypeRef{
		Kind:  TypeKindUnion,
		Union: append([]TypeRef(nil), items...),
	}
}

func Object(fields ...Field) TypeRef {
	return TypeRef{
		Kind:   TypeKindObject,
		Fields: append([]Field(nil), fields...),
	}
}
