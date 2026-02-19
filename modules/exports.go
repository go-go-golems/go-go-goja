package modules

import (
	"log"
)

// SetExport attaches a Go value to module exports.
func SetExport(exports settableObject, moduleName, name string, fn interface{}) {
	if err := exports.Set(name, fn); err != nil {
		log.Printf("%s: failed to set %s function: %v", moduleName, name, err)
	}
}

type settableObject interface {
	Set(name string, value interface{}) error
}
