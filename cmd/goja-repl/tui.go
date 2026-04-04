package main

import (
	"context"
	"fmt"
	"io"
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
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type tuiSettings struct {
	Profile   string `glazed:"profile"`
	SessionID string `glazed:"session-id"`
	AltScreen bool   `glazed:"alt-screen"`
}

type tuiCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*tuiCommand)(nil)

func newTUICommand(out io.Writer, opts *rootOptions) *tuiCommand {
	return &tuiCommand{
		CommandDescription: cmds.NewCommandDescription("tui",
			cmds.WithShort("Run the Bubble Tea REPL UI on top of replapi"),
			cmds.WithFlags(
				fields.New("profile", fields.TypeString, fields.WithDefault(string(replapi.ProfileInteractive)), fields.WithHelp("REPL profile: raw, interactive, persistent")),
				fields.New("session-id", fields.TypeString, fields.WithHelp("Existing persistent session id to open (requires --profile persistent)")),
				fields.New("alt-screen", fields.TypeBool, fields.WithDefault(true), fields.WithHelp("Run the TUI in the terminal alt screen")),
			),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *tuiCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := tuiSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}

	profile, err := parseTUIProfile(settings.Profile)
	if err != nil {
		return err
	}
	if settings.SessionID != "" && profile != replapi.ProfilePersistent {
		return errors.New("tui: --session-id requires --profile persistent")
	}

	logutil.InitTUILoggingToDiscard(zerolog.ErrorLevel)

	helpSystem, err := newSharedHelpSystem()
	if err != nil {
		return errors.Wrap(err, "load shared help system")
	}

	app, store, err := c.newAppWithOptions(appSupportOptions{
		profile:    profile,
		withStore:  profile == replapi.ProfilePersistent,
		helpSystem: helpSystem,
	})
	if err != nil {
		return err
	}
	if store != nil {
		defer func() { _ = store.Close() }()
	}

	sessionID := strings.TrimSpace(settings.SessionID)
	if sessionID == "" {
		requestedProfile := profile
		session, err := app.CreateSessionWithOptions(ctx, replapi.SessionOptions{Profile: &requestedProfile})
		if err != nil {
			return err
		}
		sessionID = session.ID
	}

	adapter, err := jsadapter.NewREPLAPIAdapter(app, sessionID)
	if err != nil {
		return err
	}
	defer func() { _ = adapter.Close() }()

	cfg := bobarepl.DefaultConfig()
	cfg.Title = fmt.Sprintf("goja-repl TUI (%s profile)", profile)
	cfg.Placeholder = fmt.Sprintf("Session %s | Type console.lo or fs.re, then use alt+h (drawer), ctrl+h (full help), ctrl+r (refresh)", sessionID)
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
	if settings.AltScreen {
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

	err = group.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func parseTUIProfile(raw string) (replapi.Profile, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(replapi.ProfileInteractive):
		return replapi.ProfileInteractive, nil
	case string(replapi.ProfileRaw):
		return replapi.ProfileRaw, nil
	case string(replapi.ProfilePersistent):
		return replapi.ProfilePersistent, nil
	default:
		return "", errors.Errorf("tui: unsupported profile %q", raw)
	}
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
