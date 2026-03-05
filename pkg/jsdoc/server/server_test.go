package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
)

func TestServerHandlers(t *testing.T) {
	store := model.NewDocStore()
	store.AddFile(&model.FileDoc{
		FilePath: "a.js",
		Package:  &model.Package{Name: "p1", Title: "Package 1"},
		Symbols: []*model.SymbolDoc{
			{Name: "s1", Summary: "hello", Tags: []string{"t1"}},
		},
		Examples: []*model.Example{
			{ID: "e1", Title: "Example 1", Symbols: []string{"s1"}},
		},
	})

	s := New(store, ".", "127.0.0.1", 0)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	t.Run("store", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/store")
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var decoded model.DocStore
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&decoded))
		require.Contains(t, decoded.BySymbol, "s1")
		require.Equal(t, "s1", decoded.BySymbol["s1"].Name)
	})

	t.Run("symbol_enriched_examples", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/symbol/s1")
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var decoded struct {
			Name     string           `json:"name"`
			Examples []*model.Example `json:"examples"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&decoded))
		require.Equal(t, "s1", decoded.Name)
		require.Len(t, decoded.Examples, 1)
		require.Equal(t, "e1", decoded.Examples[0].ID)
	})
}
