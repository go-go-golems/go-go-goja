package host

import "github.com/go-go-golems/go-go-goja/engine"

type StringSliceFlag []string

func (s *StringSliceFlag) String() string {
	if s == nil {
		return ""
	}
	out := make([]string, len(*s))
	copy(out, *s)
	return joinCSV(out)
}

func (s *StringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type RuntimeSetup struct {
	Directories  []string
	AllowModules []string
	Reporter     *ReportCollector
}

func NewRuntimeSetup(directories, allowModules []string) RuntimeSetup {
	resolvedDirectories := ResolveDiscoveryDirectories(directories)
	return RuntimeSetup{
		Directories:  resolvedDirectories,
		AllowModules: normalizeModuleNames(allowModules),
		Reporter:     NewReportCollector(resolvedDirectories),
	}
}

func (s RuntimeSetup) WithBuilder(builder *engine.FactoryBuilder) *engine.FactoryBuilder {
	if builder == nil {
		return nil
	}
	if len(s.Directories) == 0 {
		return builder
	}
	return builder.WithRuntimeModuleRegistrars(NewRegistrar(Config{
		Directories:  s.Directories,
		AllowModules: s.AllowModules,
		Report:       s.Reporter,
	}))
}

func (s RuntimeSetup) Snapshot() LoadReport {
	if s.Reporter == nil {
		return LoadReport{Directories: append([]string(nil), s.Directories...)}
	}
	return s.Reporter.Snapshot()
}

func joinCSV(values []string) string {
	if len(values) == 0 {
		return ""
	}
	out := values[0]
	for i := 1; i < len(values); i++ {
		out += "," + values[i]
	}
	return out
}
