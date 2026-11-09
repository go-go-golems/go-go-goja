package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/resultcontent"
)

type consoleLook struct{}

func (consoleLook) ResultContent(context.Context) ([]resultcontent.Content, error) {
	panic("console must not render content")
}

func (consoleLook) ResultPlaceholder() string { return "look console" }

func TestConsoleShowsContentPlaceholders(t *testing.T) {
	vm := goja.New()
	var logged []string
	consoleObj := vm.NewObject()
	record := func(call goja.FunctionCall) goja.Value {
		parts := make([]string, 0, len(call.Arguments))
		for _, arg := range call.Arguments {
			parts = append(parts, arg.String())
		}
		logged = append(logged, strings.Join(parts, " "))
		return goja.Undefined()
	}
	for _, name := range []string{"log", "error"} {
		if err := consoleObj.Set(name, record); err != nil {
			t.Fatal(err)
		}
	}
	if err := vm.Set("console", consoleObj); err != nil {
		t.Fatal(err)
	}
	if err := vm.Set("look", func() consoleLook { return consoleLook{} }); err != nil {
		t.Fatal(err)
	}
	if err := installConsoleContentPlaceholders(vm); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, err := vm.RunString(`console.log("img", look(), 1); console.error({nested: look()})`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(logged) != 2 || logged[0] != "img [content: look console] 1" || logged[1] != "[object Object]" {
		t.Fatalf("logged = %#v", logged)
	}
}

func TestConsoleArgsWithContentPlaceholdersKeepsPlainArgs(t *testing.T) {
	vm := goja.New()
	args := []goja.Value{vm.ToValue("a"), vm.ToValue(1)}
	if got := ConsoleArgsWithContentPlaceholders(vm, args); &got[0] != &args[0] {
		t.Fatal("plain args were copied")
	}
}

func TestRuntimeConsoleWrapperPropagatesErrors(t *testing.T) {
	vm := goja.New()
	consoleObj := vm.NewObject()
	if err := consoleObj.Set("log", func(goja.FunctionCall) goja.Value { panic(vm.NewTypeError("printer failed")) }); err != nil {
		t.Fatal(err)
	}
	if err := vm.Set("console", consoleObj); err != nil {
		t.Fatal(err)
	}
	if err := installConsoleContentPlaceholders(vm); err != nil {
		t.Fatal(err)
	}
	v, err := vm.RunString(`try { console.log("x"); "no error" } catch (e) { e instanceof TypeError ? e.message : "wrong type" }`)
	if err != nil || v.String() != "printer failed" {
		t.Fatalf("error propagation: %v %v", v, err)
	}
}
