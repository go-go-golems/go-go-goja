---
Title: "06-caddy-extending"
Source: external
LastUpdated: 2026-06-03
---

<article> <H2>Extending Caddy</H2> <p>Caddy is easy to extend because of its modular architecture. Most kinds of Caddy extensions (or plugins) are known as <em>modules</em> if they extend or plug into Caddy's configuration structure. To be clear, Caddy modules are distinct from <a href="https://github.com/golang/go/wiki/Modules">Go modules</a> (but they are also Go modules).</p> <p><strong>Prerequisites:</strong></p> <ul> <li>Basic understanding of <a href="https://caddyserver.com/docs/architecture">Caddy's architecture</a></li> <li>Go language proficiency</li> <li><a href="https://golang.org/doc/install"><code>go</code> <img src="https://caddyserver.com/old/resources/images/external-link.svg"></a></li> <li><a href="https://github.com/caddyserver/xcaddy"><code>xcaddy</code> <img src="https://caddyserver.com/old/resources/images/external-link.svg"></a></li> </ul> <H2>Quick Start</H2> <p>A Caddy module is any named type that registers itself as a Caddy module when its package is imported. Crucially, a module always implements the <a href="https://pkg.go.dev/github.com/caddyserver/caddy/v2?tab=doc#Module"><code>caddy.Module</code></a> interface, which provides its name and a constructor function.</p> <p>In a new Go module, paste the following template into a Go file and customize your package name, type name, and Caddy module ID:</p> <pre><code>package mymodule

import "github.com/caddyserver/caddy/v2"

func init() {
    caddy.RegisterModule(Gizmo{})
}

// Gizmo is an example; put your own type here.
type Gizmo struct {
}

