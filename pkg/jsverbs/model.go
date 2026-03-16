package jsverbs

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type ParameterKind string

const (
	ParameterIdentifier ParameterKind = "identifier"
	ParameterObject     ParameterKind = "object"
	ParameterArray      ParameterKind = "array"
	ParameterUnknown    ParameterKind = "unknown"
)

type DiagnosticSeverity string

const (
	DiagnosticSeverityWarning DiagnosticSeverity = "warning"
	DiagnosticSeverityError   DiagnosticSeverity = "error"
)

type Diagnostic struct {
	Severity DiagnosticSeverity
	Path     string
	Symbol   string
	Message  string
}

type ScanError struct {
	Diagnostics []Diagnostic
}

func (e *ScanError) Error() string {
	if e == nil || len(e.Diagnostics) == 0 {
		return "jsverbs scan failed"
	}
	parts := make([]string, 0, len(e.Diagnostics))
	for _, diagnostic := range e.Diagnostics {
		location := diagnostic.Path
		if diagnostic.Symbol != "" {
			location += "#" + diagnostic.Symbol
		}
		if location == "" {
			location = "jsverbs"
		}
		parts = append(parts, fmt.Sprintf("%s: %s", location, diagnostic.Message))
	}
	return strings.Join(parts, "; ")
}

type ScanOptions struct {
	IncludePublicFunctions bool
	Extensions             []string
	FailOnErrorDiagnostics bool
}

func DefaultScanOptions() ScanOptions {
	return ScanOptions{
		IncludePublicFunctions: true,
		Extensions:             []string{".js", ".cjs"},
		FailOnErrorDiagnostics: true,
	}
}

type SourceFile struct {
	Path   string
	Source []byte
}

type Registry struct {
	RootDir       string
	Files         []*FileSpec
	Diagnostics   []Diagnostic
	verbs         []*VerbSpec
	verbsByKey    map[string]*VerbSpec
	filesByModule map[string]*FileSpec
	options       ScanOptions
}

type FileSpec struct {
	AbsPath        string
	RelPath        string
	ModulePath     string
	Source         []byte
	Package        PackageSpec
	Functions      []*FunctionSpec
	functionByName map[string]*FunctionSpec
	SectionOrder   []string
	Sections       map[string]*SectionSpec
	VerbMeta       map[string]*VerbSpec
	Docs           map[string]string
}

type PackageSpec struct {
	Name    string
	Short   string
	Long    string
	Parents []string
	Tags    []string
}

type FunctionSpec struct {
	Name   string
	Params []ParameterSpec
	Doc    string
}

type ParameterSpec struct {
	Name string
	Kind ParameterKind
	Rest bool
}

type SectionSpec struct {
	Slug        string
	Title       string
	Description string
	Fields      map[string]*FieldSpec
}

type FieldSpec struct {
	Name     string
	Type     string
	Help     string
	Short    string
	Bind     string
	Section  string
	Default  interface{}
	Choices  []string
	Required bool
	Argument bool
}

type VerbSpec struct {
	FunctionName string
	Name         string
	Short        string
	Long         string
	OutputMode   string
	Parents      []string
	Tags         []string
	UseSections  []string
	Fields       map[string]*FieldSpec
	File         *FileSpec
	Params       []ParameterSpec
}

const (
	OutputModeGlaze = "glaze"
	OutputModeText  = "text"
)

func (r *Registry) Verbs() []*VerbSpec {
	ret := make([]*VerbSpec, 0, len(r.verbs))
	ret = append(ret, r.verbs...)
	return ret
}

func (r *Registry) Verb(fullPath string) (*VerbSpec, bool) {
	v, ok := r.verbsByKey[fullPath]
	return v, ok
}

func (r *Registry) ErrorDiagnostics() []Diagnostic {
	ret := []Diagnostic{}
	for _, diagnostic := range r.Diagnostics {
		if diagnostic.Severity == DiagnosticSeverityError {
			ret = append(ret, diagnostic)
		}
	}
	return ret
}

func (v *VerbSpec) FullPath() string {
	if len(v.Parents) == 0 {
		return v.Name
	}
	return strings.Join(append(append([]string{}, v.Parents...), v.Name), " ")
}

func (v *VerbSpec) SourceRef() string {
	if v == nil || v.File == nil {
		return ""
	}
	return fmt.Sprintf("%s#%s", v.File.RelPath, v.FunctionName)
}

func (v *VerbSpec) Field(name string) *FieldSpec {
	if v == nil || v.Fields == nil {
		return nil
	}
	if field, ok := v.Fields[name]; ok {
		return field
	}
	return nil
}

func (f *FieldSpec) Clone() *FieldSpec {
	if f == nil {
		return nil
	}
	ret := *f
	ret.Choices = append([]string{}, f.Choices...)
	return &ret
}

func (s *SectionSpec) Clone() *SectionSpec {
	if s == nil {
		return nil
	}
	ret := &SectionSpec{
		Slug:        s.Slug,
		Title:       s.Title,
		Description: s.Description,
		Fields:      map[string]*FieldSpec{},
	}
	fieldNames := make([]string, 0, len(s.Fields))
	for name := range s.Fields {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)
	for _, name := range fieldNames {
		ret.Fields[name] = s.Fields[name].Clone()
	}
	return ret
}

func cleanCommandWord(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return kebabCase(s)
}

func kebabCase(s string) string {
	s = filepath.ToSlash(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	var out []rune
	var prevLower bool
	for i, r := range s {
		switch {
		case r == '/' || r == '_' || r == ' ':
			if len(out) > 0 && out[len(out)-1] != '-' {
				out = append(out, '-')
			}
			prevLower = false
		case r >= 'A' && r <= 'Z':
			if i > 0 && prevLower && out[len(out)-1] != '-' {
				out = append(out, '-')
			}
			out = append(out, r+'a'-'A')
			prevLower = false
		default:
			out = append(out, r)
			prevLower = (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		}
	}
	return strings.Trim(strings.ReplaceAll(string(out), "--", "-"), "-")
}

func dedupeStrings(items []string) []string {
	seen := map[string]struct{}{}
	ret := make([]string, 0, len(items))
	for _, item := range items {
		item = cleanCommandWord(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		ret = append(ret, item)
	}
	return ret
}
