---
Title: "07-xcaddy-github"
Source: external
LastUpdated: 2026-06-03
---

<article><H2 dir="auto">xcaddy - Custom Caddy Builder</H2> <p dir="auto">This command line tool and associated Go package makes it easy to make custom builds of the <a href="https://github.com/caddyserver/caddy">Caddy Web Server</a>.</p> <p dir="auto">It is used heavily by Caddy plugin developers as well as anyone who wishes to make custom <code>caddy</code> binaries (with or without plugins).</p> <p dir="auto">Stay updated, be aware of changes, and please submit feedback! Thanks!</p> <H2 dir="auto">Requirements</H2> <ul dir="auto"> <li><a href="https://golang.org/doc/install">Go installed</a></li> </ul> <H2 dir="auto">Install</H2> <p dir="auto">You can <a href="https://github.com/caddyserver/xcaddy/releases">download binaries</a> that are already compiled for your platform from the Release tab.</p> <p dir="auto">You may also build <code>xcaddy</code> from source:</p> <pre><code>go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest</code></pre> <p dir="auto">For Debian, Ubuntu, and Raspbian, an <code>xcaddy</code> package is available from our <a href="https://cloudsmith.io/~caddy/repos/xcaddy/packages/">Cloudsmith repo</a>:</p> <pre><code>sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/xcaddy/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-xcaddy-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/xcaddy/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-xcaddy.list
sudo apt update
sudo apt install xcaddy</code></pre> <H2 dir="auto">⚠️ Pro tip</H2> <p dir="auto">If you find yourself fighting xcaddy in relation to your custom or proprietary build or development process, <strong>it might be easier to just build Caddy manually!</strong></p> <p dir="auto">Caddy's <a href="https://github.com/caddyserver/caddy/blob/master/cmd/caddy/main.go">main.go file</a>, the main entry point to the application, has instructions in the comments explaining how to build Caddy essentially the same way xcaddy does it. But when you use the <code>go</code> command directly, you have more control over the whole thing and it may save you a lot of trouble.</p> <p dir="auto">The manual build procedure is very easy: just copy the main.go into a new folder, initialize a Go module, plug in your plugins (add an <code>import</code> for each one) and then run <code>go build</code>. Of course, you may wish to customize the go.mod file to your liking (specific dependency versions, replacements, etc).</p> <H2 dir="auto">Command usage</H2> <p dir="auto">The <code>xcaddy</code> command has two primary uses:</p> <ol dir="auto"> <li>Compile custom <code>caddy</code> binaries</li> <li>A replacement for <code>go run</code> while developing Caddy plugins</li> </ol> <p dir="auto">The <code>xcaddy</code> command will use the latest version of Caddy by default. You can customize this for all invocations by setting the <code>CADDY_VERSION</code> environment variable.</p> <p dir="auto">As usual with <code>go</code> command, the <code>xcaddy</code> command will pass the <code>GOOS</code>, <code>GOARCH</code>, and <code>GOARM</code> environment variables through for cross-compilation.</p> <p dir="auto">Note that <code>xcaddy</code> will ignore the <code>vendor/</code> folder with <code>-mod=readonly</code>.</p> <H3 dir="auto">Custom builds</H3> <p dir="auto">Syntax:</p> <div><pre><code>$ xcaddy build [&lt;caddy_version&gt;]
    [--output &lt;file&gt;]
    [--with &lt;module[@version][=replacement]&gt;...]
    [--replace &lt;module[@version]=replacement&gt;...]
    [--embed &lt;[alias]:path/to/dir&gt;...]
    [--pgo &lt;file&gt;] # EXPERIMENTAL</code></pre></div> <ul dir="auto"> <li> <p dir="auto"><code>&lt;caddy_version&gt;</code> is the core Caddy version to build; defaults to <code>CADDY_VERSION</code> env variable or latest.<br> This can be the keyword <code>latest</code>, which will use the latest stable tag, or any git ref such as:</p> <ul dir="auto"> <li>A tag like <code>v2.0.1</code></li> <li>A branch like <code>master</code></li> <li>A commit like <code>a58f240d3ecbb59285303746406cab50217f8d24</code></li> </ul> </li> <li> <p dir="auto"><code>--output</code> changes the output file.</p> </li> <li> <p dir="auto"><code>--with</code> can be used multiple times to add plugins by specifying the Go module name and optionally its version, similar to <code>go get</code>. Module name is required, but specific version and/or local replacement are optional.</p> </li> <li> <p dir="auto"><code>--replace</code> is like <code>--with</code>, but does not add a blank import to the code; it only writes a replace directive to <code>go.mod</code>, which is useful when developing on Caddy's dependencies (ones that are not Caddy modules). Try this if you got an error when using <code>--with</code>, like <code>cannot find module providing package</code>.</p> </li> <li> <p dir="auto"><code>--embed</code> can be used to embed the contents of a directory into the Caddy executable. <code>--embed</code> can be passed multiple times with separate source directories. The source directory can be prefixed with a custom alias and a colon <code>:</code> to write the embedded files into an aliased subdirectory, which is useful when combined with the <code>root</code> directive and sub-directive.</p> </li> <li> <p dir="auto"><code>--pgo</code> can be used to specify a file containing a profile to use for profile guided optimization. If a file named <code>default.pgo</code> is present in the current directory, it will automatically be used. This feature is new to xcaddy and is considered experimental.</p> </li> </ul> <H4 dir="auto">Examples</H4> <pre><code>$ xcaddy build \
    --with github.com/caddyserver/ntlm-transport

