package sandbox_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/go-go-golems/go-go-goja/engine"
	bot "github.com/go-go-golems/go-go-goja/modules/sandbox"
	hostsandbox "github.com/go-go-golems/go-go-goja/pkg/sandbox"
)

func TestSandboxRegistrarExposesBotAndStore(t *testing.T) {
	scriptPath := writeBotScript(t, `
		const { defineBot } = require("sandbox")
		module.exports = defineBot(({ command, event, configure }) => {
			configure({ name: "demo-bot", tier: "gold" })
			command("ping", (ctx) => {
				const current = ctx.store.get("hits", 0)
				ctx.store.set("hits", current + 1)
				ctx.reply(`+"`pong:${current}`"+`)
				return current
			})
			event("ready", (ctx) => {
				ctx.store.set("ready", true)
				return "ready"
			})
		})
	`)

	factory, err := engine.NewBuilder(
		engine.WithModuleRootsFromScript(scriptPath, engine.DefaultModuleRootsOptions()),
	).
		WithModules(engine.DefaultRegistryModules()).
		WithRuntimeModuleRegistrars(hostsandbox.NewRegistrar(hostsandbox.Config{})).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() {
		if err := rt.Close(context.Background()); err != nil {
			t.Fatalf("close runtime: %v", err)
		}
	}()

	stateValue, ok := rt.Value(bot.RuntimeStateContextKey)
	if !ok {
		t.Fatalf("runtime sandbox state missing")
	}
	state, ok := stateValue.(*bot.RuntimeState)
	if !ok {
		t.Fatalf("runtime sandbox state = %T, want *RuntimeState", stateValue)
	}

	botValue, err := rt.Require.Require(scriptPath)
	if err != nil {
		t.Fatalf("require bot script: %v", err)
	}

	handle, err := bot.CompileBot(rt.VM, botValue)
	if err != nil {
		t.Fatalf("compile bot: %v", err)
	}

	desc, err := handle.Describe(context.Background())
	if err != nil {
		t.Fatalf("describe bot: %v", err)
	}
	if got := desc["kind"]; got != "sandbox.bot" {
		t.Fatalf("kind = %#v, want sandbox.bot", got)
	}
	metadata, ok := desc["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata = %#v", desc["metadata"])
	}
	if got := metadata["name"]; got != "demo-bot" {
		t.Fatalf("metadata.name = %#v, want demo-bot", got)
	}

	var replies []string
	replyMu := sync.Mutex{}
	replyFn := func(_ context.Context, value any) error {
		replyMu.Lock()
		defer replyMu.Unlock()
		replies = append(replies, fmt.Sprint(value))
		return nil
	}

	result, err := handle.DispatchCommand(context.Background(), bot.DispatchRequest{
		Name:  "ping",
		Reply: replyFn,
	})
	if err != nil {
		t.Fatalf("dispatch command: %v", err)
	}
	if got := fmt.Sprint(result); got != "0" {
		t.Fatalf("dispatch result = %s, want 0", got)
	}
	if len(replies) != 1 || replies[0] != "pong:0" {
		t.Fatalf("replies = %#v, want [pong:0]", replies)
	}

	result, err = handle.DispatchCommand(context.Background(), bot.DispatchRequest{
		Name:  "ping",
		Reply: replyFn,
	})
	if err != nil {
		t.Fatalf("dispatch command second time: %v", err)
	}
	if got := fmt.Sprint(result); got != "1" {
		t.Fatalf("dispatch result second time = %s, want 1", got)
	}
	if len(replies) != 2 || replies[1] != "pong:1" {
		t.Fatalf("replies after second dispatch = %#v, want [..., pong:1]", replies)
	}

	eventResult, err := handle.DispatchEvent(context.Background(), bot.DispatchRequest{Name: "ready"})
	if err != nil {
		t.Fatalf("dispatch event: %v", err)
	}
	if got := fmt.Sprint(eventResult); got != "[ready]" {
		t.Fatalf("dispatch event result = %s, want [ready]", got)
	}

	if got := state.Store().Get("hits", nil); fmt.Sprint(got) != "2" {
		t.Fatalf("store hits = %#v, want 2", got)
	}
	if got := state.Store().Get("ready", nil); got != true {
		t.Fatalf("store ready = %#v, want true", got)
	}
}

func TestSandboxStateIsRuntimeLocal(t *testing.T) {
	botScript := `
		const { defineBot } = require("sandbox")
		module.exports = defineBot(({ command }) => {
			command("count", (ctx) => {
				const current = ctx.store.get("count", 0)
				ctx.store.set("count", current + 1)
				return current
			})
		})
	`
	scriptPath := writeBotScript(t, botScript)

	factory, err := engine.NewBuilder(
		engine.WithModuleRootsFromScript(scriptPath, engine.DefaultModuleRootsOptions()),
	).
		WithModules(engine.DefaultRegistryModules()).
		WithRuntimeModuleRegistrars(hostsandbox.NewRegistrar(hostsandbox.Config{})).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt1, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime 1: %v", err)
	}
	defer func() { _ = rt1.Close(context.Background()) }()

	rt2, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime 2: %v", err)
	}
	defer func() { _ = rt2.Close(context.Background()) }()

	bot1, err := rt1.Require.Require(scriptPath)
	if err != nil {
		t.Fatalf("runtime 1 require bot script: %v", err)
	}
	handle1, err := bot.CompileBot(rt1.VM, bot1)
	if err != nil {
		t.Fatalf("compile runtime 1 bot: %v", err)
	}

	bot2, err := rt2.Require.Require(scriptPath)
	if err != nil {
		t.Fatalf("runtime 2 require bot script: %v", err)
	}
	handle2, err := bot.CompileBot(rt2.VM, bot2)
	if err != nil {
		t.Fatalf("compile runtime 2 bot: %v", err)
	}

	first, err := handle1.DispatchCommand(context.Background(), bot.DispatchRequest{Name: "count"})
	if err != nil {
		t.Fatalf("runtime 1 first dispatch: %v", err)
	}
	if got := fmt.Sprint(first); got != "0" {
		t.Fatalf("runtime 1 first dispatch = %s, want 0", got)
	}
	second, err := handle2.DispatchCommand(context.Background(), bot.DispatchRequest{Name: "count"})
	if err != nil {
		t.Fatalf("runtime 2 first dispatch: %v", err)
	}
	if got := fmt.Sprint(second); got != "0" {
		t.Fatalf("runtime 2 first dispatch = %s, want 0", got)
	}

	firstAgain, err := handle1.DispatchCommand(context.Background(), bot.DispatchRequest{Name: "count"})
	if err != nil {
		t.Fatalf("runtime 1 second dispatch: %v", err)
	}
	if got := fmt.Sprint(firstAgain); got != "1" {
		t.Fatalf("runtime 1 second dispatch = %s, want 1", got)
	}
}

func writeBotScript(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "bot.js")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write bot script: %v", err)
	}
	return path
}
