// Package fuzz contains coverage-guided fuzz tests for the go-go-goja REPL
// evaluation pipeline. Run with:
//
//	make fuzz            # 30s per target (default)
//	make fuzz FUZZTIME=5m
package fuzz

// Seed corpus organized by category. Add new seeds here when you find
// interesting edge cases — the mutator uses these as starting material.

// SeedsMinimal contains basic valid and near-valid inputs.
var SeedsMinimal = []string{
	"",
	" ",
	"\n",
	"\t",
	";",
	"1",
	"'hello'",
	"true",
	"false",
	"null",
	"undefined",
	"this",
	"{}",
	"[]",
	"()",
	"/.*/",
}

// SeedsDeclarations exercises the rewrite binding-capture path.
var SeedsDeclarations = []string{
	"const x = 1",
	"let y = 2",
	"var z = 3",
	"const a = 1, b = 2",
	"let a = 1, b = 2, c = 3",
	"function f() { return 1 }",
	"function add(a, b) { return a + b }",
	"class A {}",
	"class Point { constructor(x, y) { this.x = x; this.y = y } }",
	"class Foo { static bar() { return 42 } }",
	"const { a, b } = { a: 1, b: 2 }",
	"const [x, y] = [1, 2]",
	"const { a: renamed } = { a: 1 }",
	"const [...rest] = [1, 2, 3]",
}

// SeedsExpressions exercises last-expression capture in the rewrite.
var SeedsExpressions = []string{
	"1 + 2",
	"'hello'.toUpperCase()",
	"[1, 2, 3].map(x => x * 2)",
	"Math.max(1, 2, 3)",
	"JSON.stringify({ a: 1 })",
	"Object.keys({ a: 1, b: 2 }).join(',')",
	"new Array(3).fill(0)",
	"typeof undefined",
	"typeof null",
	"typeof 1",
	"typeof 's'",
	"typeof true",
	"typeof {}",
	"typeof function(){}",
	"void 0",
	"delete {}",
	"in 'a' ? 1 : 2",
	"'a' in { a: 1 }",
	"1 === 1",
	"1 !== 2",
}

// SeedsMixed exercises declaration + expression combinations.
var SeedsMixed = []string{
	"const x = 1; x",
	"const x = 1; x + 1",
	"let a = 1; let b = 2; a + b",
	"const x = [1, 2]; x.length",
	"const obj = { count: 0 }; obj.count = 5; obj.count",
	"function f() { return 42 }; f()",
	"class A { m() { return 1 } }; new A().m()",
	"var a = 1; var b = 2; a + b",
}

// SeedsAsync exercises promise/top-level-await paths.
var SeedsAsync = []string{
	"Promise.resolve(1)",
	"Promise.reject('err')",
	"new Promise(r => r(42))",
	"async function f() { return 1 }; f()",
	"async function f() { await Promise.resolve(2); return 3 }; f()",
	"await Promise.resolve(1)",
	"await 42",
}

// SeedsErrorInputs should produce errors but never panics.
var SeedsErrorInputs = []string{
	"throw new Error('test')",
	"throw 1",
	"throw 'string'",
	"throw null",
	"undefined.property",
	"null.toString()",
	"JSON.parse('{invalid')",
	"new (-1)",
	"const x = ;",
	"function f( {}",
	"return 1",
	"break",
	"continue",
	"yield 1",
}

// SeedsUnicode exercises non-ASCII inputs.
var SeedsUnicode = []string{
	"const 你好 = 'world'; 你好",
	"'\\x00\\x01\\x02'",
	"'\\uffff'",
	"`template ${1 + 2}`",
	"// comment\n1",
	"/* block comment */ 2",
	"/* multi\nline\ncomment */ 3",
	"String.fromCharCode(0, 0xFFFF, 0x10FFFF)",
	"'éñøṙ'",
	"'\\u0041'",
}

// SeedsDeepNesting exercises deeply nested structures.
var SeedsDeepNesting = []string{
	"((((1))))",
	"if (true) { if (true) { if (true) { 1 } } }",
	"const a = {a:{a:{a:1}}}; a",
	"const f = () => () => () => 1; f()()()",
	"try { throw 1 } catch(e) { e }",
	"try { try { throw 1 } catch(e) { throw e } } catch(e2) { e2 }",
	"for (let i = 0; i < 3; i++) { i }",
	"while (false) { }",
	"do { break } while (false)",
	"switch(1) { case 1: 1; break; default: 0 }",
	"label: for(;;) break label",
}

