package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules/express"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:18790", "listen address for manual server mode")
	script := flag.String("script", "examples/xgoja/20-express-hello-world/scripts/server.js", "JavaScript route script")
	smoke := flag.Bool("smoke", false, "run an in-process smoke test instead of listening")
	flag.Parse()

	if err := run(context.Background(), *listen, *script, *smoke); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, listen, script string, smoke bool) error {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, RejectRawRoutes: true})
	factory, err := engine.NewRuntimeFactoryBuilder().WithModules(express.NewRegistrar(host)).Build()
	if err != nil {
		return err
	}
	rt, err := factory.NewRuntime(engine.WithStartupContext(ctx), engine.WithLifetimeContext(ctx))
	if err != nil {
		return err
	}
	defer func() { _ = rt.Close(ctx) }()
	host.SetRuntime(rt.Owner)
	data, err := os.ReadFile(script)
	if err != nil {
		return err
	}
	if _, err := rt.Owner.Call(ctx, "load-hello-world-example", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(string(data))
		return nil, runErr
	}); err != nil {
		return err
	}
	if smoke {
		return runSmoke(host)
	}
	log.Printf("serving hello world demo on http://%s", listen)
	server := &http.Server{
		Addr:              listen,
		Handler:           host,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return serveWithShutdown(ctx, server)
}

func serveWithShutdown(ctx context.Context, server *http.Server) error {
	serveCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-serveCtx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}
	return <-errCh
}

func runSmoke(handler http.Handler) error {
	server := httptest.NewServer(handler)
	defer server.Close()
	client := server.Client()
	checks := []smokeCheck{
		{name: "root hello", path: "/", wantStatus: http.StatusOK, wantBody: "Hello, world!"},
		{name: "param hello", path: "/hello/goja", wantStatus: http.StatusOK, wantBody: `"message":"Hello, goja!"`},
		{name: "health", path: "/healthz", wantStatus: http.StatusOK, wantBody: `"ok":true`},
	}
	for _, check := range checks {
		if err := checkRequest(client, server.URL, check); err != nil {
			return err
		}
	}
	return nil
}

type smokeCheck struct {
	name       string
	path       string
	wantStatus int
	wantBody   string
}

func checkRequest(client *http.Client, baseURL string, check smokeCheck) error {
	req, err := http.NewRequest(http.MethodGet, baseURL+check.path, bytes.NewReader(nil))
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", check.name, err)
	}
	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		return fmt.Errorf("%s: read body: %w", check.name, readErr)
	}
	if resp.StatusCode != check.wantStatus {
		return fmt.Errorf("%s: status=%d body=%s", check.name, resp.StatusCode, string(body))
	}
	if check.wantBody != "" && !bytes.Contains(body, []byte(check.wantBody)) {
		return fmt.Errorf("%s: body=%s missing %s", check.name, string(body), check.wantBody)
	}
	fmt.Printf("ok %-16s %d\n", check.name, resp.StatusCode)
	return nil
}
