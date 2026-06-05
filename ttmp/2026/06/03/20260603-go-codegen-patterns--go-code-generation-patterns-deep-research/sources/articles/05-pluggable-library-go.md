---
Title: "05-pluggable-library-go"
Source: external
LastUpdated: 2026-06-03
---

<div><p>While exploring <a href="https://www.envoyproxy.io/docs/envoy/latest/start/sandboxes/golang-http">Envoy Proxy</a>, I got intrigued by how users can write custom code as plugins and load those implementations at runtime. This curiosity led me down a rabbit hole of research, where I stumbled upon the <code>buildmode=plugin</code> option in <a href="https://pkg.go.dev/cmd/go#hdr-Build_modes">Go’s official documentation</a>. <a href="https://pkg.go.dev/plugin#Symbol">The documentation was pretty straightforward</a>, so I decided to try it out, and now I want to share what I’ve learned.</p><H2>What is go buildmode=plugin?</H2><p>The <code>go buildmode=plugin</code> option allows you to compile Go code into a shared object file. This file can be loaded by another Go program at runtime. It’s useful when you want to add new features to your application without rebuilding it. Instead, you can load new features as plugins.</p><p>A plugin in Go is a package compiled into a shared object (.so) file. This file can be loaded using the <a href="https://pkg.go.dev/plugin">plugin package in Go</a>, which lets you open the plugin, look up symbols (like functions or variables), and use them.</p><H2>Hands-on Example</H2> <p>To make this a bit more concrete, let’s dive into an example where this feature really shines.</p><p>I’ve put together a simple demo backend project that exposes an API for calculating the n-th Fibonacci sequence. You can find the full code <a href="https://github.com/josestg/yt-go-plugin">here</a>. For demonstration purposes, I’ve intentionally used a slow Fibonacci implementation. Given that the computation is slow, I added a caching layer to store the results, so if the same n-th Fibonacci number is requested again, it doesn’t need to be recalculated—we just return the cached result.</p><p>The API is exposed via a <code>GET /fib/{n}</code> endpoint, where <code>n</code> is the Fibonacci number you want to calculate. Here’s a look at how the API is implemented:</p><pre><code class="language-go" data-lang="go">// Fibonacci calculates the nth Fibonacci number.
// This algorithm is not optimized and is used for demonstration purposes.
func Fibonacci(n int64) int64 {
    if n &lt;= 1 {
        return n
    }
    return Fibonacci(n-1) + Fibonacci(n-2)
}

