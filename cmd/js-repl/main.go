package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-go-golems/bobatea/pkg/eventbus"
	"github.com/go-go-golems/bobatea/pkg/logutil"
	"github.com/go-go-golems/bobatea/pkg/repl"
	"github.com/go-go-golems/bobatea/pkg/timeline"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/host"
	jsadapter "github.com/go-go-golems/go-go-goja/pkg/repl/adapters/bobatea"
	js "github.com/go-go-golems/go-go-goja/pkg/repl/evaluators/javascript"
	"github.com/rs/zerolog"
)

type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func parseLevel(s string) zerolog.Level {
	switch strings.ToLower(s) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error", "err":
		return zerolog.ErrorLevel
	default:
		return zerolog.ErrorLevel
	}
}

func main() {
	ll := flag.String("log-level", "error", "log level: trace, debug, info, warn, error")
	lf := flag.String("log-file", "", "log file path (optional)")
	var pluginDirs stringSliceFlag
	flag.Var(&pluginDirs, "plugin-dir", fmt.Sprintf("directory containing HashiCorp go-plugin module binaries (defaults to %s/... when omitted)", host.DefaultDiscoveryRoot()))
	flag.Parse()

	level := parseLevel(*ll)
	if *lf != "" {
		logutil.InitTUILoggingToFile(level, *lf)
	} else {
		logutil.InitTUILoggingToDiscard(level)
	}

	evaluatorConfig := js.DefaultConfig()
	evaluatorConfig.PluginDirectories = host.ResolveDiscoveryDirectories(pluginDirs)
	evaluator, err := jsadapter.NewJavaScriptEvaluator(evaluatorConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = evaluator.Close()
	}()

	cfg := repl.DefaultConfig()
	cfg.Title = "go-go-goja JavaScript REPL (Bobatea UI + jsparse completion/help)"
	cfg.Placeholder = "Type console.lo or fs.re, then use alt+h (drawer), ctrl+h (full help), ctrl+r (refresh)"
	cfg.Autocomplete.Enabled = true
	cfg.Autocomplete.FocusToggleKey = "ctrl+t"
	cfg.Autocomplete.TriggerKeys = []string{"tab"}
	cfg.Autocomplete.AcceptKeys = []string{"enter", "tab"}
	cfg.HelpBar.Enabled = true
	cfg.HelpDrawer.Enabled = true

	bus, err := eventbus.NewInMemoryBus()
	if err != nil {
		log.Fatal(err)
	}
	repl.RegisterReplToTimelineTransformer(bus)

	model := repl.NewModel(evaluator, cfg, bus.Publisher)
	programOptions := make([]tea.ProgramOption, 0, 1)
	if os.Getenv("BOBATEA_NO_ALT_SCREEN") != "1" {
		programOptions = append(programOptions, tea.WithAltScreen())
	}
	p := tea.NewProgram(model, programOptions...)
	timeline.RegisterUIForwarder(bus, p)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errs := make(chan error, 2)
	go func() { errs <- bus.Run(ctx) }()
	go func() {
		_, runErr := p.Run()
		cancel()
		errs <- runErr
	}()
	if runErr := <-errs; runErr != nil {
		log.Fatal(runErr)
	}
}
