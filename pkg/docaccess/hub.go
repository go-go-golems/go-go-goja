package docaccess

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Hub struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

func NewHub() *Hub {
	return &Hub{
		providers: map[string]Provider{},
	}
}

func (h *Hub) Register(provider Provider) error {
	if h == nil {
		return fmt.Errorf("docaccess hub is nil")
	}
	if provider == nil {
		return fmt.Errorf("docaccess provider is nil")
	}
	descriptor := provider.Descriptor()
	if strings.TrimSpace(descriptor.ID) == "" {
		return fmt.Errorf("docaccess provider has empty source ID")
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.providers[descriptor.ID]; ok {
		return fmt.Errorf("docaccess provider %q already registered", descriptor.ID)
	}
	h.providers[descriptor.ID] = provider
	return nil
}

func (h *Hub) MustRegister(provider Provider) {
	if err := h.Register(provider); err != nil {
		panic(err)
	}
}

func (h *Hub) Sources() []SourceDescriptor {
	if h == nil {
		return nil
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	out := make([]SourceDescriptor, 0, len(h.providers))
	for _, provider := range h.providers {
		out = append(out, provider.Descriptor())
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (h *Hub) Get(ctx context.Context, ref EntryRef) (*Entry, error) {
	if h == nil {
		return nil, fmt.Errorf("docaccess hub is nil")
	}

	provider, ok := h.provider(ref.SourceID)
	if !ok {
		return nil, ErrEntryNotFound
	}
	return provider.Get(ctx, ref)
}

func (h *Hub) FindByID(sourceID, kind, id string) (*Entry, error) {
	return h.Get(context.Background(), EntryRef{
		SourceID: sourceID,
		Kind:     kind,
		ID:       id,
	})
}

func (h *Hub) Search(ctx context.Context, q Query) ([]Entry, error) {
	if h == nil {
		return nil, fmt.Errorf("docaccess hub is nil")
	}

	providers := h.providersForQuery(q)
	out := make([]Entry, 0, 16)
	for _, provider := range providers {
		entries, err := provider.Search(ctx, q)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if MatchesQuery(entry, q) {
				out = append(out, entry)
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Ref.SourceID != out[j].Ref.SourceID {
			return out[i].Ref.SourceID < out[j].Ref.SourceID
		}
		if out[i].Ref.Kind != out[j].Ref.Kind {
			return out[i].Ref.Kind < out[j].Ref.Kind
		}
		if out[i].Title != out[j].Title {
			return out[i].Title < out[j].Title
		}
		return out[i].Ref.ID < out[j].Ref.ID
	})

	if q.Limit > 0 && len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out, nil
}

func (h *Hub) provider(sourceID string) (Provider, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	provider, ok := h.providers[sourceID]
	return provider, ok
}

func (h *Hub) providersForQuery(q Query) []Provider {
	h.mu.RLock()
	defer h.mu.RUnlock()

	out := make([]Provider, 0, len(h.providers))
	if len(q.SourceIDs) == 0 {
		for _, provider := range h.providers {
			out = append(out, provider)
		}
		return out
	}

	for _, sourceID := range q.SourceIDs {
		if provider, ok := h.providers[sourceID]; ok {
			out = append(out, provider)
		}
	}
	return out
}