// NewHandler returns an HTTP handler that calculates the nth Fibonacci number.
func NewHandler(l *slog.Logger, c cache.Cache, exp time.Duration) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        started := time.Now()
        defer func() {
            l.Info("request completed", "duration", time.Since(started).String())
        }()

        param := r.PathValue("n")
        n, err := strconv.ParseInt(param, 10, 64)
        if err != nil {
            l.Error("cannot parse path value", "param", param, "error", err)
            sendJSON(l, w, map[string]any{"error": "invalid value"}, http.StatusBadRequest)
            return
        }

        ctx := r.Context()

        result := make(chan int64)
        go func() {
            cached, err := c.Get(ctx, param)
            if err != nil {
                l.Debug("cache miss; calculating the fib(n)", "n", n, "cache_error", err)
                v := Fibonacci(n)
                l.Debug("fib(n) calculated", "n", n, "result", v)
                if err := c.Set(ctx, param, strconv.FormatInt(v, 10), exp); err != nil {
                    l.Error("cannot set cache", "error", err)
                }
                result &lt;- v
                return
            }

            l.Debug("cache hit; returning the cached value", "n", n, "value", cached)
            v, _ := strconv.ParseInt(cached, 10, 64)
            result &lt;- v
        }()

        select {
        case v := &lt;-result:
            sendJSON(l, w, map[string]any{"result": v}, http.StatusOK)
        case &lt;-ctx.Done():
            l.Info("request cancelled")
        }
    }
}</code></pre> <blockquote><p>code: <a href="https://github.com/josestg/yt-go-plugin/blob/8661a4569c6264e54cac0ad6a912011a1a777f44/internal/fibonacci/fibonacci.go#L13-L66">https://github.com/josestg/yt-go-plugin/blob/8661a4569c6264e54cac0ad6a912011a1a777f44/internal/fibonacci/fibonacci.go#L13-L66</a></p></blockquote> <p>The code does the following:</p><ol><li>The <code>NewHandler</code> function creates a new <code>http.Handler</code>. It takes a logger, cache, and expiration duration as dependencies. The <code>cache.Cache</code> is an interface, which we’ll define shortly.</li><li>The returned <code>http.Handler</code> parses the <code>n</code> value from the path parameters. If there’s an error, it sends an error response. Otherwise, it checks if the n-th Fibonacci number is already in the cache. If it’s not, the handler calculates the number and stores it in the cache for future requests.</li><li>A goroutine handles the Fibonacci calculation and caching in a separate process, while the select statement waits for either the calculation to complete or the client to cancel the request. This ensures that if the client cancels the request, we don’t waste resources waiting for the calculation to finish.</li></ol><p>Now, <strong>we want to make the cache implementation selectable at runtime, when the application starts</strong>. A straightforward approach would be to create multiple implementations within the same codebase and use a config to select the desired implementation. However, the downside is that the unselected implementations would still be part of the compiled binary, which increases the binary size. While build tags could be a solution, we’ll save that for another article. For now, we want the implementation to be chosen at runtime, not at build time. This is where <code>buildmode=plugin</code> really shines.</p><H3>Ensuring the Application Works Without a Plugin</H3> <p>Since we’ve defined <code>cache.Cache</code> as an interface, we can create implementations of this interface anywhere—even in a different repository. But first, let’s take a look at the <code>Cache</code> interface:</p><pre><code class="language-go" data-lang="go">// Cache defines the interface for a cache implementation.
type Cache interface {
    // Set stores a key-value pair in the cache with a specified expiration time.
    Set(ctx context.Context, key, val string, exp time.Duration) error

    // Get retrieves a value from the cache by its key.
    // Returns ErrNotFound if the key is not found.
    // Returns ErrExpired if the key has expired.
    Get(ctx context.Context, key string) (string, error)
}</code></pre> <blockquote><p>code: <a href="https://github.com/josestg/yt-go-plugin/blob/8661a4569c6264e54cac0ad6a912011a1a777f44/cache/cache.go#L34-L43">https://github.com/josestg/yt-go-plugin/blob/8661a4569c6264e54cac0ad6a912011a1a777f44/cache/cache.go#L34-L43</a></p></blockquote> <p>Since <code>NewHandler</code> requires a <code>cache.Cache</code> implementation as a dependency, it’s a good idea to have a default implementation to ensure the code doesn’t break. So, let’s create a no-op (no-operation) implementation that does nothing.</p><pre><code class="language-go" data-lang="go">// nopCache is a no-operation cache implementation.
type nopCache int

// NopCache a singleton cache instance, which does nothing.
const NopCache nopCache = 0

// Ensure that NopCache implements the Cache interface.
var _ Cache = NopCache

// Set is a no-op and always returns nil.
func (nopCache) Set(context.Context, string, string, time.Duration) error { return nil }

// Get always returns ErrNotFound, indicating that the key does not exist in the cache.
func (nopCache) Get(context.Context, string) (string, error) { return "", ErrNotFound }</code></pre> <blockquote><p>code: <a href="https://github.com/josestg/yt-go-plugin/blob/8661a4569c6264e54cac0ad6a912011a1a777f44/cache/cache.go#L48-L61">https://github.com/josestg/yt-go-plugin/blob/8661a4569c6264e54cac0ad6a912011a1a777f44/cache/cache.go#L48-L61</a></p></blockquote> <p>This <code>NopCache</code> implements the <code>cache.Cache</code> interface but doesn’t actually do anything. It’s just there to make sure the handler works properly.</p><p>If we run the code without any custom <code>cache.Cache</code> implementation, the API will work fine, but the results won’t be cached—meaning each call will recalculate the Fibonacci number. Here’s what the logs look like when using <code>NopCache</code> with <code>n=45</code>:</p><pre><code class="language-bash" data-lang="bash">./bin/demo -port=8080 -log-level=debug

time=2024-08-22T17:39:06.853+07:00 level=INFO msg="application started"
time=2024-08-22T17:39:06.854+07:00 level=DEBUG msg="using configuration" config="{Port:8080 LogLevel:DEBUG CacheExpiration:15s CachePluginPath: CachePluginFactoryName:Factory}"
time=2024-08-22T17:39:06.854+07:00 level=INFO msg="no cache plugin configured; using nop cache"
time=2024-08-22T17:39:06.854+07:00 level=INFO msg=listening addr=:8080

