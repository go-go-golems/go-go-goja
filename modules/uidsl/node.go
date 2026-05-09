package uidsl

// Node is a normalized HTML node produced by ui.dsl.
type Node interface{ isNode() }

type Document struct {
	Title string
	Head  []Node
	Body  []Node
}

func (*Document) isNode() {}

type Element struct {
	Tag      string
	Attrs    map[string]any
	Children []Node
}

func (*Element) isNode() {}

type Text struct{ Value string }

func (*Text) isNode() {}

type RawHTML struct{ Value string }

func (*RawHTML) isNode() {}

type Fragment struct{ Children []Node }

func (*Fragment) isNode() {}
