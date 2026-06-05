---
Title: "04-eli-bendersky-ast"
Source: external
LastUpdated: 2026-06-03
---

<div> <p>Go is well-known for having great tooling for analyzing code written in the language, right in the standard library with the <tt>go/*</tt> packages (<tt>go/parser</tt>, <tt>go/ast</tt>, <tt>go/types</tt> etc.); in addition, the <a href="https://pkg.go.dev/golang.org/x/tools">golang.org/x/tools</a> module contains several supplemental packages that are even more powerful. I've used one of these packages to describe how to write multi-package analysis tools in a <a href="https://eli.thegreenplace.net/2020/writing-multi-package-analysis-tools-for-go/">post from last year</a>.</p> <p>Here I want to write about a slightly different task: <em>rewriting</em> Go source code using AST-based tooling. I will begin by providing a quick introduction to how existing capabilities of the stdlib <tt>go/ast</tt> package can be used to find points of interest in an AST. Then, I'll show how some simple rewrites can be done with the <tt>go/ast</tt> package without requiring additional tooling. Finally, I'll discuss the limitations of this approach and the <tt>golang.org/x/tools/astutil</tt> package which provides much more powerful AST editing capabilities.</p> <p>This post assumes some basic level of familiarity with ASTs (Abstract Syntax Trees) in general, and ASTs for Go in particular.</p> <div> <H2>Finding points of interest in a Go AST</H2> <p>Throughout this post, we're going to be using the following simple Go snippet as our lab rat:</p> <pre><code>package p

func pred() bool {
  return true
}

func pp(x int) int {
  if x &gt; 2 &amp;&amp; pred() {
    return 5
  }

  var b = pred()
  if b {
    return 6
  }
  return 0
}</code></pre> <p>Let's start by finding all calls to the <tt>pred</tt> function in this code. The <tt>go/ast</tt> package provides two approaches for finding points of interest in the code. First, we'll discuss <tt>ast.Walk</tt>. The full code sample for this part is <a href="https://github.com/eliben/code-for-blog/blob/main/2021/go-ast-rewrite/ast-find-call-visit.go">on GitHub</a>. We begin by parsing the source code (which we'll be piping into standard input):</p> <pre><code>fset := token.NewFileSet()
file, err := parser.ParseFile(fset, "src.go", os.Stdin, 0)
if err != nil {
  log.Fatal(err)
}</code></pre> <p>Now we create a new value implementing the <tt>ast.Visitor</tt> interface and call <tt>ast.Walk</tt>:</p> <pre><code>visitor := &amp;Visitor{fset: fset}
ast.Walk(visitor, file)</code></pre> <p>Finally, the interesting part of the code is the <tt>Visitor</tt> type:</p> <pre><code>type Visitor struct {
  fset *token.FileSet
}

func (v *Visitor) Visit(n ast.Node) ast.Visitor {
  if n == nil {
    return nil
  }

  switch x := n.(type) {
  case *ast.CallExpr:
    id, ok := x.Fun.(*ast.Ident)
    if ok {
      if id.Name == "pred" {
        fmt.Printf("Visit found call to pred() at %s\n", v.fset.Position(n.Pos()))
      }
    }
  }
  return v
}</code></pre> <p>Our visitor is only interested in AST nodes of type <tt>CallExpr</tt>. Once it sees such a node, it checks the name of the called function, and reports matches. Note the type assertion on <tt>x.Fun</tt>; we only want to report calls when the function is referred to by an <tt>ast.Ident</tt>. In Go, we could call functions in other ways, like invoking anonymous functions directly - e.g. <tt>func(){}()</tt>.</p> <p>We have a <tt>FileSet</tt> stored in the visitor; this is only used here to report positions in the parsed code properly. To save space, the AST stores all position information in a single <tt>int</tt> (aliased as the <tt>token.Pos</tt> type), and the <tt>FileSet</tt> is required to translate these numbers into actual positions of the expected <tt>&lt;filename&gt;:line:column</tt> form.</p> </div> <div> <H2>Visualizing the Go AST</H2> <p>At this point it's worth mentioning some useful tools that help writing analyzers for Go ASTs. First and foremost, the <tt>go/ast</tt> package has a <tt>Print</tt> function that will emit an AST in a textual format. Here's how the full <tt>if</tt> statement in our code snippet would look if printed this way:</p> <pre><code>.  .  1: *ast.IfStmt {
.  .  .  If: 9:2
.  .  .  Cond: *ast.BinaryExpr {
.  .  .  .  X: *ast.BinaryExpr {
.  .  .  .  .  X: *ast.Ident {
.  .  .  .  .  .  NamePos: 9:5
.  .  .  .  .  .  Name: "x"
.  .  .  .  .  .  Obj: *(obj @ 72)
.  .  .  .  .  }
.  .  .  .  .  OpPos: 9:7
.  .  .  .  .  Op: &gt;
.  .  .  .  .  Y: *ast.BasicLit {
.  .  .  .  .  .  ValuePos: 9:9
.  .  .  .  .  .  Kind: INT
.  .  .  .  .  .  Value: "2"
.  .  .  .  .  }
.  .  .  .  }
.  .  .  .  OpPos: 9:11
.  .  .  .  Op: &amp;&amp;
.  .  .  .  Y: *ast.CallExpr {
.  .  .  .  .  Fun: *ast.Ident {
.  .  .  .  .  .  NamePos: 9:14
.  .  .  .  .  .  Name: "pred"
.  .  .  .  .  .  Obj: *(obj @ 11)
.  .  .  .  .  }
.  .  .  .  .  Lparen: 9:18
.  .  .  .  .  Ellipsis: -
.  .  .  .  .  Rparen: 9:19
.  .  .  .  }
.  .  .  }
.  .  .  Body: *ast.BlockStmt {
.  .  .  .  Lbrace: 9:21
.  .  .  .  List: []ast.Stmt (len = 1) {
.  .  .  .  .  0: *ast.ReturnStmt {
.  .  .  .  .  .  Return: 10:3
.  .  .  .  .  .  Results: []ast.Expr (len = 1) {
.  .  .  .  .  .  .  0: *ast.BasicLit {
.  .  .  .  .  .  .  .  ValuePos: 10:10
.  .  .  .  .  .  .  .  Kind: INT
.  .  .  .  .  .  .  .  Value: "5"
.  .  .  .  .  .  .  }
.  .  .  .  .  .  }
.  .  .  .  .  }
.  .  .  .  }
.  .  .  .  Rbrace: 11:2
.  .  .  }</code></pre> <p>A somewhat more interactive way to explore this AST dump is using the web page at <a href="http://goast.yuroyoro.net/">http://goast.yuroyoro.net/</a>, where you can paste your source and get an AST dump with expandable and collapsible sections. This helps focus only on parts we're interested in; here's an extract from our AST:</p> <img alt="Screenshot of graphical AST dump tool" src="https://eli.thegreenplace.net/images/2021/goast-dump-expand.png"> <p>(Update: <a href="https://astexplorer.net/">https://astexplorer.net/</a> is an even nicer AST explorer)</p> </div> <div> <H2>Using the ast.Inspect API</H2> <p>Using <tt>ast.Walk</tt> for finding interesting nodes is pretty straightforward, but it requires scaffolding that feels a bit heavy for simple needs - defining a custom type that implements the <tt>ast.Visitor</tt> interface, and so on. Luckily, the <tt>go/ast</tt> package provides a lighter-weight API - <tt>Inspect</tt>; it only needs to be provided a closure. Here's our program to find calls to <tt>pred()</tt> rewritten with <tt>ast.Inspect</tt>:</p> <pre><code>func main() {
  fset := token.NewFileSet()
  file, err := parser.ParseFile(fset, "src.go", os.Stdin, 0)
  if err != nil {
    log.Fatal(err)
  }

  ast.Inspect(file, func(n ast.Node) bool {
    switch x := n.(type) {
    case *ast.CallExpr:
      id, ok := x.Fun.(*ast.Ident)
      if ok {
        if id.Name == "pred" {
          fmt.Printf("Inspect found call to pred() at %s\n", fset.Position(n.Pos()))
        }
      }
    }
    return true
  })
}</code></pre> <p>The actual AST node matching logic is the same, but the surrounding code is somewhat simpler. Unless there's a strong need to use <tt>ast.Walk</tt> specifically, <tt>ast.Inspect</tt> is the approach I recommend, and it's the one we'll be using in the next section to actually rewrite the AST.</p> </div> <div> <H2>Simple AST rewrites</H2> <p>To begin, it's important to highlight that the AST returned by the parser is a mutable object. It's a collection of node values interconnected via pointers to each other. We can change this set of nodes in any way we wish - or even create a wholly new set of nodes - and then use the <tt>go/format</tt> package to emit Go formatted source code back from the AST. The following program will simply emit back the Go program it's provided (though it will drop the comments with the default configuration):</p> <pre><code>func main() {
  fset := token.NewFileSet()
  file, err := parser.ParseFile(fset, "src.go", os.Stdin, 0)
  if err != nil {
    log.Fatal(err)
  }

  format.Node(os.Stdout, fset, file)
}</code></pre> <p>Now, back to rewriting that AST. Let's make a couple of changes:</p> <ol> <li>We'll rename the function <tt>pred</tt> to <tt>pred2</tt>, and rename all the call sites to call the new function name.</li> <li>We'll inject a printout into the beginning of each function body - emulating some sort of instrumentation we could add this way.</li> </ol> <p>Given the original code snippet, the output will look like this (with the changed/new lines highlighted):</p> <pre><code>package p

func pred2() bool {
  fmt.Println("instrumentation")
  return true
}

func pp(x int) int {
  fmt.Println("instrumentation")
  if x &gt; 2 &amp;&amp; pred2() {
    return 5
  }

  var b = pred2()
  if b {
    return 6
  }
  return 0
}</code></pre> <p><em>[Note: we're not adding an import of fmt here - this is left as an exercise for the reader]</em></p> <p>The full code of our rewriting program is available <a href="https://github.com/eliben/code-for-blog/blob/main/2021/go-ast-rewrite/ast-inspect-rewrite.go">here</a>. It's using <tt>ast.Inspect</tt> to find the nodes it wants to operate on. Here's the renaming of the call sites:</p> <pre><code>ast.Inspect(file, func(n ast.Node) bool {
  switch x := n.(type) {
  case *ast.CallExpr:
    id, ok := x.Fun.(*ast.Ident)
    if ok {
      if id.Name == "pred" {
        id.Name += "2"
      }
    }
    // ...</code></pre> <p>If the function is called by an identifier, the code just appends <tt>"2"</tt> to the name. Again, we're not operating on some copy of the AST - this is the <em>real, living</em> AST we're editing here.</p> <p>Now let's move on to the next <tt>case</tt>, where we're handing function declarations:</p> <pre><code>case *ast.FuncDecl:
  if x.Name.Name == "pred" {
    x.Name.Name += "2"
  }

  newCallStmt := &amp;ast.ExprStmt{
    X: &amp;ast.CallExpr{
      Fun: &amp;ast.SelectorExpr{
        X: &amp;ast.Ident{
          Name: "fmt",
        },
        Sel: &amp;ast.Ident{
          Name: "Println",
        },
      },
      Args: []ast.Expr{
        &amp;ast.BasicLit{
          Kind:  token.STRING,
          Value: `"instrumentation"`,
        },
      },
    },
  }

  x.Body.List = append([]ast.Stmt{newCallStmt}, x.Body.List...)</code></pre> <p>The first three lines in this <tt>case</tt> do the same as we did for the call sites - just rename the <tt>pred</tt> function to <tt>pred2</tt>. The rest of the code is adding the printout to the start of a function body.</p> <p>That task is fairly easy to accomplish since each <tt>FuncDecl</tt> has a <tt>Body</tt> which is an <tt>ast.StmtList</tt>, which itself holds a slice of <tt>ast.Stmt</tt> in its <tt>List</tt> attribute. Out program prepends a new expression to this slice, in effect adding a new statement to the very beginning of the function body. The statement is a hand-crafted AST node. You must be thinking - how did I know how to build this node?</p> <p>It's really not a big deal once you get the hang of it. Parsing small snippets of code and dumping their ASTs helps, as well as the detailed documentation of the <tt>go/ast</tt> package. I also found the <a href="https://github.com/reflog/go2ast">go2ast</a> tool very useful; it takes a piece of code and emits exactly the Go code needed to build its AST.</p> <p>Finally, at the end of the program we emit back the modified AST:</p> <pre><code>fmt.Println("Modified AST:")
format.Node(os.Stdout, fset, file)</code></pre> <p>And this gets us the modified snippet shown at the beginning of this section.</p> </div> <H2>Limitations of AST editing with Walk and Inspect</H2> <p>So far we've managed to rewrite the AST in a couple of interesting ways using <tt>ast.Inspect</tt> for finding the nodes. Can we do <em>any</em> kind of rewrite this way?</p> <p>It turns out the answer to this question is <strong>no</strong>, or at least not easily. As a motivating example, consider the following task: we'd like to rewrite each call to <tt>pred()</tt> so that it's logically negated, or turns into <tt>!pred()</tt>. How do we do that?</p> <p>It's worth spending a few minutes thinking about this question before reading on.</p> <p>The issue is that when <tt>ast.Inspect</tt> (or <tt>ast.Walk</tt>) hands us an <tt>ast.Node</tt>, we can change the node's contents and its children, but we cannot replace the node itself. To replace the node itself, we'd need access to its parent, but <tt>ast.Inspect</tt> does not give us any way to access its parent. A different, slightly more technical way to think about it is: we get handed a node pointer <em>by value</em>, meaning that we can tweak the node it points to, but can't set the pointer to point to a different node. To achieve the latter, <tt>ast.Inspect</tt> would have to hand us a pointer to a pointer to the node.</p> <p>This limitation was discussed <a href="https://github.com/golang/go/issues/17108">several years ago</a>, and finally in 2017 a new package appeared in the "extended stdlib" <tt>golang.org/x/tools</tt> module - <a href="https://pkg.go.dev/golang.org/x/tools/go/ast/astutil">astutil</a>.</p> <div> <H2>More powerful rewriting with astutil</H2> <p>The APIs <tt>astutil</tt> provides let us not only find nodes of interest in the AST, but also a way to replace the node itself, not just its contents. In fact, the package provides several useful helpers to delete, replace and insert new nodes through the <tt>Cursor</tt> type. A full walkthrough of the capabilities of <tt>astutil</tt> is outside the scope of this post, but I will show how to use it in order to implement our task of turning each <tt>pred()</tt> into <tt>!pred()</tt>. Here we go:</p> <pre><code>func main() {
  fset := token.NewFileSet()
  file, err := parser.ParseFile(fset, "src.go", os.Stdin, 0)
  if err != nil {
    log.Fatal(err)
  }

  astutil.Apply(file, nil, func(c *astutil.Cursor) bool {
    n := c.Node()
    switch x := n.(type) {
    case *ast.CallExpr:
      id, ok := x.Fun.(*ast.Ident)
      if ok {
        if id.Name == "pred" {
          c.Replace(&amp;ast.UnaryExpr{
            Op: token.NOT,
            X:  x,
          })
        }
      }
    }

    return true
  })

  fmt.Println("Modified AST:")
  format.Node(os.Stdout, fset, file)
}</code></pre> <p>Instead of calling <tt>ast.Inspect</tt>, we call <tt>astutil.Apply</tt>, which also walks the AST recursively and gives our closure access to the node. <tt>Apply</tt> lets us register a callback for the node both <em>before</em> and <em>after</em> it was visited; in this case we only provide the <em>after</em> case.</p> <p>Our closure identifies the call to <tt>pred</tt> in a way that should be similar by now. It then uses the <tt>Cursor</tt> type to replace this node by a new one which is just the same node wrapped in a unary <tt>NOT</tt> expression. Hidden in its implementation, the <tt>Cursor</tt> type does have access to the parent of each node, making it possible to replace the actual node with something else.</p> </div> </div>