time=2024-08-22T17:39:19.465+07:00 level=DEBUG msg="cache miss; calculating the fib(n)" n=45 cache_error="cache: key not found"
time=2024-08-22T17:39:23.246+07:00 level=DEBUG msg="fib(n) calculated" n=45 result=1134903170
time=2024-08-22T17:39:23.246+07:00 level=INFO msg="request completed" duration=3.781674792s

time=2024-08-22T17:39:26.409+07:00 level=DEBUG msg="cache miss; calculating the fib(n)" n=45 cache_error="cache: key not found"
time=2024-08-22T17:39:30.222+07:00 level=DEBUG msg="fib(n) calculated" n=45 result=1134903170
time=2024-08-22T17:39:30.222+07:00 level=INFO msg="request completed" duration=3.813693s</code></pre> <p>As expected, both calls take around 3 seconds since there’s no caching.</p><H3>Implementing the Plugin</H3> <p>Since the library we want to make pluggable is <code>cache.Cache</code>, we need to implement that interface. <strong>You can implement this interface anywhere—even in a separate repository</strong>. For this example, I’ve created two implementations: one using <a href="https://github.com/josestg/yt-go-plugin-memcache">in-memory cache</a> and another using <a href="https://github.com/josestg/yt-go-plugin-rediscache">Redis</a> both in separate repository.</p><p><strong>In-Memory Cache Plugin</strong></p> <pre><code class="language-go" data-lang="go">package main

import (
    "context"
    "log/slog"
    "sync"
    "time"

    "github.com/josestg/yt-go-plugin/cache"
)

// Value represents a cache entry.
type Value struct {
    Data  string
    ExpAt time.Time
}

// Memcache is a simple in-memory cache.
type Memcache struct {
    mu    sync.RWMutex
    log   *slog.Logger
    store map[string]Value
}

// Factory is the symbol the plugin loader will try to load. It must implement the cache.Factory signature.
var Factory cache.Factory = New

// New creates a new Memcache instance.
func New(log *slog.Logger) (cache.Cache, error) {
    log.Info("[plugin/memcache] loaded")
    c := &amp;Memcache{
        mu:    sync.RWMutex{},
        log:   log,
        store: make(map[string]Value),
    }
    return c, nil
}

func (m *Memcache) Set(ctx context.Context, key, val string, exp time.Duration) error {
    m.log.InfoContext(ctx, "[plugin/memcache] set", "key", key, "val", val, "exp", exp)
    m.mu.Lock()
    m.log.DebugContext(ctx, "[plugin/memcache] lock acquired")
    defer func() {
        m.mu.Unlock()
        m.log.DebugContext(ctx, "[plugin/memcache] lock released")
    }()

    m.store[key] = Value{
        Data:  val,
        ExpAt: time.Now().Add(exp),
    }

    return nil
}

func (m *Memcache) Get(ctx context.Context, key string) (string, error) {
    m.log.InfoContext(ctx, "[plugin/memcache] get", "key", key)
    m.mu.RLock()
    v, ok := m.store[key]
    m.mu.RUnlock()
    if !ok {
        return "", cache.ErrNotFound
    }

    if time.Now().After(v.ExpAt) {
        m.log.InfoContext(ctx, "[plugin/memcache] key expired", "key", key, "val", v)
        m.mu.Lock()
        delete(m.store, key)
        m.mu.Unlock()
        return "", cache.ErrExpired
    }

    m.log.InfoContext(ctx, "[plugin/memcache] key found", "key", key, "val", v)
    return v.Data, nil
}</code></pre> <blockquote><p>code: <a href="https://github.com/josestg/yt-go-plugin-memcache/blob/29b76a5bd23308d41b99dc7bc06a67efa8d417a8/memcache.go">https://github.com/josestg/yt-go-plugin-memcache/blob/29b76a5bd23308d41b99dc7bc06a67efa8d417a8/memcache.go</a></p></blockquote> <p><strong>Redis Cache Plugin</strong></p> <pre><code class="language-go" data-lang="go">package main

import (
    "cmp"
    "context"
    "errors"
    "fmt"
    "log/slog"
    "os"
    "strconv"
    "time"

    "github.com/josestg/yt-go-plugin/cache"
    "github.com/redis/go-redis/v9"
)

// RedisCache is a cache implementation that uses Redis.
type RedisCache struct {
    log    *slog.Logger
    client *redis.Client
}

// Factory is the symbol the plugin loader will try to load. It must implement the cache.Factory signature.
var Factory cache.Factory = New