// SeedsObjectEdgeCases exercises runtime observation / introspection.
var SeedsObjectEdgeCases = []string{
	"Object.create(null)",
	"const obj = { get x() { return 1 } }; obj.x",
	"const key = 'a'; const obj = { [key]: 1 }; obj",
	"const s = Symbol('test'); const obj = { [s]: 1 }; obj[s]",
	"Object.defineProperty({}, 'x', { value: 1, writable: false })",
	"new Proxy({}, { get: () => 42 })",
	"Object.freeze({ a: 1 })",
	"Object.seal({ a: 1 })",
	"Object.assign({}, { a: 1 }, { b: 2 })",
	"const obj = { a: 1 }; Object.keys(obj)",
	"const obj = { a: 1 }; Object.values(obj)",
	"const obj = { a: 1 }; Object.entries(obj)",
	"new Map([[1, 'a'], [2, 'b']])",
	"new Set([1, 2, 3])",
	"new WeakMap()",
	"new WeakSet()",
	"new Date()",
	"new Error('test')",
	"/regex/g",
	"new RegExp('pattern', 'i')",
}

// SeedsTypeCoercion exercises JS type coercion edge cases.
var SeedsTypeCoercion = []string{
	"[] + []",
	"[] + {}",
	"{} + []",
	"+true",
	"+false",
	"!!''",
	"!!0",
	"!!null",
	"!!undefined",
	"'1' - 1",
	"'1' + 1",
	"null == undefined",
	"null === undefined",
	"NaN === NaN",
	"isNaN(NaN)",
	"isFinite(Infinity)",
	"typeof NaN",
}

// SeedsStressSerialization produces values that stress JSON marshaling.
var SeedsStressSerialization = []string{
	"new Array(100).fill(0)",
	"new Array(1000).fill(null)",
	"JSON.parse('{\"a\":1}')",
	"JSON.parse('[1,2,3]')",
	"JSON.stringify(null)",
	"Array.from({ length: 10 }, (_, i) => i)",
	"Object.fromEntries([['a', 1], ['b', 2]])",
}

// SeedsConsoleCapture exercises console interception.
var SeedsConsoleCapture = []string{
	"console.log('hello')",
	"console.info('info')",
	"console.warn('warn')",
	"console.error('error')",
	"console.debug('debug')",
	"console.log(1, 'two', true, null, undefined)",
	"console.log({ a: 1 })",
	"console.log([1, 2, 3])",
}

// AllSeeds returns every seed from every category, deduplicated.
func AllSeeds() []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range [][]string{
		SeedsMinimal,
		SeedsDeclarations,
		SeedsExpressions,
		SeedsMixed,
		SeedsAsync,
		SeedsErrorInputs,
		SeedsUnicode,
		SeedsDeepNesting,
		SeedsObjectEdgeCases,
		SeedsTypeCoercion,
		SeedsStressSerialization,
		SeedsConsoleCapture,
	} {
		for _, v := range s {
			if _, ok := seen[v]; !ok {
				seen[v] = struct{}{}
				out = append(out, v)
			}
		}
	}
	return out
}

// SequenceSeeds returns pairs of (first, second) for session-sequence fuzzing.
func SequenceSeeds() [][2]string {
	return [][2]string{
		{"const x = 1", "x + 1"},
		{"let y = 'hello'", "y + ' world'"},
		{"function f() { return 42 }", "f()"},
		{"var z = null", "z"},
		{"const arr = [1, 2, 3]", "arr.map(x => x * 2)"},
		{"const obj = { count: 0 }", "obj.count = 5"},
		{"", "1 + 1"},
		{"1 + 1", ""},
		{"const x = 1", "const x = 2"},
		{"let a = 1", "a = 99"},
		{"throw new Error('x')", "1 + 1"},
		{"const f = (x) => x * 2", "f(21)"},
		{"class A { m() { return 1 } }", "new A().m()"},
		{"const { a, b } = { a: 1, b: 2 }", "a + b"},
		{"const [x, y] = [10, 20]", "x * y"},
		{"Promise.resolve(1)", "1 + 1"},
		{"console.log('hi')", "42"},
	}
}

// PersistenceSeeds returns triples of (seed, restore, continuation) for persistence fuzzing.
func PersistenceSeeds() [][3]string {
	return [][3]string{
		{"const x = 1", "x + 1", "x * 2"},
		{"function f() { return 42 }", "f()", "f() + 1"},
		{"let a = [1, 2]; a.push(3)", "a.length", "a[0]"},
		{"const obj = { count: 0 }; obj.count = 5", "obj.count", "obj.count + 1"},
		{"class P { constructor(x) { this.x = x } }", "new P(1).x", "new P(2).x"},
		{"const x = 1", "", "x"},
		{"const f = x => x * x", "f(3)", "f(4)"},
	}
}
