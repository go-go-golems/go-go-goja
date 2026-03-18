package sdk

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"google.golang.org/protobuf/types/known/structpb"
)

type dispatchKey struct {
	exportName string
	methodName string
}

func buildDispatchTable(def *moduleDefinition) (map[dispatchKey]Handler, error) {
	dispatch := make(map[dispatchKey]Handler, len(def.exports))
	for _, exp := range def.exports {
		switch exp.kind {
		case contract.ExportKind_EXPORT_KIND_UNSPECIFIED:
			return nil, fmt.Errorf("sdk export %q has unspecified kind", exp.name)
		case contract.ExportKind_EXPORT_KIND_FUNCTION:
			dispatch[dispatchKey{exportName: exp.name}] = exp.handler
		case contract.ExportKind_EXPORT_KIND_OBJECT:
			for _, method := range exp.methods {
				dispatch[dispatchKey{exportName: exp.name, methodName: method.name}] = method.handler
			}
		default:
			return nil, fmt.Errorf("sdk export %q has unsupported kind %q", exp.name, exp.kind.String())
		}
	}
	return dispatch, nil
}

func (m *Module) Invoke(ctx context.Context, req *contract.InvokeRequest) (*contract.InvokeResponse, error) {
	if m == nil {
		return nil, fmt.Errorf("sdk module is nil")
	}
	if req == nil {
		req = &contract.InvokeRequest{}
	}

	exportName := strings.TrimSpace(req.GetExportName())
	methodName := strings.TrimSpace(req.GetMethodName())
	handler, ok := m.dispatch[dispatchKey{exportName: exportName, methodName: methodName}]
	if !ok {
		return nil, fmt.Errorf("sdk invoke: unsupported export %q method %q", exportName, methodName)
	}

	result, err := handler(ctx, &Call{
		ExportName: exportName,
		MethodName: methodName,
		Args:       decodeArgs(req.GetArgs()),
		RawArgs:    append([]*structpb.Value(nil), req.GetArgs()...),
	})
	if err != nil {
		return nil, err
	}
	encoded, err := encodeResult(result)
	if err != nil {
		return nil, fmt.Errorf("sdk invoke %q/%q: %w", exportName, methodName, err)
	}
	return &contract.InvokeResponse{Result: encoded}, nil
}
