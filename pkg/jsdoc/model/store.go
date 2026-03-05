package model

// DocStore is the aggregate of all parsed FileDoc entries.
type DocStore struct {
	Files []*FileDoc `json:"files"`

	// Indexes for fast lookup.
	ByPackage map[string]*Package   `json:"by_package,omitempty"`
	BySymbol  map[string]*SymbolDoc `json:"by_symbol,omitempty"`
	ByExample map[string]*Example   `json:"by_example,omitempty"`

	// Concept index: concept → []symbol names.
	ByConcept map[string][]string `json:"by_concept,omitempty"`
}

// NewDocStore creates an empty DocStore.
func NewDocStore() *DocStore {
	return &DocStore{
		ByPackage: make(map[string]*Package),
		BySymbol:  make(map[string]*SymbolDoc),
		ByExample: make(map[string]*Example),
		ByConcept: make(map[string][]string),
	}
}

// AddFile integrates a FileDoc into the store, updating all indexes.
func (s *DocStore) AddFile(fd *FileDoc) {
	// Remove existing entries for this file.
	s.removeFile(fd.FilePath)

	s.Files = append(s.Files, fd)

	if fd.Package != nil {
		s.ByPackage[fd.Package.Name] = fd.Package
	}
	for _, sym := range fd.Symbols {
		s.BySymbol[sym.Name] = sym
		for _, c := range sym.Concepts {
			s.ByConcept[c] = appendUnique(s.ByConcept[c], sym.Name)
		}
	}
	for _, ex := range fd.Examples {
		s.ByExample[ex.ID] = ex
	}
}

// removeFile removes all entries from a given file path.
func (s *DocStore) removeFile(filePath string) {
	var remaining []*FileDoc
	for _, fd := range s.Files {
		if fd.FilePath == filePath {
			// Remove from indexes.
			if fd.Package != nil {
				delete(s.ByPackage, fd.Package.Name)
			}
			for _, sym := range fd.Symbols {
				delete(s.BySymbol, sym.Name)
				for _, c := range sym.Concepts {
					s.ByConcept[c] = removeItem(s.ByConcept[c], sym.Name)
				}
			}
			for _, ex := range fd.Examples {
				delete(s.ByExample, ex.ID)
			}
		} else {
			remaining = append(remaining, fd)
		}
	}
	s.Files = remaining
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

func removeItem(slice []string, item string) []string {
	out := slice[:0]
	for _, s := range slice {
		if s != item {
			out = append(out, s)
		}
	}
	return out
}
