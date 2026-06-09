package fs

import "os"

type Backend interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, mode os.FileMode) error
	Exists(path string) bool
	Mkdir(path string, recursive bool, mode os.FileMode) error
	ReadDir(path string) ([]string, error)
	Stat(path string) (fileStats, error)
	Remove(path string) error
	AppendFile(path string, data []byte, mode os.FileMode) error
	Rename(oldPath, newPath string) error
	CopyFile(src, dst string) error
	RemoveAll(path string) error
}

type MountInfo struct {
	Mount string `json:"mount"`
	Root  string `json:"root"`
}

type Capabilities struct {
	Backend  string      `json:"backend"`
	Read     bool        `json:"read"`
	Write    bool        `json:"write"`
	Embedded bool        `json:"embedded"`
	Mounts   []MountInfo `json:"mounts,omitempty"`
}

type CapabilityReporter interface {
	FSCapabilities() Capabilities
}

func CapabilitiesForBackend(backend Backend) Capabilities {
	if reporter, ok := backend.(CapabilityReporter); ok {
		return reporter.FSCapabilities()
	}
	return Capabilities{Backend: "custom", Read: true, Write: true}
}

type Option func(*m)

func New(opts ...Option) *m {
	mod := &m{name: "fs", backend: OSBackend{}}
	for _, opt := range opts {
		if opt != nil {
			opt(mod)
		}
	}
	return mod
}

func WithName(name string) Option {
	return func(mod *m) {
		mod.name = name
	}
}

func WithBackend(backend Backend) Option {
	return func(mod *m) {
		if backend != nil {
			mod.backend = backend
		}
	}
}

func (mod m) fileSystem() Backend {
	if mod.backend == nil {
		return OSBackend{}
	}
	return mod.backend
}
