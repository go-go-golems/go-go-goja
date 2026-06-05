---
Title: "08-zakaria-ast"
Source: external
LastUpdated: 2026-06-03
---

<div>    <p>I worked recently on a task that consists of adding Swagger docs comments (from the <a href="https://github.com/swaggo/swag">swag</a> project) to HTTP handler functions in a project written in Go. The swag project, for those who never heard of, is a tool that allows generating API specifications in Swagger (or OpenAPI) format. Since the size of the code base I was dealing with was considerable, I thought I’d explore ways to generate those comments automatically using code. In other words, writing code to generate code. One important factor that pushed me in this direction is the fact that the codebase was well structured and was following consistent naming conventions (Writing clean code always pays off!). In this post, I would like to share how I approached this problem.</p> <H2>Background:</H2> <p>My ultimate goal was to write a utility to modify the source code by adding doc comments above functions whose name was following a certain pattern. A clean and intuitive way of achieving this is by modifying the abstract syntax tree, or the AST. The AST constitues the bedrock of the compilation process, regardless of the language. Simply put, the AST represents all the constructs of a program as nodes of a tree data structure <sup id="fnref:1"><a href="#fn:1">1</a></sup>. The motivations behind having a program represented as a tree are numerous, but beyond the scope of this post. All we need to know is that our main requirement here is to build an AST and traverse it until we find the node that represent a well defined function and then add a comment statement to it ( not sure if there is a separate node for the comment, it does not matter)</p> <H2>Approach description:</H2> <p>One important charachteristic that drives people to adopt Golang as a programming language is its standard library. One can find a package for pretty much anything. In my case, the <code class="language-plaintext highlighter-rouge">go/ast</code> and the <code class="language-plaintext highlighter-rouge">go/parser</code> packages were what I was looking for. I believe it’s a great thing that Golang folks are exposing the functionality used to compile the language itself to the language users. The <code class="language-plaintext highlighter-rouge">parser</code> package has functionality to read a source code file and build an AST, mainly through the <a href="https://pkg.go.dev/go/parser#ParseFile">ParseFile</a> function while the <code class="language-plaintext highlighter-rouge">ast</code> package has functionality to traverse the AST, and also useful types to represent each construct within the tree.</p> <p>Going back to the source code, I had to document a large number of methods that serve as HTTP handlers. They were mainly two handler functions for each endpoint: one for GET and one for POST. The name of the functions starts with <code class="language-plaintext highlighter-rouge">Read</code> for the GET and starts with <code class="language-plaintext highlighter-rouge">Create</code> for the POST. For example:</p> <pre><code class="language-go" data-lang="go">func (h *AlertHandler) ReadAlert(w http.ResponseWriter, r *http.Request) {
    //Get handler 
}

func (h *AlertHandler) CreateAlert(w http.ResponseWriter, r *http.Request) {
    //POST handler
}</code></pre> <p>The first step is to parse the source files:</p> <pre><code class="language-go" data-lang="go">import (
    "go/ast"
    "go/parser"
    "go/token"
    "log"
    "os"
)

sourceBaseDir := "./pkg/handlers"
sourceFiles, err := os.ReadDir(sourceBaseDir)
if err != nil {
    log.Fatal(err)
}
for _, sf := range sourceFiles {
 fset := token.NewFileSet()
 f, err := parser.ParseFile(fset, sf, nil, parser.ParseComments)
 if err != nil {
    log.Fatal(err)
  }
  //more to come
}</code></pre> <p>Because I am dealing with comments, I have to set the mode to <code class="language-plaintext highlighter-rouge">ParseComments</code> for the <code class="language-plaintext highlighter-rouge">ParseFile</code> method. Otherwise, all the comments will be discarded.</p> <p>The next step is building a comment map using the <code class="language-plaintext highlighter-rouge">ast.NewCommentMap</code> function. A <a href="https://pkg.go.dev/go/ast#CommentMap">CommentMap</a> maps between tree nodes and their comments. This is another hassle, made easy by the <code class="language-plaintext highlighter-rouge">ast</code> package. All we have to do is to call <code class="language-plaintext highlighter-rouge">NewCommentMap</code>:</p> <pre><code class="language-go" data-lang="go">import (
    "go/ast"
)

