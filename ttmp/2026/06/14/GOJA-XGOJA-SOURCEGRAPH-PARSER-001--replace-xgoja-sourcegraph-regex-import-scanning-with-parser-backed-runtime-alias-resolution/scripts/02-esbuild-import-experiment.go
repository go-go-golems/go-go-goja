//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/evanw/esbuild/pkg/api"
)

type metafile struct {
	Inputs map[string]struct {
		Imports []struct {
			Path     string `json:"path"`
			Kind     string `json:"kind"`
			External bool   `json:"external"`
		} `json:"imports"`
	} `json:"inputs"`
}

func write(path, s string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		panic(err)
	}
}

func run(label, entry string, external []string) {
	var resolved []string
	result := api.Build(api.BuildOptions{
		EntryPoints: []string{entry},
		Bundle:      true,
		Write:       false,
		Metafile:    true,
		LogLevel:    api.LogLevelSilent,
		Platform:    api.PlatformNeutral,
		Format:      api.FormatCommonJS,
		External:    external,
		Plugins: []api.Plugin{{
			Name: "trace-resolve",
			Setup: func(build api.PluginBuild) {
				build.OnResolve(api.OnResolveOptions{Filter: ".*"}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					resolved = append(resolved, fmt.Sprintf("path=%q importer=%q kind=%v namespace=%q", args.Path, args.Importer, args.Kind, args.Namespace))
					return api.OnResolveResult{}, nil
				})
			},
		}},
	})
	fmt.Println("\n==", label, "==")
	fmt.Println("errors:", len(result.Errors), "warnings:", len(result.Warnings))
	for _, e := range result.Errors {
		fmt.Println("ERR:", e.Text)
	}
	fmt.Println("OnResolve callbacks:")
	for _, s := range resolved {
		fmt.Println(" ", s)
	}
	if result.Metafile != "" {
		var mf metafile
		if err := json.Unmarshal([]byte(result.Metafile), &mf); err != nil {
			panic(err)
		}
		fmt.Println("Metafile imports:")
		for input, data := range mf.Inputs {
			for _, imp := range data.Imports {
				fmt.Printf("  from=%s path=%q kind=%s external=%v\n", input, imp.Path, imp.Kind, imp.External)
			}
		}
	}
}

func main() {
	dir, err := os.MkdirTemp("", "xgoja-esbuild-imports-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	write(filepath.Join(dir, "helper.js"), `export const helper = "helper";`)
	write(filepath.Join(dir, "more.js"), `export const more = "more";`)
	write(filepath.Join(dir, "dynamic.js"), `export const dyn = "dyn";`)
	write(filepath.Join(dir, "entry.js"), `
import "./helper.js";
export { more } from "./more.js";
const assets = require("fs:assets");
const express = require("express");
import("./dynamic.js");
const dynamicRequire = require(["fs", "host"].join(":"));
`)
	write(filepath.Join(dir, "entry.ts"), `
import type { Thing } from "./types";
import { helper } from "./helper";
import assets from "fs:assets";
export { more } from "./more";
export interface Thing { name: string }
console.log(helper, assets);
`)
	write(filepath.Join(dir, "types.ts"), `export interface Thing { name: string }`)

	run("JS without externals", filepath.Join(dir, "entry.js"), nil)
	run("JS with colon/bare externals", filepath.Join(dir, "entry.js"), []string{"fs:assets", "fs:host", "express"})
	run("TS with colon external", filepath.Join(dir, "entry.ts"), []string{"fs:assets"})
}
