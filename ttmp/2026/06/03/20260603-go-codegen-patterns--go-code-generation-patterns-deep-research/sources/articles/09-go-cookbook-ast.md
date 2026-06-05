---
Title: "09-go-cookbook-ast"
Source: external
LastUpdated: 2026-06-03
---

<main><H2>AST Manipulation</H2> <p>Learn how to manipulate Abstract Syntax Trees (AST) in Go using the go/ast and go/token packages</p> <div><p>Go provides powerful capabilities for manipulating source code using Abstract Syntax Trees (AST). This guide covers the basics of traversing and modifying ASTs with Go's <code>go/ast</code> and <code>go/token</code> packages.</p> <H2>Basic AST Traversal</H2> <p>Here's a simple example that shows how to parse a Go source file and traverse its AST:</p> <div><pre><code>package main

import (
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "log"
)

func main() {
    src := `
    package main

    func main() {
        fmt.Println("Hello, Go AST!")
    }
    `

    // Create a new token.FileSet for tracking position information.
    fset := token.NewFileSet()

    // Parse the source code and generate an AST.
    node, err := parser.ParseFile(fset, "", src, parser.AllErrors)
    if err != nil {
        log.Fatal(err)
    }

    // Traverse the AST.
    ast.Inspect(node, func(n ast.Node) bool {
        switch x := n.(type) {
        case *ast.FuncDecl:
            fmt.Printf("Function declaration: %s\n", x.Name.Name)
        }
        return true
    })
}</code></pre></div> <H2>Modifying AST Nodes</H2> <p>Modify AST nodes to transform code. Here's an example that changes function names:</p> <div><pre><code>package main

import (
    "go/ast"
    "go/parser"
    "go/printer"
    "go/token"
    "log"
    "os"
)

func main() {
    src := `
    package main

    func oldName() {
        fmt.Println("Old function!")
    }
    `

    fset := token.NewFileSet()
    node, err := parser.ParseFile(fset, "", src, parser.AllErrors)
    if err != nil {
        log.Fatal(err)
    }

    // Modify function names.
    ast.Inspect(node, func(n ast.Node) bool {
        if fn, ok := n.(*ast.FuncDecl); ok {
            if fn.Name.Name == "oldName" {
                fn.Name.Name = "newName"
            }
        }
        return true
    })

    // Print the modified code.
    printer.Fprint(os.Stdout, fset, node)
}</code></pre></div> <H2>AST Node Insertion</H2> <p>Insert new nodes to extend an AST.</p> <div><pre><code>package main

import (
    "go/ast"
    "go/parser"
    "go/token"
    "go/printer"
    "os"
    "log"
)

func main() {
    src := `
    package main

    func main() {
        fmt.Println("Start")
    }
    `

    fset := token.NewFileSet()
    node, err := parser.ParseFile(fset, "", src, parser.AllErrors)
    if err != nil {
        log.Fatal(err)
    }

    // Create a new statement to insert.
    newCall := &amp;ast.ExprStmt{
        X: &amp;ast.CallExpr{
            Fun: &amp;ast.SelectorExpr{
                X:   ast.NewIdent("fmt"),
                Sel: ast.NewIdent("Println"),
            },
            Args: []ast.Expr{
                &amp;ast.BasicLit{
                    Kind:  token.STRING,
                    Value: "\"New statement\"",
                },
            },
        },
    }

    // Insert the new statement into the function body.
    for _, decl := range node.Decls {
        if fn, ok := decl.(*ast.FuncDecl); ok &amp;&amp; fn.Name.Name == "main" {
            fn.Body.List = append(fn.Body.List, newCall)
        }
    }

    // Print the modified code.
    printer.Fprint(os.Stdout, fset, node)
}</code></pre></div> <H2>Best Practices</H2> <ul> <li>Always use <code>token.FileSet</code> to maintain accurate position information.</li> <li>Use <code>ast.Inspect</code> for simplified AST traversal.</li> <li>Update nodes in a depth-first order to ensure dependent nodes are processed correctly.</li> <li>Maintain code formatting using <code>go/printer</code> to output modified ASTs neatly.</li> </ul> <H2>Common Pitfalls</H2> <ul> <li>Overlooking token position management can lead to incorrect code transformations.</li> <li>Forgetting to validate node types before casting can cause runtime panics.</li> <li>Assuming tree node structures remain constant across different Go versions.</li> </ul> <H2>Performance Tips</H2> <ul> <li>For large codebases, leverage parallel processing when analyzing multiple files.</li> <li>Cache results from <code>token.FileSet</code> as it can be reused for multiple file parses.</li> <li>Avoid excessive AST traversals by combining related transformations within a single pass.</li> </ul> </div></main>