commentMap := ast.NewCommentMap(fset, f, f.Comments)</code></pre> <p>Now, we need to inspect the AST to look for any function that starts with either <code class="language-plaintext highlighter-rouge">Read</code> or <code class="language-plaintext highlighter-rouge">Create</code>. This can be achieved by calling <a href="https://pkg.go.dev/go/ast#Inspect">ast.Inspect</a> on the root node:</p> <pre><code class="language-go" data-lang="go">import (
    "go/ast"
)

ast.Inspect(f, func(n ast.Node) bool {
    switch x := n.(type) {
        //if this is a function declation
    case *ast.FuncDecl:
        if strings.HasPrefix(x.Name.Name, "Read") {
            //add swagger comment for GET 
        } else if strings.HasPrefix(x.Name.Name, "Create") {
            //add swagger comment for POST 
        }
    }
    return true
})</code></pre> <p>We have found our function nodes within the AST, coolness! The next step is to add function comments. Let’s assume here that the <code class="language-plaintext highlighter-rouge">generateComment</code> function creates the swagger comments based on the function name and decides whether to generate comments for either the GET or the POST. Here is an example of the template that I used for POST:</p> <pre><code>// @ID {{ .HandlerName }}-post
// @Tags {{ .HandlerNameCap }}
// @Param payload body {{ .HandlerNameCap }} true "request"
// @Accept  application/json
// @Produce  json
// @Success 200 {object} {{ .HandlerNameCap }}
// @Failure 400 {object} core.ErrorMessage
// @Router /{{ .HandlerName }} [{{ .Method }}]</code></pre> <p>As mentionned earlier, the code base was following strict naming convention in a way that makes it easy to predict the endpoint name, the returned object, the request body parameters…etc. The example above is for the <code class="language-plaintext highlighter-rouge">Create</code> prefixed functions. Now our inspection code looks like:</p> <pre><code class="language-go" data-lang="go">import (
    "go/ast"
    "go/token"
)

ast.Inspect(f, func(n ast.Node) bool {
    switch x := n.(type) {
        //if this is a function declation
    case *ast.FuncDecl:
        if strings.HasPrefix(x.Name.Name, "Read") || strings.HasPrefix(x.Name.Name, "Create") {
              commentText := generateComment(x.Name.Name)
              commentMap[x] = []*ast.CommentGroup{{List: []*ast.Comment{{Text: commentText, Slash: token.Pos(int(x.Pos() - 1))}}}}
        } 
    }
    return true
})</code></pre> <p><code class="language-plaintext highlighter-rouge">token.Pos(int(x.Pos() - 1))</code> means that the comment is to be inserted at the line right above the function.</p> <p>We should not forget off course to write back the result to disk (we can here overwrite the original file). For this purpose, I used the <code class="language-plaintext highlighter-rouge">go/printer</code> package which can translate an AST back to code and write it to a <code class="language-plaintext highlighter-rouge">Writer</code>:</p> <pre><code class="language-go" data-lang="go">import (
    "go/printer"
    "os"
)

srcFile, err := os.Create(sf)
if err != nil {
    log.Fatal(err)
}

err = printer.Fprint(srcFile, fset, f)
if err != nil {
    log.Fatal(err)
}</code></pre> <p>Result:</p> <pre><code class="language-go" data-lang="go">// @ID alert-post
// @Tags Alert
// @Param payload body Alert true "request"
// @Accept  application/json
// @Produce  json
// @Success 200 {object} Alert
// @Failure 400 {object} core.ErrorMessage
// @Router /alert [post]
func (h *AlertHandler) CreateAlert(w http.ResponseWriter, r *http.Request) {
    //...
}</code></pre> <p>Not only I was able to automate the process and save myself long boring moments of doing copy and paste, but also I made it easy to introduce new changes and updates to the doc comments at the blink of an eye. Automation and code generation can be a blessing sometimes.</p>      <div id="footnotes"><ol><li id="fn:1"><p><a href="https://ocw.mit.edu/courses/6-004-computation-structures-spring-2017/pages/c11/c11s1/">https://ocw.mit.edu/courses/6-004-computation-structures-spring-2017/pages/c11/c11s1/</a> <a class="footnote-backref" title="return to article" href="#fnref:1">↩</a></p></li></ol></div></div>