// New creates a new RedisCache instance.
func New(log *slog.Logger) (cache.Cache, error) {
    log.Info("[plugin/rediscache] loaded")
    db, err := strconv.Atoi(cmp.Or(os.Getenv("REDIS_DB"), "0"))
    if err != nil {
        return nil, fmt.Errorf("parse redis db: %w", err)
    }

    c := &amp;RedisCache{
        log: log,
        client: redis.NewClient(&amp;redis.Options{
            Addr:     cmp.Or(os.Getenv("REDIS_ADDR"), "localhost:6379"),
            Password: cmp.Or(os.Getenv("REDIS_PASSWORD"), ""),
            DB:       db,
        }),
    }

    return c, nil
}

func (r *RedisCache) Set(ctx context.Context, key, val string, exp time.Duration) error {
    r.log.InfoContext(ctx, "[plugin/rediscache] set", "key", key, "val", val, "exp", exp)
    return r.client.Set(ctx, key, val, exp).Err()
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
    r.log.InfoContext(ctx, "[plugin/rediscache] get", "key", key)
    res, err := r.client.Get(ctx, key).Result()
    if errors.Is(err, redis.Nil) {
        r.log.InfoContext(ctx, "[plugin/rediscache] key not found", "key", key)
        return "", cache.ErrNotFound
    }
    r.log.InfoContext(ctx, "[plugin/rediscache] key found", "key", key, "val", res)
    return res, err
}</code></pre> <blockquote><p>code: <a href="https://github.com/josestg/yt-go-plugin-rediscache/blob/01154faa9fcf96323fa276d6c328d42ae0bce81b/rediscache.go">https://github.com/josestg/yt-go-plugin-rediscache/blob/01154faa9fcf96323fa276d6c328d42ae0bce81b/rediscache.go</a></p></blockquote> <p>As you can see, both plugins implement the <code>cache.Cache</code> interface. Here are a couple of important things to note:</p><ol><li>Both plugins are implemented in the <code>main</code> package. This is mandatory because when we build the code as a plugin, Go requires at least one <code>main</code> package. That said, it doesn’t mean you have to write all your code in a single file. You can organize it as a typical Go project with multiple files and packages. I’ve kept it in a single file here for simplicity.</li><li>Both plugins have <code>var Factory cache.Factory = New</code>. While not mandatory, this is a good practice. We create a type that we expect every plugin to follow as a signature for the implementation constructor. Both plugins ensure that their <code>New</code> function (the actual constructor) is of type <code>cache.Factory</code>. This is important when we look up the constructor later.</li></ol><p>Building the plugin is straightforward—just add the <code>-buildmode=plugin</code> flag.</p><pre><code class="language-bash" data-lang="bash"># build the in memory cache plugin
go build -buildmode=plugin -o memcache.so memcache.go

