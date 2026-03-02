package modules

import (
	"github.com/rs/zerolog/log"
)

// SetExport attaches a Go value to module exports.
func SetExport(exports settableObject, moduleName, name string, fn interface{}) {
	if err := exports.Set(name, fn); err != nil {
		log.Error().Str("module", moduleName).Str("export", name).Err(err).Msg("modules: failed to set export")
	}
}

type settableObject interface {
	Set(name string, value interface{}) error
}
