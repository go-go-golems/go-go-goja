package main

import (
	"encoding/json"
	"fmt"

	"github.com/dop251/goja"
)

func main() {
	vm := goja.New()
	claims := map[string]any{"role": "admin", "nested": map[string]any{"level": "one"}}
	tenantIDs := []string{"o1", "o2"}
	obj := vm.NewObject()
	_ = obj.Set("claims", claims)
	_ = obj.Set("tenantIds", tenantIDs)
	if err := vm.Set("actor", obj); err != nil {
		panic(err)
	}
	if _, err := vm.RunString(`
		actor.claims.role = "mutated";
		actor.claims.extra = "new";
		actor.claims.nested.level = "two";
		actor.tenantIds[0] = "mutated-tenant";
		actor.tenantIds.push("o3");
	`); err != nil {
		panic(err)
	}
	out := map[string]any{"claims": claims, "tenantIDs": tenantIDs}
	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))
}
