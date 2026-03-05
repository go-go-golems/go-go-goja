package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDocStoreAddFileOverwritesByFilePath(t *testing.T) {
	store := NewDocStore()

	fd1 := &FileDoc{
		FilePath: "a.js",
		Package:  &Package{Name: "p1"},
		Symbols: []*SymbolDoc{
			{Name: "s1", Concepts: []string{"c1"}},
		},
		Examples: []*Example{{ID: "e1"}},
	}
	store.AddFile(fd1)

	require.Equal(t, 1, len(store.Files))
	require.Contains(t, store.ByPackage, "p1")
	require.Contains(t, store.BySymbol, "s1")
	require.Contains(t, store.ByExample, "e1")
	require.Equal(t, []string{"s1"}, store.ByConcept["c1"])

	fd2 := &FileDoc{
		FilePath: "a.js",
		Package:  &Package{Name: "p2"},
		Symbols: []*SymbolDoc{
			{Name: "s2", Concepts: []string{"c1"}},
		},
		Examples: []*Example{{ID: "e2"}},
	}
	store.AddFile(fd2)

	require.Equal(t, 1, len(store.Files))
	require.NotContains(t, store.ByPackage, "p1")
	require.Contains(t, store.ByPackage, "p2")

	require.NotContains(t, store.BySymbol, "s1")
	require.Contains(t, store.BySymbol, "s2")

	require.NotContains(t, store.ByExample, "e1")
	require.Contains(t, store.ByExample, "e2")

	require.Equal(t, []string{"s2"}, store.ByConcept["c1"])
}

func TestDocStoreAddFileCanBeUsedToRemoveEntries(t *testing.T) {
	store := NewDocStore()

	store.AddFile(&FileDoc{
		FilePath: "a.js",
		Package:  &Package{Name: "p1"},
		Symbols:  []*SymbolDoc{{Name: "s1", Concepts: []string{"c1"}}},
		Examples: []*Example{{ID: "e1"}},
	})

	store.AddFile(&FileDoc{FilePath: "a.js"})

	require.Equal(t, 1, len(store.Files))
	require.Equal(t, "a.js", store.Files[0].FilePath)
	require.Nil(t, store.Files[0].Package)
	require.Empty(t, store.Files[0].Symbols)
	require.Empty(t, store.Files[0].Examples)

	require.NotContains(t, store.ByPackage, "p1")
	require.NotContains(t, store.BySymbol, "s1")
	require.NotContains(t, store.ByExample, "e1")
	require.Empty(t, store.ByConcept["c1"])
}
