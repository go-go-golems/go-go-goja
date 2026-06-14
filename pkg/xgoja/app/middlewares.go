package app

import (
	"context"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	cmdsources "github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	glazedconfig "github.com/go-go-golems/glazed/pkg/config"
	"github.com/spf13/cobra"
)

// MiddlewaresFromSpec returns the Glazed parser middleware chain for generated
// xgoja commands. If the runtime plan does not request an environment namespace or
// config file loading, it preserves the historical default chain: Cobra flags,
// positional arguments, and field defaults only.
func MiddlewaresFromSpec(runtimePlan *RuntimePlan) cli.CobraMiddlewaresFunc {
	envPrefix := EffectiveEnvPrefix(runtimePlan)
	hasConfig := runtimePlan != nil && runtimePlan.App.ConfigFile != nil && runtimePlan.App.ConfigFile.Enabled

	if envPrefix == "" && !hasConfig {
		return cli.CobraCommandDefaultMiddlewares
	}

	return func(parsedCommandSections *values.Values, cmd *cobra.Command, args []string) ([]cmdsources.Middleware, error) {
		// The returned slice is ordered from highest to lowest precedence because
		// Glazed middlewares call next before applying their own source values.
		// Effective precedence is: defaults < config < env < args < cobra flags.
		middlewares := []cmdsources.Middleware{
			cmdsources.FromCobra(cmd, fields.WithSource("cobra")),
			cmdsources.FromArgs(args, fields.WithSource("arguments")),
		}

		if envPrefix != "" {
			middlewares = append(middlewares, cmdsources.FromEnv(envPrefix, fields.WithSource("env")))
		}

		if hasConfig {
			middlewares = append(middlewares,
				cmdsources.FromConfigPlanBuilder(
					func(_ context.Context, _ *values.Values) (*glazedconfig.Plan, error) {
						return buildConfigPlan(runtimePlan.App.ConfigFile, runtimePlan.AppName(), parsedCommandSections)
					},
					cmdsources.WithParseOptions(fields.WithSource("config")),
				),
			)
		}

		middlewares = append(middlewares,
			cmdsources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
		)

		return middlewares, nil
	}
}

func buildConfigPlan(config *ConfigFilePlan, appName string, parsed *values.Values) (*glazedconfig.Plan, error) {
	explicit := ""
	if parsed != nil {
		commandSettings := &cli.CommandSettings{}
		if err := parsed.DecodeSectionInto(cli.CommandSettingsSlug, commandSettings); err == nil {
			explicit = strings.TrimSpace(commandSettings.ConfigFile)
		}
	}

	fileName := strings.TrimSpace(config.FileName)
	if fileName == "" {
		fileName = "config.yaml"
	}

	plan := glazedconfig.NewPlan(
		glazedconfig.WithLayerOrder(
			glazedconfig.LayerSystem,
			glazedconfig.LayerUser,
			glazedconfig.LayerRepo,
			glazedconfig.LayerCWD,
			glazedconfig.LayerExplicit,
		),
		glazedconfig.WithDedupePaths(),
	)

	for _, layer := range config.Layers {
		switch strings.TrimSpace(layer) {
		case "system":
			plan.Add(glazedconfig.SystemAppConfig(appName).Named("system-app-config").Kind("app-config"))
		case "xdg":
			plan.Add(glazedconfig.XDGAppConfig(appName).Named("xdg-app-config").Kind("app-config"))
		case "home":
			plan.Add(glazedconfig.HomeAppConfig(appName).Named("home-app-config").Kind("app-config"))
		case "git-root":
			plan.Add(glazedconfig.GitRootFile(fileName).Named("git-root-config").Kind("local-file"))
		case "cwd":
			plan.Add(glazedconfig.WorkingDirFile(fileName).Named("cwd-config").Kind("local-file"))
		case "explicit":
			if explicit != "" {
				plan.Add(glazedconfig.ExplicitFile(explicit).Named("explicit-config").Kind("explicit-file"))
			}
		}
	}

	return plan, nil
}

// EffectiveEnvPrefix returns the explicit envPrefix when present, otherwise a
// shell-safe prefix derived from appName. The runtime plan name is intentionally not
// used as an implicit env namespace; existing plans without app.name/app.envPrefix
// must keep their historical flag/argument/default-only parser behavior.
func EffectiveEnvPrefix(runtimePlan *RuntimePlan) string {
	if runtimePlan == nil {
		return ""
	}
	if prefix := strings.TrimSpace(runtimePlan.App.EnvPrefix); prefix != "" {
		return strings.ToUpper(prefix)
	}
	return DefaultEnvPrefix(runtimePlan.App.Name)
}

// DefaultEnvPrefix converts an application name into a shell-safe environment
// variable namespace. It is deliberately stricter than Glazed's built-in
// strings.ToUpper(AppName) behavior because generated binaries commonly use
// hyphenated app names such as "my-app".
func DefaultEnvPrefix(appName string) string {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		return ""
	}
	var b strings.Builder
	lastUnderscore := false
	for _, r := range appName {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r - 'a' + 'A')
			lastUnderscore = false
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		case r == '-' || r == '_' || r == '.' || r == ' ':
			if b.Len() > 0 && !lastUnderscore {
				b.WriteRune('_')
				lastUnderscore = true
			}
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return ""
	}
	if out[0] >= '0' && out[0] <= '9' {
		out = "APP_" + out
	}
	return out
}
