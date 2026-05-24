package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	gochannel "github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-go-golems/bobatea/pkg/eventbus"
	"github.com/go-go-golems/bobatea/pkg/logutil"
	bobarepl "github.com/go-go-golems/bobatea/pkg/repl"
	"github.com/go-go-golems/bobatea/pkg/timeline"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	jsadapter "github.com/go-go-golems/go-go-goja/pkg/repl/adapters/bobatea"
	jsevaluator "github.com/go-go-golems/go-go-goja/pkg/repl/evaluators/javascript"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type tuiCommand struct {
	*cmds.CommandDescription
	factory *RuntimeFactory
	spec    *Spec
}

var _ cmds.BareCommand = (*tuiCommand)(nil)

type tuiSettings struct {
	Runtime   string `glazed:"runtime"`
	AltScreen bool   `glazed:"alt-screen"`
}

func newTUICommand(factory *RuntimeFactory, spec *Spec) cmds.Command {
	profile := commandRuntime(spec.Commands.Repl, firstRuntime(spec))
	return &tuiCommand{
		CommandDescription: cmds.NewCommandDescription(commandName(spec.Commands.Repl, "repl"),
			cmds.WithShort("Run an interactive TUI REPL for a generated xgoja runtime"),
			cmds.WithLong(`
TUI starts a Bubble Tea JavaScript REPL backed by a generated xgoja runtime
profile. The selected runtime profile controls which provider modules are
available through require().
`),
			cmds.WithFlags(
				fields.New("runtime", fields.TypeString,
					fields.WithDefault(profile),
					fields.WithHelp("Runtime profile to use")),
				fields.New("alt-screen", fields.TypeBool,
					fields.WithDefault(true),
					fields.WithHelp("Run the TUI in the terminal alt screen")),
			),
		),
		factory: factory,
		spec:    spec,
	}
}

func (c *tuiCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := tuiSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	return runTUI(ctx, c.factory, c.spec, settings.Runtime, settings.AltScreen)
}

func runTUI(ctx context.Context, factory *RuntimeFactory, spec *Spec, profile string, altScreen bool) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if factory == nil {
		return fmt.Errorf("runtime factory is required")
	}
	logutil.InitTUILoggingToDiscard(zerolog.ErrorLevel)

	adapter, err := newXGojaTUIEvaluator(ctx, factory, profile)
	if err != nil {
		return err
	}
	defer func() { _ = adapter.Close() }()

	cfg := bobarepl.DefaultConfig()
	name := "xgoja"
	if spec != nil && strings.TrimSpace(spec.Name) != "" {
		name = strings.TrimSpace(spec.Name)
	}
	cfg.Title = fmt.Sprintf("%s TUI (%s runtime)", name, profile)
	cfg.Placeholder = fmt.Sprintf("Runtime %s | Type JavaScript, then use alt+h for help", profile)
	cfg.Autocomplete.Enabled = true
	cfg.Autocomplete.FocusToggleKey = "ctrl+t"
	cfg.Autocomplete.TriggerKeys = []string{"tab"}
	cfg.Autocomplete.AcceptKeys = []string{"enter", "tab"}
	cfg.HelpBar.Enabled = true
	cfg.HelpDrawer.Enabled = true

	bus, err := newQuietInMemoryBus()
	if err != nil {
		return err
	}
	bobarepl.RegisterReplToTimelineTransformer(bus)

	model := bobarepl.NewModel(adapter, cfg, bus.Publisher)
	programOptions := make([]tea.ProgramOption, 0, 1)
	if altScreen {
		programOptions = append(programOptions, tea.WithAltScreen())
	}
	program := tea.NewProgram(model, programOptions...)
	timeline.RegisterUIForwarder(bus, program)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	group, groupCtx := errgroup.WithContext(runCtx)
	group.Go(func() error {
		return bus.Run(groupCtx)
	})
	group.Go(func() error {
		defer cancel()
		_, err := program.Run()
		return err
	})
	if err := group.Wait(); err != nil && err != context.Canceled {
		return err
	}
	return nil
}

type xgojaTUIEvaluator struct {
	runtimeCloser interface{ Close(context.Context) error }
	evaluator     *jsadapter.JavaScriptEvaluator
}

func newXGojaTUIEvaluator(ctx context.Context, factory *RuntimeFactory, profile string) (*xgojaTUIEvaluator, error) {
	rt, err := factory.NewRuntime(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("create runtime: %w", err)
	}
	evaluator, err := jsadapter.NewJavaScriptEvaluator(jsevaluator.Config{
		EnableModules: true,
		Runtime:       rt.VM,
	})
	if err != nil {
		_ = rt.Close(ctx)
		return nil, err
	}
	return &xgojaTUIEvaluator{runtimeCloser: rt, evaluator: evaluator}, nil
}

func (e *xgojaTUIEvaluator) EvaluateStream(ctx context.Context, code string, emit func(bobarepl.Event)) error {
	return e.evaluator.EvaluateStream(ctx, code, emit)
}

func (e *xgojaTUIEvaluator) GetPrompt() string { return e.evaluator.GetPrompt() }
func (e *xgojaTUIEvaluator) GetName() string   { return e.evaluator.GetName() }
func (e *xgojaTUIEvaluator) SupportsMultiline() bool {
	return e.evaluator.SupportsMultiline()
}
func (e *xgojaTUIEvaluator) GetFileExtension() string { return e.evaluator.GetFileExtension() }
func (e *xgojaTUIEvaluator) CompleteInput(ctx context.Context, req bobarepl.CompletionRequest) (bobarepl.CompletionResult, error) {
	return e.evaluator.CompleteInput(ctx, req)
}
func (e *xgojaTUIEvaluator) GetHelpBar(ctx context.Context, req bobarepl.HelpBarRequest) (bobarepl.HelpBarPayload, error) {
	return e.evaluator.GetHelpBar(ctx, req)
}
func (e *xgojaTUIEvaluator) GetHelpDrawer(ctx context.Context, req bobarepl.HelpDrawerRequest) (bobarepl.HelpDrawerDocument, error) {
	return e.evaluator.GetHelpDrawer(ctx, req)
}
func (e *xgojaTUIEvaluator) Close() error {
	if e == nil {
		return nil
	}
	if e.evaluator != nil {
		_ = e.evaluator.Close()
	}
	if e.runtimeCloser != nil {
		return e.runtimeCloser.Close(context.Background())
	}
	return nil
}

func newQuietInMemoryBus() (*eventbus.Bus, error) {
	logger := watermill.NopLogger{}
	pubsub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 1024}, logger)
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, err
	}
	return &eventbus.Bus{
		Router:     router,
		Publisher:  pubsub,
		Subscriber: pubsub,
	}, nil
}

var _ bobarepl.Evaluator = (*xgojaTUIEvaluator)(nil)
var _ bobarepl.InputCompleter = (*xgojaTUIEvaluator)(nil)
var _ bobarepl.HelpBarProvider = (*xgojaTUIEvaluator)(nil)
var _ bobarepl.HelpDrawerProvider = (*xgojaTUIEvaluator)(nil)
