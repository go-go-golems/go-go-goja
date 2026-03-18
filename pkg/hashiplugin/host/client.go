package host

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/shared"
	"github.com/hashicorp/go-plugin"
)

// LoadedModule is a validated plugin client plus its manifest.
type LoadedModule struct {
	Path        string
	Manifest    *contract.ModuleManifest
	Module      contract.JSModule
	Client      *plugin.Client
	CallTimeout time.Duration
}

func (m *LoadedModule) RequireName() string {
	if m == nil || m.Manifest == nil {
		return ""
	}
	return m.Manifest.GetModuleName()
}

func (m *LoadedModule) Close() {
	if m == nil || m.Client == nil {
		return
	}
	m.Client.Kill()
}

func (m *LoadedModule) Invoke(ctx context.Context, req *contract.InvokeRequest) (*contract.InvokeResponse, error) {
	if m == nil || m.Module == nil {
		return nil, fmt.Errorf("plugin module is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok && m.CallTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, m.CallTimeout)
		defer cancel()
	}
	return m.Module.Invoke(ctx, req)
}

// LoadModules starts plugin subprocesses, dispenses the JS module service, and
// validates the returned manifests.
func LoadModules(cfg Config, paths []string) ([]*LoadedModule, error) {
	cfg = cfg.withDefaults()
	out := make([]*LoadedModule, 0, len(paths))
	for _, path := range paths {
		loaded, err := LoadModule(cfg, path)
		if err != nil {
			closeLoaded(out)
			return nil, err
		}
		out = append(out, loaded)
	}
	return out, nil
}

func LoadModule(cfg Config, path string) (*LoadedModule, error) {
	cfg = cfg.withDefaults()

	client := plugin.NewClient(&plugin.ClientConfig{
		Cmd:              exec.Command(path),
		HandshakeConfig:  shared.Handshake,
		VersionedPlugins: shared.VersionedClientPluginSets(),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		AutoMTLS:         cfg.AutoMTLS,
		StartTimeout:     cfg.StartTimeout,
		Logger:           cfg.Logger,
		SyncStdout:       io.Discard,
		SyncStderr:       io.Discard,
		Stderr:           io.Discard,
	})

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("start plugin %q: %w", path, err)
	}

	raw, err := rpcClient.Dispense(shared.ServiceName)
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("dispense plugin service from %q: %w", path, err)
	}

	mod, ok := raw.(contract.JSModule)
	if !ok {
		client.Kill()
		return nil, fmt.Errorf("plugin %q dispensed unexpected type %T", path, raw)
	}

	manifestCtx, cancel := context.WithTimeout(context.Background(), cfg.CallTimeout)
	defer cancel()
	manifest, err := mod.Manifest(manifestCtx)
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("read plugin manifest from %q: %w", path, err)
	}
	if err := ValidateManifest(cfg, manifest); err != nil {
		client.Kill()
		return nil, fmt.Errorf("validate plugin manifest from %q: %w", path, err)
	}

	return &LoadedModule{
		Path:        path,
		Manifest:    manifest,
		Module:      mod,
		Client:      client,
		CallTimeout: cfg.CallTimeout,
	}, nil
}

func closeLoaded(modules []*LoadedModule) {
	for _, mod := range modules {
		mod.Close()
	}
}
