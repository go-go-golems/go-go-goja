package main

import (
	"context"
	"sort"
	"sync"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/sdk"
)

func main() {
	store := &kvStore{
		values: map[string]string{},
	}

	sdk.Serve(
		sdk.MustModule(
			"plugin:kv",
			sdk.Version("v1"),
			sdk.Doc("Example stateful plugin with object methods"),
			sdk.Capabilities("examples", "stateful", "object-methods"),
			sdk.Object("store",
				sdk.ObjectDoc("In-memory key/value store scoped to the plugin process"),
				sdk.Method("set", store.set, sdk.ExportDoc("Set a key to a string value")),
				sdk.Method("get", store.get, sdk.ExportDoc("Get a key, returning null if it is absent")),
				sdk.Method("delete", store.delete, sdk.ExportDoc("Delete a key and report whether it existed")),
				sdk.Method("keys", store.keys, sdk.ExportDoc("List keys in sorted order")),
				sdk.Method("clear", store.clear, sdk.ExportDoc("Remove all entries")),
				sdk.Method("size", store.size, sdk.ExportDoc("Return the number of stored entries")),
			),
		),
	)
}

type kvStore struct {
	mu     sync.Mutex
	values map[string]string
}

func (s *kvStore) set(_ context.Context, call *sdk.Call) (any, error) {
	key, err := call.String(0)
	if err != nil {
		return nil, err
	}
	value, err := call.String(1)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	return map[string]any{
		"key":   key,
		"value": value,
		"size":  len(s.values),
	}, nil
}

func (s *kvStore) get(_ context.Context, call *sdk.Call) (any, error) {
	key, err := call.String(0)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[key]
	if !ok {
		return nil, nil
	}
	return value, nil
}

func (s *kvStore) delete(_ context.Context, call *sdk.Call) (any, error) {
	key, err := call.String(0)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, existed := s.values[key]
	delete(s.values, key)
	return map[string]any{
		"deleted": existed,
		"size":    len(s.values),
	}, nil
}

func (s *kvStore) keys(context.Context, *sdk.Call) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.values))
	for key := range s.values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, key)
	}
	return out, nil
}

func (s *kvStore) clear(context.Context, *sdk.Call) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := len(s.values)
	s.values = map[string]string{}
	return map[string]any{
		"cleared": count,
	}, nil
}

func (s *kvStore) size(context.Context, *sdk.Call) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.values), nil
}