$ xcaddy build v2.0.1 \
    --with github.com/caddyserver/ntlm-transport@v0.1.1

$ xcaddy build master \
    --with github.com/caddyserver/ntlm-transport

$ xcaddy build a58f240d3ecbb59285303746406cab50217f8d24 \
    --with github.com/caddyserver/ntlm-transport

$ xcaddy build \
    --with github.com/caddyserver/ntlm-transport=../../my-fork

$ xcaddy build \
    --with github.com/caddyserver/ntlm-transport@v0.1.1=../../my-fork</code></pre> <p dir="auto">You can even replace Caddy core using the <code>--with</code> flag:</p> <div><pre><code>$ xcaddy build \
    --with github.com/caddyserver/caddy/v2=../../my-caddy-fork
    
$ xcaddy build \
    --with github.com/caddyserver/caddy/v2=github.com/my-user/caddy/v2@some-branch</code></pre></div> <p dir="auto">This allows you to hack on Caddy core (and optionally plug in extra modules at the same time!) with relative ease.</p> <hr> <p dir="auto">If <code>--embed</code> is used without an alias prefix, the contents of the source directory are written directly into the root directory of the embedded filesystem within the Caddy executable. The contents of multiple unaliased source directories will be merged together:</p> <div><pre><code>$ xcaddy build --embed ./my-files --embed ./my-other-files
$ cat Caddyfile
{
    # You must declare a custom filesystem using the `embedded` module.
    # The first argument to `filesystem` is an arbitrary identifier
    # that will also be passed to `fs` directives.
    filesystem my_embeds embedded
}

localhost {
    # This serves the files or directories that were
    # contained inside of ./my-files and ./my-other-files
    file_server {
        fs my_embeds
    }
}</code></pre></div> <p dir="auto">You may also prefix the source directory with a custom alias and colon separator to write the source directory's contents to a separate subdirectory within the <code>embedded</code> filesystem:</p> <div><pre><code>$ xcaddy build --embed foo:./sites/foo --embed bar:./sites/bar
$ cat Caddyfile
{
    filesystem my_embeds embedded
}

foo.localhost {
    # This serves the files or directories that were
    # contained inside of ./sites/foo
    root * /foo
    file_server {
        fs my_embeds
    }
}

