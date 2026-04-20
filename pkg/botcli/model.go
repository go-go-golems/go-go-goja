package botcli

import (
	"fmt"
	"path/filepath"

	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
)

const BotRepositoryFlag = "bot-repository"

type Bootstrap struct {
	Repositories []Repository
}

type Repository struct {
	Name      string
	Source    string
	SourceRef string
	RootDir   string
}

type ScannedRepository struct {
	Repository Repository
	Registry   *jsverbs.Registry
}

type DiscoveredBot struct {
	Repository ScannedRepository
	Verb       *jsverbs.VerbSpec
}

func (b DiscoveredBot) FullPath() string {
	if b.Verb == nil {
		return ""
	}
	return b.Verb.FullPath()
}

func (b DiscoveredBot) SourceRef() string {
	if b.Verb == nil {
		return b.Repository.Repository.Name
	}
	return b.Verb.SourceRef()
}

func (b DiscoveredBot) SourceLabel() string {
	if b.Verb == nil || b.Verb.File == nil {
		return b.Repository.Repository.Name
	}
	if b.Verb.File.AbsPath != "" {
		if rel, err := filepath.Rel(b.Repository.Repository.RootDir, b.Verb.File.AbsPath); err == nil {
			return filepath.ToSlash(rel)
		}
		return b.Verb.File.AbsPath
	}
	return b.Verb.File.RelPath
}

func (b DiscoveredBot) Validate() error {
	if b.Repository.Registry == nil {
		return fmt.Errorf("bot registry is nil")
	}
	if b.Verb == nil {
		return fmt.Errorf("bot verb is nil")
	}
	return nil
}