// CaddyModule returns the Caddy module information.
func (Gizmo) CaddyModule() caddy.ModuleInfo {
    return caddy.ModuleInfo{
        ID:  "foo.gizmo",
        New: func() caddy.Module { return new(Gizmo) },
    }
}</code></pre> <p>Then run this command from your project's directory, and you should see your module in the list:</p> <pre><code class="language-bash" data-lang="bash">xcaddy list-modules
...
foo.gizmo
...</code></pre>  <p>Congratulations, your module registers with Caddy and can be used in <a href="https://caddyserver.com/docs/json/">Caddy's config document</a> in whatever places use modules in the same namespace.</p> <p>Under the hood, <code>xcaddy</code> is simply making a new Go module that requires both Caddy and your plugin (with an appropriate <code>replace</code> to use your local development version), then adds an import to ensure it is compiled in:</p> <pre><code>import _ "github.com/example/mymodule"</code></pre> <H2>Module Basics</H2> <p>Caddy modules:</p> <ol> <li>Implement the <code>caddy.Module</code> interface to provide an ID and constructor</li> <li>Have a unique name in the proper namespace</li> <li>Usually satisfy some interface(s) that are meaningful to the host module for that namespace</li> </ol> <p><strong>Host modules</strong> (or <em>parent modules</em>) are modules which load/initialize other modules. They typically define namespaces for guest modules.</p> <p><strong>Guest modules</strong> (or <em>child modules</em>) are modules which get loaded or initialized. All modules are guest modules.</p> <H2>Module IDs</H2> <p>Each Caddy module has a unique ID, consisting of a namespace and name:</p> <ul> <li>A complete ID looks like <code>foo.bar.module_name</code></li> <li>The namespace would be <code>foo.bar</code></li> <li>The name would be <code>module_name</code> which must be unique in its namespace</li> </ul> <p>Module IDs must use <code>snake_case</code> convention.</p> <H3>Namespaces</H3> <p>Namespaces are like classes, i.e. a namespace defines some functionality that is common among all modules within it. For example, we can expect that all modules within the <code>http.handlers</code> namespace are HTTP handlers. It follows that a host module may type-assert guest modules in that namespace from <code>interface{}</code> types into a more specific, useful type such as <code>caddyhttp.MiddlewareHandler</code>.</p> <p>A guest module must be properly namespaced in order for it to be recognized by a host module because host modules will ask Caddy for modules within a certain namespace to provide the functionality desired by the host module. For example, if you were to write an HTTP handler module called <code>gizmo</code>, your module's name would be <code>http.handlers.gizmo</code>, because the <code>http</code> app will look for handlers in the <code>http.handlers</code> namespace.</p> <p>Put another way, Caddy modules are expected to implement <a href="https://caddyserver.com/docs/extending-caddy/namespaces">certain interfaces</a> depending on their module namespace. With this convention, module developers can say intuitive things such as, "All modules in the <code>http.handlers</code> namespace are HTTP handlers." More technically, this usually means, "All modules in the <code>http.handlers</code> namespace implement the <code>caddyhttp.MiddlewareHandler</code> interface." Because that method set is known, the more specific type can be asserted and used.</p> <p><strong><a href="https://caddyserver.com/docs/extending-caddy/namespaces">View a table mapping all the standard Caddy namespaces to their Go types.</a></strong></p> <p>The <code>caddy</code> and <code>admin</code> namespaces are reserved and cannot be app names.</p> <p>To write modules which plug into 3rd-party host modules, consult those modules for their namespace documentation.</p> <H3>Names</H3> <p>The name within a namespace is significant and highly visible to users, but is not particularly important, as long as it is unique, concise, and makes sense for what it does.</p> <H2>App Modules</H2> <p>Apps are modules with an empty namespace, and which conventionally become their own top-level namespace. App modules implement the <a href="https://pkg.go.dev/github.com/caddyserver/caddy/v2?tab=doc#App"><code>caddy.App</code></a> interface.</p> <p>These modules appear in the <a href="https://caddyserver.com/docs/json/#apps"><code>"apps"</code></a> property of the top-level of Caddy's config:</p> <pre><code>{
    "apps": {}
}</code></pre> <p>Example <a href="https://caddyserver.com/docs/json/apps/">apps</a> are <code>http</code> and <code>tls</code>. Theirs is the empty namespace.</p> <p>Guest modules written for these apps should be in a namespace derived from the app name. For example, HTTP handlers use the <code>http.handlers</code> namespace and TLS certificate loaders use the <code>tls.certificates</code> namespace.</p> <H2>Module Implementation</H2> <p>A module can be virtually any type, but structs are the most common because they can hold user configuration.</p> <H3>Configuration</H3> <p>Most modules require some configuration. Caddy takes care of this automatically, as long as your type is compatible with JSON. Thus, if a module is a struct type, it will need struct tags on its fields, which should use <code>snake_casing</code> according to Caddy convention:</p> <pre><code>type Gizmo struct {
    MyField string `json:"my_field,omitempty"`
    Number  int    `json:"number,omitempty"`
}</code></pre> <p>Using the <code>omitempty</code> option in the struct tag will omit the field from the JSON output if it is the zero value for its type. This is useful to keep the JSON config clean and concise when marshaled (e.g. adapting from Caddyfile to JSON).</p> <p>When a module is initialized, it will already have its configuration filled out. It is also possible to perform additional <a href="#provisioning">provisioning</a> and <a href="#validating">validation</a> steps after a module is initialized.</p> <H3>Module Lifecycle</H3> <p>A module's life begins when it is loaded by a host module. The following happens:</p> <ol> <li><a href="https://pkg.go.dev/github.com/caddyserver/caddy/v2?tab=doc#ModuleInfo.New"><code>New()</code></a> is called to get an instance of the module's value.</li> <li>The module's configuration is unmarshaled into that instance.</li> <li>If the module is a <a href="https://pkg.go.dev/github.com/caddyserver/caddy/v2?tab=doc#Provisioner"><code>caddy.Provisioner</code></a>, the <code>Provision()</code> method is called.</li> <li>If the module is a <a href="https://pkg.go.dev/github.com/caddyserver/caddy/v2?tab=doc#Validator"><code>caddy.Validator</code></a>, the <code>Validate()</code> method is called.</li> <li>At this point, the host module is given the loaded guest module as an <code>interface{}</code> value, so the host module will usually type-assert the guest module into a more useful type. Check the documentation for the host module to know what is required of a guest module in its namespace, e.g. what methods need to be implemented.</li> <li>When a module is no longer needed, and if it is a <a href="https://pkg.go.dev/github.com/caddyserver/caddy/v2?tab=doc#CleanerUpper"><code>caddy.CleanerUpper</code></a>, the <code>Cleanup()</code> method is called.</li> </ol> <p>Note that multiple loaded instances of your module may overlap at a given time! During config changes, new modules are started before the old ones are stopped. Be sure to use global state carefully. Use the <a href="https://pkg.go.dev/github.com/caddyserver/caddy/v2#UsagePool"><code>caddy.UsagePool</code></a> type to help manage global state across module loads. If your module listens on a socket, use <code>caddy.Listen*()</code> to get a socket that supports overlapping usage.</p> <H3>Provisioning</H3> <p>A module's configuration will be unmarshaled into its value automatically (when loading the JSON config). This means, for example, that struct fields will be filled out for you.</p> <p>However, if your module requires additional provisioning steps, you can implement the (optional) <a href="https://pkg.go.dev/github.com/caddyserver/caddy/v2?tab=doc#Provisioner"><code>caddy.Provisioner</code></a> interface:</p> <pre><code>// Provision sets up the module.
func (g *Gizmo) Provision(ctx caddy.Context) error {
    // TODO: set up the module
    return nil
}</code></pre> <p>This is where you should set default values for fields that were not provided by the user (fields that are not their zero value). If a field is required, you may return an error if it is not set. For numeric fields where the zero value has meaning (e.g. some timeout duration), you may want to support <code>-1</code> to mean "off" rather than <code>0</code>, so you may set a default value if the user did not configure it.</p> <p>This is also typically where host modules will load their guest/child modules.</p> <p>A module may access other apps by calling <code>ctx.App()</code>, but modules must not have circular dependencies. In other words, a module loaded by the <code>http</code> app cannot depend on the <code>tls</code> app if a module loaded by the <code>tls</code> app depends on the <code>http</code> app. (Very similar to rules forbidding import cycles in Go.)</p> <p>Additionally, you should avoid performing expensive operations in <code>Provision</code>, since provisioning is performed even if a config is only being validated. When in the provisioning phase, do not expect that the module will actually be used.</p> <H4>Logs</H4> <p>See <a href="https://caddyserver.com/docs/logging">how logging works</a> in Caddy. If your module needs logging, do not use <code>log.Print*()</code> from the Go standard library. In other words, <strong>do not use Go's global logger</strong>. Caddy uses high-performance, highly flexible, structured logging with <a href="https://github.com/uber-go/zap">zap</a>.</p> <p>To emit logs, get a logger in your module's Provision method:</p> <pre><code>func (g *Gizmo) Provision(ctx caddy.Context) error {
    g.logger = ctx.Logger() // g.logger is a *zap.Logger
}</code></pre> <p>Then you can emit structured, leveled logs using <code>g.logger</code>. See <a href="https://pkg.go.dev/go.uber.org/zap?tab=doc#Logger">zap's godoc</a> for details.</p> <H3>Validating</H3> <p>Modules which would like to validate their configuration may do so by satisfying the (optional) <a href="https://pkg.go.dev/github.com/caddyserver/caddy/v2?tab=doc#Validator"><code>caddy.Validator</code></a> interface:</p> <pre><code>// Validate validates that the module has a usable config.
func (g Gizmo) Validate() error {
    // TODO: validate the module's setup
    return nil
}</code></pre> <p>Validate should be a read-only function. It is run after the <code>Provision()</code> method.</p> <H3>Interface guards</H3> <p>Caddy module behavior is implicit because Go interfaces are satisfied implicitly. Simply adding the right methods to your module's type is all it takes to make or break your module's correctness. Thus, making a typo or getting the method signature wrong can lead to unexpected (lack of) behavior.</p> <p>Fortunately, there is an easy, no-overhead, compile-time check you can add to your code to ensure you've added the right methods. These are called interface guards:</p> <pre><code>var _ InterfaceName = (*YourType)(nil)</code></pre> <p>Replace <code>InterfaceName</code> with the interface you intend to satisfy, and <code>YourType</code> with the name of your module's type.</p> <p>For example, an HTTP handler such as the static file server might satisfy multiple interfaces:</p> <pre><code>// Interface guards
var (
    _ caddy.Provisioner           = (*FileServer)(nil)
    _ caddyhttp.MiddlewareHandler = (*FileServer)(nil)
)</code></pre> <p>This prevents the program from compiling if <code>*FileServer</code> does not satisfy those interfaces.</p> <p>Without interface guards, confusing bugs can slip in. For example, if your module must provision itself before being used but your <code>Provision()</code> method has a mistake (e.g. misspelled or wrong signature), provisioning will never happen, leading to head-scratching. Interface guards are super easy and can prevent that. They usually go at the bottom of the file.</p> <H2>Host Modules</H2> <p>A module becomes a host module when it loads its own guest modules. This is useful if a piece of the module's functionality can be implemented in different ways.</p> <p>A host module is almost always a struct. Normally, supporting a guest module requires two struct fields: one to hold its raw JSON, and another to hold its decoded value:</p> <pre><code>type Gizmo struct {
    GadgetRaw json.RawMessage `json:"gadget,omitempty" caddy:"namespace=foo.gizmo.gadgets inline_key=gadgeter"`

    Gadget Gadgeter `json:"-"`
}</code></pre> <p>The first field (<code>GadgetRaw</code> in this example) is where the raw, unprovisioned JSON form of the guest module can be found.</p> <p>The second field (<code>Gadget</code>) is where the final, provisioned value will eventually be stored. Since the second field is not user-facing, we exclude it from JSON with a struct tag. (You could also unexport it if it is not needed by other packages, and then no struct tag is needed.)</p> <H3>Caddy struct tags</H3> <p>The <code>caddy</code> struct tag on the raw module field helps Caddy to know the namespace and name (comprising the complete ID) of the module to load. It is also used for generating documentation.</p> <p>The struct tag has a very simple format: <code>key1=val1 key2=val2 ...</code></p> <p>For module fields, the struct tag will look like:</p> <pre><code>`caddy:"namespace=foo.bar inline_key=baz"`</code></pre> <p>The <code>namespace=</code> part is required. It defines the namespace in which to look for the module.</p> <p>The <code>inline_key=</code> part is only used if the module's name will be found <em>inline</em> with the module itself; this implies that the value is an object where one of the keys is the <em>inline key</em>, and its value is the name of the module. If omitted, then the field type must be a <a href="https://pkg.go.dev/github.com/caddyserver/caddy/v2?tab=doc#ModuleMap"><code>caddy.ModuleMap</code></a> or <code>[]caddy.ModuleMap</code>, where the map key is the module name.</p> <H3>Loading guest modules</H3> <p>To load a guest module, call <a href="https://pkg.go.dev/github.com/caddyserver/caddy/v2?tab=doc#Context.LoadModule"><code>ctx.LoadModule()</code></a> during the provision phase:</p> <pre><code>// Provision sets up g and loads its gadget.
func (g *Gizmo) Provision(ctx caddy.Context) error {
    if g.GadgetRaw != nil {
        val, err := ctx.LoadModule(g, "GadgetRaw")
        if err != nil {
            return fmt.Errorf("loading gadget module: %v", err)
        }
        g.Gadget = val.(Gadgeter)
    }
    return nil
}</code></pre> <p>Note that the <code>LoadModule()</code> call takes a pointer to the struct and the field name as a string. Weird, right? Why not just pass the struct field directly? It's because there are a few different ways to load modules depending on the layout of the config. This method signature allows Caddy to use reflection to figure out the best way to load the module and, most importantly, read its struct tags.</p> <p>If a guest module must explicitly be set by the user, you should return an error if the Raw field is nil or empty before trying to load it.</p> <p>Notice how the loaded module is type-asserted: <code>g.Gadget = val.(Gadgeter)</code> - this is because the returned <code>val</code> is a <code>interface{}</code> type which is not very useful. However, we expect that all modules in the declared namespace (<code>foo.gizmo.gadgets</code> from the struct tag in our example) implement the <code>Gadgeter</code> interface, so this type assertion is safe, and then we can use it!</p> <p>If your host module defines a new namespace, be sure to document both that namespace and its Go type(s) for developers <a href="https://caddyserver.com/docs/extending-caddy/namespaces">like we have done here</a>.</p> <H2>Module Documentation</H2> <p>Register the module to make a new Caddy module show up in the module documentation and be available in <a href="http://caddyserver.com/download">http://caddyserver.com/download</a>. The registration is available at <a href="http://caddyserver.com/account">http://caddyserver.com/account</a>. Create a new account if you don't have one already and click on "Register package".</p> <H2>Complete Example</H2> <p>Let's suppose we want to write an HTTP handler module. This will be a contrived middleware for demonstration purposes which prints the visitor's IP address to a stream on every HTTP request.</p> <p>We also want it to be configurable via the Caddyfile, because most people prefer to use the Caddyfile in non-automated situations. We do this by registering a Caddyfile handler directive, which is a kind of directive that can add a handler to the HTTP route. We also implement the <code>caddyfile.Unmarshaler</code> interface. By adding these few lines of code, this module can be configured with the Caddyfile! For example: <code>visitor_ip stdout</code>.</p> <p>Here is the code for such a module, with explanatory comments:</p> <pre><code>package visitorip