# build the redis cache plugin
go build -buildmode=plugin -o rediscache.so rediscache.go</code></pre> <p>Running these commands will produce <code>memcache.so</code> and <code>rediscache.so</code>, which are shared object binaries that can be loaded at runtime by the <code>bin/demo</code> binary.</p><H3>Implementing the Plugin Loader</H3> <p>The plugin loader is pretty simple. We can use the standard <code>plugin</code> library in Go, which provides two functions, both of which are self-explanatory:</p><ol><li><a href="https://pkg.go.dev/plugin#Open">Open</a>: opens the shared object binary file.</li><li><a href="https://pkg.go.dev/plugin#Plugin.Lookup">Lookup</a>: searches for an exported symbol in the shared object. The symbol can be a function or a variable. But here’s the catch: <strong>all symbols returned by <code>Lookup</code> have a type pointer to <code>any</code></strong>, even if the symbol itself isn’t declared as a pointer type. Let’s see this in action.</li></ol><p>Here’s the code to load the plugin:</p><pre><code class="language-go" data-lang="go">// loadCachePlugin loads a cache implementation from a shared object (.so) file at the specified path.
// It calls the constructor function by name, passing the necessary dependencies, and returns the initialized cache.
// If path is empty, it returns the NopCache implementation.
func loadCachePlugin(log *slog.Logger, path, name string) (cache.Cache, error) {
    if path == "" {
        log.Info("no cache plugin configured; using nop cache")
        return cache.NopCache, nil
    }

    plug, err := plugin.Open(path)
    if err != nil {
        return nil, fmt.Errorf("open plugin %q: %w", path, err)
    }

    sym, err := plug.Lookup(name)
    if err != nil {
        return nil, fmt.Errorf("lookup symbol New: %w", err)
    }

    factoryPtr, ok := sym.(*cache.Factory)
    if !ok {
        return nil, fmt.Errorf("unexpected type %T; want %T", sym, factoryPtr)
    }

    factory := *factoryPtr
    return factory(log)
}</code></pre> <blockquote><p>code: <a href="https://github.com/josestg/yt-go-plugin/blob/8661a4569c6264e54cac0ad6a912011a1a777f44/main.go#L61-L84">https://github.com/josestg/yt-go-plugin/blob/8661a4569c6264e54cac0ad6a912011a1a777f44/main.go#L61-L84</a></p></blockquote> <p>Take a closer look at this line: <code>factoryPtr, ok := sym.(*cache.Factory)</code>. We’re looking for the symbol <code>plug.Lookup("Factory")</code>, and as we’ve seen, each implementation has <code>var Factory cache.Factory = New</code>, not <code>var Factory *cache.Factory = New</code>.</p><p>Here’s how <code>cache.Factory</code> is defined:</p><pre><code class="language-go" data-lang="go">// Factory defines the function signature for creating a cache implementation.
type Factory func(log *slog.Logger) (Cache, error)</code></pre> <blockquote><p>code: <a href="https://github.com/josestg/yt-go-plugin/blob/8661a4569c6264e54cac0ad6a912011a1a777f44/cache/cache.go#L45-L46">https://github.com/josestg/yt-go-plugin/blob/8661a4569c6264e54cac0ad6a912011a1a777f44/cache/cache.go#L45-L46</a></p></blockquote> <p>So, we need to dereference <code>factoryPtr</code> before calling it with the given logger.</p><H2>Demo</H2> <p>If we look at the <code>bin/demo</code> package’s main function, we can pass the plugin path and factory name as command-line arguments:</p><pre><code class="language-go" data-lang="go">var cfg conf
flag.IntVar(&amp;cfg.Port, "port", 8080, "port to listen on")
flag.TextVar(&amp;cfg.LogLevel, "log-level", slog.LevelInfo, "log level")
flag.StringVar(&amp;cfg.CachePluginPath, "cache-plugin-path", "", "path to the cache plugin")
flag.StringVar(&amp;cfg.CachePluginFactoryName, "cache-plugin-factory-name", "Factory", "name of the factory function in the cache plugin")
flag.DurationVar(&amp;cfg.CacheExpiration, "cache-expiration", 15*time.Second, "duration that a cache entry will be valid for")
flag.Parse()</code></pre> <blockquote><p>code: <a href="https://github.com/josestg/yt-go-plugin/blob/8661a4569c6264e54cac0ad6a912011a1a777f44/main.go#L25-L31">https://github.com/josestg/yt-go-plugin/blob/8661a4569c6264e54cac0ad6a912011a1a777f44/main.go#L25-L31</a></p></blockquote> <p>Or you can check out the details in the help menu:</p><pre><code class="language-bash" data-lang="bash">./bin/demo -h

Usage of ./bin/demo:
  -cache-expiration duration
        duration that a cache entry will be valid for (default 15s)
  -cache-plugin-factory-name string
        name of the factory function in the cache plugin (default "Factory")
  -cache-plugin-path string
        path to the cache plugin
  -log-level value
        log level (default INFO)
  -port int
        port to listen on (default 8080)</code></pre> <H3>Using the In-Memory Cache Implementation</H3><pre><code class="language-bash" data-lang="bash">./bin/demo -port=8080 -log-level=debug -cache-plugin-path=./memcache.so -cache-plugin-factory-name=Factory</code></pre> <p>Logs after calling <code>http://localhost:8080/fib/45</code> twice:</p><pre><code class="language-bash" data-lang="bash">time=2024-08-22T18:31:08.372+07:00 level=INFO msg="application started"
time=2024-08-22T18:31:08.372+07:00 level=DEBUG msg="using configuration" config="{Port:8080 LogLevel:DEBUG CacheExpiration:15s CachePluginPath:./memcache.so CachePluginFactoryName:Factory}"
time=2024-08-22T18:31:08.376+07:00 level=INFO msg="[plugin/memcache] loaded"
time=2024-08-22T18:31:08.376+07:00 level=INFO msg=listening addr=:8080

