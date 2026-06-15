package main

import (
	"fmt"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth"
)

func main() {
	_, err := hostauth.ResolveConfig(hostauth.Config{
		Mode:   hostauth.ModeNone,
		Stores: hostauth.StoresConfig{Default: hostauth.StoreConfig{Driver: string(hostauth.StoreDriverPostgres)}},
	}, hostauth.ResolveOptions{})
	if err != nil {
		fmt.Printf("mode=none still validates store config and failed: %v\n", err)
		return
	}
	fmt.Println("mode=none ignored store config")
}
