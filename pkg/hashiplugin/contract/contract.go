package contract

import "context"

// JSModule is the host/plugin contract used on both sides of the go-plugin
// transport boundary.
type JSModule interface {
	Manifest(ctx context.Context) (*ModuleManifest, error)
	Invoke(ctx context.Context, req *InvokeRequest) (*InvokeResponse, error)
}
