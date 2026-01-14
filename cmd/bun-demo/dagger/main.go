package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dagger.io/dagger"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const (
	nodeImage      = "node:20.18.1"
	jsDir          = "js"
	assetsDir      = "assets"
	assetsSplitDir = "assets-split"
)

var (
	logLevel   string
	projectDir string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "bun-demo-dagger",
		Short: "Run Dagger pipelines for bun-demo assets",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return configureLogging(logLevel)
		},
	}

	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&projectDir, "project-dir", ".", "path to cmd/bun-demo directory")

	rootCmd.AddCommand(
		newDepsCmd(),
		newBundleCmd(),
		newBundleSplitCmd(),
		newBundleTSXCmd(),
		newRenderTSXCmd(),
		newTypecheckCmd(),
		newTranspileCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func configureLogging(level string) error {
	parsed, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		return errors.Wrap(err, "parse log level")
	}
	zerolog.SetGlobalLevel(parsed)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	return nil
}

type pipeline struct {
	client     *dagger.Client
	projectDir string
}

func withPipeline(ctx context.Context, projectDir string, run func(ctx context.Context, pipe *pipeline) error) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		return errors.Wrap(err, "connect to dagger")
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Error().Err(err).Msg("close dagger client")
		}
	}()

	pipe := &pipeline{
		client:     client,
		projectDir: filepath.Clean(projectDir),
	}
	if err := pipe.validateProjectDir(); err != nil {
		return err
	}
	return run(ctx, pipe)
}

func (p *pipeline) validateProjectDir() error {
	manifest := filepath.Join(p.projectDir, jsDir, "package.json")
	if _, err := os.Stat(manifest); err != nil {
		return errors.Wrapf(err, "expected bun-demo JS workspace at %s", manifest)
	}
	return nil
}

func (p *pipeline) jsContainer() *dagger.Container {
	src := p.client.Host().Directory(p.projectDir)
	cache := p.client.CacheVolume("bun-demo-npm-cache")
	return p.client.Container().
		From(nodeImage).
		WithMountedDirectory("/src", src).
		WithMountedCache("/root/.npm", cache).
		WithWorkdir(filepath.ToSlash(filepath.Join("/src", jsDir))).
		WithEnvVariable("CI", "1").
		WithExec([]string{"npm", "install", "--no-audit", "--no-fund"})
}

func (p *pipeline) exportFile(ctx context.Context, file *dagger.File, dest string) error {
	dest = filepath.Clean(dest)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return errors.Wrapf(err, "ensure output dir for %s", dest)
	}
	if _, err := file.Export(ctx, dest); err != nil {
		return errors.Wrapf(err, "export file to %s", dest)
	}
	return nil
}

func (p *pipeline) exportDir(ctx context.Context, dir *dagger.Directory, dest string) error {
	dest = filepath.Clean(dest)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return errors.Wrapf(err, "ensure output dir for %s", dest)
	}
	if _, err := dir.Export(ctx, dest); err != nil {
		return errors.Wrapf(err, "export directory to %s", dest)
	}
	return nil
}

func (p *pipeline) runScript(script string) *dagger.Container {
	return p.jsContainer().WithExec([]string{"npm", "run", script})
}

func (p *pipeline) bundle(ctx context.Context) error {
	ctr := p.runScript("build")
	file := ctr.File("/src/js/dist/bundle.cjs")
	dest := filepath.Join(p.projectDir, assetsDir, "bundle.cjs")
	return p.exportFile(ctx, file, dest)
}

func (p *pipeline) bundleSplit(ctx context.Context) error {
	ctr := p.runScript("build:split")
	dir := ctr.Directory("/src/js/dist-split")
	dest := filepath.Join(p.projectDir, assetsSplitDir)
	return p.exportDir(ctx, dir, dest)
}

func (p *pipeline) bundleTSX(ctx context.Context) error {
	ctr := p.runScript("build:tsx")
	file := ctr.File("/src/js/dist/tsx-bundle.cjs")
	dest := filepath.Join(p.projectDir, assetsDir, "tsx-bundle.cjs")
	return p.exportFile(ctx, file, dest)
}

func (p *pipeline) renderTSX(ctx context.Context) error {
	ctr := p.runScript("build:tsx").
		WithExec([]string{
			"node",
			"-e",
			"const fs=require('fs');const mod=require('./dist/tsx-bundle.cjs');const html=(mod.renderHtml?mod.renderHtml():mod.run());fs.writeFileSync('dist/tsx.html', html);",
		})
	file := ctr.File("/src/js/dist/tsx.html")
	dest := filepath.Join(p.projectDir, assetsDir, "tsx.html")
	return p.exportFile(ctx, file, dest)
}

func (p *pipeline) typecheck(ctx context.Context) error {
	ctr := p.runScript("typecheck")
	if _, err := ctr.Sync(ctx); err != nil {
		return errors.Wrap(err, "run typecheck")
	}
	return nil
}

func (p *pipeline) transpile(ctx context.Context) error {
	ctr := p.runScript("build").
		WithExec([]string{
			"./node_modules/.bin/esbuild",
			"dist/bundle.cjs",
			"--target=es5",
			"--format=cjs",
			"--outfile=dist/bundle.es5.cjs",
		})
	file := ctr.File("/src/js/dist/bundle.es5.cjs")
	dest := filepath.Join(p.projectDir, jsDir, "dist", "bundle.es5.cjs")
	return p.exportFile(ctx, file, dest)
}

func newDepsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deps",
		Short: "Install JS dependencies in a Dagger container",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPipeline(cmd.Context(), projectDir, func(ctx context.Context, pipe *pipeline) error {
				ctr := pipe.jsContainer()
				if _, err := ctr.Sync(ctx); err != nil {
					return errors.Wrap(err, "install dependencies")
				}
				return nil
			})
		},
	}
}

func newBundleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bundle",
		Short: "Bundle the main demo entrypoint",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPipeline(cmd.Context(), projectDir, func(ctx context.Context, pipe *pipeline) error {
				return pipe.bundle(ctx)
			})
		},
	}
}

func newBundleSplitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bundle-split",
		Short: "Bundle the split demo entrypoints",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPipeline(cmd.Context(), projectDir, func(ctx context.Context, pipe *pipeline) error {
				return pipe.bundleSplit(ctx)
			})
		},
	}
}

func newBundleTSXCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bundle-tsx",
		Short: "Bundle the TSX demo entrypoint",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPipeline(cmd.Context(), projectDir, func(ctx context.Context, pipe *pipeline) error {
				return pipe.bundleTSX(ctx)
			})
		},
	}
}

func newRenderTSXCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "render-tsx",
		Short: "Render TSX HTML from the bundled output",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPipeline(cmd.Context(), projectDir, func(ctx context.Context, pipe *pipeline) error {
				return pipe.renderTSX(ctx)
			})
		},
	}
}

func newTypecheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "typecheck",
		Short: "Run the TypeScript typecheck",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPipeline(cmd.Context(), projectDir, func(ctx context.Context, pipe *pipeline) error {
				return pipe.typecheck(ctx)
			})
		},
	}
}

func newTranspileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "transpile",
		Short: "Transpile the bundle to ES5",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPipeline(cmd.Context(), projectDir, func(ctx context.Context, pipe *pipeline) error {
				return pipe.transpile(ctx)
			})
		},
	}
}
