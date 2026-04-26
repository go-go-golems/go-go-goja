package main

//go:generate go run ../gen-dts --out ./js/src/types/goja-modules.d.ts --module fs,exec,database,events,node:events --strict
//go:generate go run ./generate_build.go