import (
    "fmt"
    "io"
    "net/http"
    "os"

    "github.com/caddyserver/caddy/v2"
    "github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
    "github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
    "github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
    caddy.RegisterModule(Middleware{})
    httpcaddyfile.RegisterHandlerDirective("visitor_ip", parseCaddyfile)
}

// Middleware implements an HTTP handler that writes the
// visitor's IP address to a file or stream.
type Middleware struct {
    // The file or stream to write to. Can be "stdout"
    // or "stderr".
    Output string `json:"output,omitempty"`

    w io.Writer
}

// CaddyModule returns the Caddy module information.
func (Middleware) CaddyModule() caddy.ModuleInfo {
    return caddy.ModuleInfo{
        ID:  "http.handlers.visitor_ip",
        New: func() caddy.Module { return new(Middleware) },
    }
}

// Provision implements caddy.Provisioner.
func (m *Middleware) Provision(ctx caddy.Context) error {
    switch m.Output {
    case "stdout":
        m.w = os.Stdout
    case "stderr":
        m.w = os.Stderr
    default:
        return fmt.Errorf("an output stream is required")
    }
    return nil
}

// Validate implements caddy.Validator.
func (m *Middleware) Validate() error {
    if m.w == nil {
        return fmt.Errorf("no writer")
    }
    return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
    m.w.Write([]byte(r.RemoteAddr))
    return next.ServeHTTP(w, r)
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (m *Middleware) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
    d.Next() // consume directive name

    // require an argument
    if !d.NextArg() {
        return d.ArgErr()
    }

    // store the argument
    m.Output = d.Val()
    return nil
}

// parseCaddyfile unmarshals tokens from h into a new Middleware.
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
    var m Middleware
    err := m.UnmarshalCaddyfile(h.Dispenser)
    return m, err
}

// Interface guards
var (
    _ caddy.Provisioner           = (*Middleware)(nil)
    _ caddy.Validator             = (*Middleware)(nil)
    _ caddyhttp.MiddlewareHandler = (*Middleware)(nil)
    _ caddyfile.Unmarshaler       = (*Middleware)(nil)
)</code></pre>    </article>