bar.localhost {
    # This serves the files or directories that were
    # contained inside of ./sites/bar
    root * /bar
    file_server {
        fs my_embeds
    }
}</code></pre></div> <p dir="auto">This allows you to serve 2 sites from 2 different embedded directories, which are referenced by aliases, from a single Caddy executable.</p> <hr> <p dir="auto">If you need to work on Caddy's dependencies, you can use the <code>--replace</code> flag to replace it with a local copy of that dependency (or your fork on github etc if you need):</p> <div><pre><code>$ xcaddy build some-branch-on-caddy \
    --replace golang.org/x/net=../net</code></pre></div> <H3 dir="auto">For plugin development</H3> <p dir="auto">If you run <code>xcaddy</code> from within the folder of the Caddy plugin you're working on <em>without the <code>build</code> subcommand</em>, it will build Caddy with your current module and run it, as if you manually plugged it in and invoked <code>go run</code>.</p> <p dir="auto">The binary will be built and run from the current directory, then cleaned up.</p> <p dir="auto">The current working directory must be inside an initialized Go module.</p> <p dir="auto">Syntax:</p> <div><pre><code>$ xcaddy &lt;args...&gt;</code></pre></div> <ul dir="auto"> <li><code>&lt;args...&gt;</code> are passed through to the <code>caddy</code> command.</li> </ul> <p dir="auto">For example:</p> <pre><code>$ xcaddy list-modules
$ xcaddy run
$ xcaddy run --config caddy.json</code></pre> <p dir="auto">The race detector can be enabled by setting <code>XCADDY_RACE_DETECTOR=1</code>. The DWARF debug info can be enabled by setting <code>XCADDY_DEBUG=1</code>.</p> <H3 dir="auto">Getting xcaddy's version</H3> <div><pre><code>$ xcaddy version</code></pre></div> <H2 dir="auto">Library usage</H2> <pre><code>builder := xcaddy.Builder{
    CaddyVersion: "v2.0.0",
    Plugins: []xcaddy.Dependency{
        {
            ModulePath: "github.com/caddyserver/ntlm-transport",
            Version:    "v0.1.1",
        },
    },
}
err := builder.Build(context.Background(), "./caddy")</code></pre> <p dir="auto">Versions can be anything compatible with <code>go get</code>.</p> <H2 dir="auto">Environment variables</H2> <p dir="auto">Because the subcommands and flags are constrained to benefit rapid plugin prototyping, xcaddy does read some environment variables to take cues for its behavior and/or configuration when there is no room for flags.</p> <ul dir="auto"> <li><code>CADDY_VERSION</code> sets the version of Caddy to build.</li> <li><code>XCADDY_RACE_DETECTOR=1</code> enables the Go race detector in the build.</li> <li><code>XCADDY_DEBUG=1</code> enables the DWARF debug information in the build.</li> <li><code>XCADDY_SETCAP=1</code> will run <code>sudo setcap cap_net_bind_service=+ep</code> on the resulting binary. By default, the <code>sudo</code> command will be used if it is found; set <code>XCADDY_SUDO=0</code> to avoid using <code>sudo</code> if necessary.</li> <li><code>XCADDY_SKIP_BUILD=1</code> causes xcaddy to not compile the program, it is used in conjunction with build tools such as <a href="https://goreleaser.com/">GoReleaser</a>. Implies <code>XCADDY_SKIP_CLEANUP=1</code>.</li> <li><code>XCADDY_SKIP_CLEANUP=1</code> causes xcaddy to leave build artifacts on disk after exiting.</li> <li><code>XCADDY_WHICH_GO</code> sets the go command to use when for example more then 1 version of go is installed.</li> <li><code>XCADDY_GO_BUILD_FLAGS</code> overrides default build arguments. Supports Unix-style shell quoting, for example: XCADDY_GO_BUILD_FLAGS="-ldflags '-w -s'". The provided flags are applied to <code>go</code> commands: build, clean, get, install, list, run, and test</li> <li><code>XCADDY_GO_MOD_FLAGS</code> overrides default <code>go mod</code> arguments. Supports Unix-style shell quoting.</li> </ul>   </article>