time=2024-08-22T18:31:16.850+07:00 level=INFO msg="[plugin/memcache] get" key=45
time=2024-08-22T18:31:16.850+07:00 level=DEBUG msg="cache miss; calculating the fib(n)" n=45 cache_error="cache: key not found"
time=2024-08-22T18:31:20.752+07:00 level=DEBUG msg="fib(n) calculated" n=45 result=1134903170
time=2024-08-22T18:31:20.752+07:00 level=INFO msg="[plugin/memcache] set" key=45 val=1134903170 exp=15s
time=2024-08-22T18:31:20.752+07:00 level=DEBUG msg="[plugin/memcache] lock acquired"
time=2024-08-22T18:31:20.752+07:00 level=DEBUG msg="[plugin/memcache] lock released"
time=2024-08-22T18:31:20.753+07:00 level=INFO msg="request completed" duration=3.903607875s

time=2024-08-22T18:31:24.781+07:00 level=INFO msg="[plugin/memcache] get" key=45
time=2024-08-22T18:31:24.783+07:00 level=INFO msg="[plugin/memcache] key found" key=45 val="{Data:1134903170 ExpAt:2024-08-22 18:31:35.752647 +0700 WIB m=+27.380493292}"
time=2024-08-22T18:31:24.783+07:00 level=DEBUG msg="cache hit; returning the cached value" n=45 value=1134903170
time=2024-08-22T18:31:24.783+07:00 level=INFO msg="request completed" duration=1.825042ms</code></pre> <H3>Using the Redis Cache Implementation</H3><pre><code class="language-bash" data-lang="bash">./bin/demo -port=8080 -log-level=debug -cache-plugin-path=./rediscache.so -cache-plugin-factory-name=Factory</code></pre> <p>Logs after calling <code>http://localhost:8080/fib/45</code> twice:</p><pre><code class="language-bash" data-lang="bash">time=2024-08-22T18:33:49.920+07:00 level=INFO msg="application started"
time=2024-08-22T18:33:49.920+07:00 level=DEBUG msg="using configuration" config="{Port:8080 LogLevel:DEBUG CacheExpiration:15s CachePluginPath:./rediscache.so CachePluginFactoryName:Factory}"
time=2024-08-22T18:33:49.937+07:00 level=INFO msg="[plugin/rediscache] loaded"
time=2024-08-22T18:33:49.937+07:00 level=INFO msg=listening addr=:8080

time=2024-08-22T18:34:01.143+07:00 level=INFO msg="[plugin/rediscache] get" key=45
time=2024-08-22T18:34:01.150+07:00 level=INFO msg="[plugin/rediscache] key not found" key=45
time=2024-08-22T18:34:01.150+07:00 level=DEBUG msg="cache miss; calculating the fib(n)" n=45 cache_error="cache: key not found"
time=2024-08-22T18:34:04.931+07:00 level=DEBUG msg="fib(n) calculated" n=45 result=1134903170
time=2024-08-22T18:34:04.931+07:00 level=INFO msg="[plugin/rediscache] set" key=45 val=1134903170 exp=15s
time=2024-08-22T18:34:04.934+07:00 level=INFO msg="request completed" duration=3.791582708s

time=2024-08-22T18:34:07.932+07:00 level=INFO msg="[plugin/rediscache] get" key=45
time=2024-08-22T18:34:07.936+07:00 level=INFO msg="[plugin/rediscache] key found" key=45 val=1134903170
time=2024-08-22T18:34:07.936+07:00 level=DEBUG msg="cache hit; returning the cached value" n=45 value=1134903170
time=2024-08-22T18:34:07.936+07:00 level=INFO msg="request completed" duration=4.403083ms</code></pre> <H2>Conclusion</H2> <p>The <code>buildmode=plugin</code> feature in Go is a powerful tool for enhancing applications, such as adding custom caching solutions in Envoy Proxy. It allows you to build and use plugins, enabling you to load and execute custom code at runtime without altering the main application. This not only helps in reducing the binary size but also speeds up the build process. Since plugins can be composed and updated independently, you only need to rebuild the main application if there are changes, avoiding the need to rebuild unchanged plugins.</p><p>However, it’s important to consider some drawbacks. Plugin loading can introduce runtime overhead, and the plugin system has certain limitations compared to statically linked code. For instance, there may be issues with cross-platform compatibility and debugging complexity. You should carefully evaluate these aspects based on your specific needs. For more information and detailed warnings about using plugins, refer to the <a href="https://pkg.go.dev/plugin#hdr-Warnings">Go official documentation on plugins</a>.</p></div>
