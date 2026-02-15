//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"strings"

	"github.com/dop251/goja"
)

func main() {
	vm := goja.New()

	vm.Set("captureStack", func(call goja.FunctionCall) goja.Value {
		frames := vm.CaptureCallStack(0, nil)
		fmt.Println("== CaptureCallStack (inside JS call) ==")
		for i, f := range frames {
			pos := f.Position()
			fmt.Printf("#%d %s:%d:%d\n", i, pos.Filename, pos.Line, pos.Column)
		}
		return goja.Undefined()
	})

	src := `
const symCustom = Symbol("custom");

const config = {
  apiUrl: "https://api.example.com/v3",
  timeout: 3000,
  retries: 5,
  debug: false,
  [Symbol.iterator]: function* iterator() { yield 1; },
  [Symbol.toPrimitive]: function() { return "cfg"; },
  [symCustom]: "meta"
};

class Animal {
  constructor(name) {
    this.name = name;
    this.alive = true;
    this.energy = 0;
  }

  eat(food) {
    if (!this.alive) {
      throw new Error("dead");
    }
    this.energy += food.calories;
    return this;
  }

  sleep() {
    return "zzz";
  }
}

class Dog extends Animal {
  constructor(name) {
    super(name);
    this.breed = "lab";
  }

  bark() {
    const sound = this.breed === "husky" ? "awoo" : "woof";
    return sound;
  }

  fetch(item) {
    return this.eat(item);
  }
}

function main() {
  const rex = new Dog("Rex");
  captureStack();
  return rex;
}

globalThis.main = main;
globalThis.config = config;
globalThis.Dog = Dog;
globalThis.Animal = Animal;
globalThis.symCustom = symCustom;
`

	if _, err := vm.RunString(src); err != nil {
		panic(err)
	}

	mainFn, ok := goja.AssertFunction(vm.Get("main"))
	if !ok {
		panic("main is not callable")
	}

	v, err := mainFn(goja.Undefined())
	if err != nil {
		panic(err)
	}

	rex := v.ToObject(vm)
	fmt.Println("== Dog instance own keys ==")
	fmt.Printf("GetOwnPropertyNames=%v\n", rex.GetOwnPropertyNames())
	fmt.Printf("Keys=%v\n", rex.Keys())
	fmt.Printf("Symbols=%v\n", symbolStrings(rex.Symbols()))

	fmt.Println("== Dog instance prototype chain ==")
	for i, p := 0, rex.Prototype(); p != nil; i, p = i+1, p.Prototype() {
		ctor := p.Get("constructor")
		name := "<anon>"
		if ctorObj, ok := ctor.(*goja.Object); ok {
			if n := ctorObj.Get("name"); n != nil {
				name = n.String()
			}
		}
		fmt.Printf("level=%d ctor=%s ownNames=%v\n", i, name, p.GetOwnPropertyNames())
	}

	configObj := vm.Get("config").ToObject(vm)
	fmt.Println("== config keys and symbols ==")
	fmt.Printf("GetOwnPropertyNames=%v\n", configObj.GetOwnPropertyNames())
	fmt.Printf("Symbols=%v\n", symbolStrings(configObj.Symbols()))

	for _, key := range []string{"apiUrl", "timeout", "retries", "debug"} {
		desc, err := vm.RunString(fmt.Sprintf("JSON.stringify(Object.getOwnPropertyDescriptor(config, %q))", key))
		if err != nil {
			panic(err)
		}
		fmt.Printf("descriptor[%s]=%s\n", key, desc.String())
	}

	fmt.Println("== thrown error stack string ==")
	_, err = vm.RunString(`new Dog("Rex").fetch()`)
	if err != nil {
		if ex, ok := err.(*goja.Exception); ok {
			fmt.Println(ex.String())
			lines := strings.Split(ex.String(), "\n")
			if len(lines) > 0 {
				fmt.Printf("errorMessage=%s\n", lines[0])
			}
		} else {
			fmt.Printf("non-exception error=%v\n", err)
		}
	}
}

func symbolStrings(syms []*goja.Symbol) []string {
	ret := make([]string, 0, len(syms))
	for _, s := range syms {
		if s == nil {
			continue
		}
		ret = append(ret, s.String())
	}
	return ret
}
