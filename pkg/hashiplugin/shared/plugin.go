package shared

import (
	"context"
	"fmt"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	ProtocolVersion  = 1
	ServiceName      = "js_module"
	MagicCookieKey   = "GO_GO_GOJA_PLUGIN"
	MagicCookieValue = "js-module"
)

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  ProtocolVersion,
	MagicCookieKey:   MagicCookieKey,
	MagicCookieValue: MagicCookieValue,
}

// JSModulePlugin exposes the JS module service over go-plugin gRPC transport.
type JSModulePlugin struct {
	plugin.NetRPCUnsupportedPlugin
	Impl contract.JSModule
}

var _ plugin.GRPCPlugin = (*JSModulePlugin)(nil)

func (p *JSModulePlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	if p == nil || p.Impl == nil {
		return fmt.Errorf("hashiplugin shared: nil JS module implementation")
	}
	contract.RegisterJSModuleServiceServer(s, &grpcServer{impl: p.Impl})
	return nil
}

func (p *JSModulePlugin) GRPCClient(ctx context.Context, _ *plugin.GRPCBroker, conn *grpc.ClientConn) (interface{}, error) {
	return &grpcClient{
		ctx:    ctx,
		client: contract.NewJSModuleServiceClient(conn),
	}, nil
}

// ClientPluginSet returns the plugin set used by hosts when dispensing JS modules.
func ClientPluginSet() plugin.PluginSet {
	return plugin.PluginSet{
		ServiceName: &JSModulePlugin{},
	}
}

// ServerPluginSet returns the plugin set used by plugin subprocesses.
func ServerPluginSet(impl contract.JSModule) plugin.PluginSet {
	return plugin.PluginSet{
		ServiceName: &JSModulePlugin{Impl: impl},
	}
}

// VersionedClientPluginSets returns the versioned plugin mapping for hosts.
func VersionedClientPluginSets() map[int]plugin.PluginSet {
	return map[int]plugin.PluginSet{
		ProtocolVersion: ClientPluginSet(),
	}
}

// VersionedServerPluginSets returns the versioned plugin mapping for plugins.
func VersionedServerPluginSets(impl contract.JSModule) map[int]plugin.PluginSet {
	return map[int]plugin.PluginSet{
		ProtocolVersion: ServerPluginSet(impl),
	}
}

type grpcServer struct {
	contract.UnimplementedJSModuleServiceServer
	impl contract.JSModule
}

func (s *grpcServer) GetManifest(ctx context.Context, _ *emptypb.Empty) (*contract.ModuleManifest, error) {
	return s.impl.Manifest(ctx)
}

func (s *grpcServer) Invoke(ctx context.Context, req *contract.InvokeRequest) (*contract.InvokeResponse, error) {
	if req == nil {
		req = &contract.InvokeRequest{}
	}
	return s.impl.Invoke(ctx, req)
}

type grpcClient struct {
	ctx    context.Context
	client contract.JSModuleServiceClient
}

func (c *grpcClient) Manifest(ctx context.Context) (*contract.ModuleManifest, error) {
	return c.client.GetManifest(normalizeContext(ctx, c.ctx), &emptypb.Empty{})
}

func (c *grpcClient) Invoke(ctx context.Context, req *contract.InvokeRequest) (*contract.InvokeResponse, error) {
	if req == nil {
		req = &contract.InvokeRequest{}
	}
	return c.client.Invoke(normalizeContext(ctx, c.ctx), req)
}

func normalizeContext(ctx context.Context, fallback context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	if fallback != nil {
		return fallback
	}
	return context.Background()
}
