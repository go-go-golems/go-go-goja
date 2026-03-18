package docaccess

type SourceKind string

const (
	SourceKindGlazedHelp SourceKind = "glazed-help"
	SourceKindJSDoc      SourceKind = "jsdoc"
	SourceKindPlugin     SourceKind = "plugin"
	SourceKindDocmgr     SourceKind = "docmgr"
)

type SourceDescriptor struct {
	ID            string
	Kind          SourceKind
	Title         string
	Summary       string
	RuntimeScoped bool
	Metadata      map[string]any
}

type EntryRef struct {
	SourceID string
	Kind     string
	ID       string
}

type Entry struct {
	Ref       EntryRef
	Title     string
	Summary   string
	Body      string
	Topics    []string
	Tags      []string
	Path      string
	KindLabel string
	Related   []EntryRef
	Metadata  map[string]any
}

type Query struct {
	Text      string
	SourceIDs []string
	Kinds     []string
	Topics    []string
	Tags      []string
	Limit     int
}
