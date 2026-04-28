package main

//go:generate go run ../gen-dts --out ./js/src/types/goja-modules.d.ts --module fs,node:fs,exec,database,events,node:events,crypto,node:crypto,path,node:path,os,node:os --strict
//go:generate go run ./generate_build.go
