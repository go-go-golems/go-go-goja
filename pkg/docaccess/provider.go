package docaccess

import (
	"context"
	"errors"
)

var ErrEntryNotFound = errors.New("doc entry not found")

type Provider interface {
	Descriptor() SourceDescriptor
	List(ctx context.Context) ([]EntryRef, error)
	Get(ctx context.Context, ref EntryRef) (*Entry, error)
	Search(ctx context.Context, q Query) ([]Entry, error)
}
