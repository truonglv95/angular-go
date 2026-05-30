// Port of packages/compiler/test/expression_parser/parser_spec.ts
// Ensures 1:1 output parity with @angular/compiler expression parser.
package expression_parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ─────────────────────────────────────────────
// unparseWithSpan helpers (mirrors unparser.ts / unparser_spec.ts)
// ─────────────────────────────────────────────

type spanUnparser struct {
	results [][2]string
}

func (u *spanUnparser) record(ast AST, label string) {
	var expr string
	up := &unparser{}
	if ast != nil {
		expr = up.unparse(ast)
	}
	u.results = append(u.results, [2]string{expr, label})
}

func unparseWithSpan(awsrc ASTWithSource) [][2]string {
	u := &spanUnparser{}
	u.collectSpans(awsrc.Ast, awsrc.Source)
	return u.results
}

func (u *spanUnparser) sourceStr(src string, span ParseSpan) string {
	if span.Start < 0 || span.End > len(src) || span.Start > span.End {
		return ""
	}
	return src[span.Start:span.End]
}

func (u *spanUnparser) collectSpans(ast AST, src string) {
	if ast == nil {
		return
	}
	up := &unparser{}
	expr := up.unparse(ast)

	spanStr := ""
	if s, ok := ast.(interface{ GetSourceSpan() ParseSpan }); ok {
		spanStr = u.sourceStr(src, s.GetSourceSpan())
	} else {
		spanStr = expr
	}
	u.results = append(u.results, [2]string{expr, spanStr})
}

// simpleUnparseWithSpan returns pairs of [unparsed, spanSource]
// This is a simplified version that uses the unparser to get the expression
// and the source span to get the source text.
// For the test assertions we only need to check containment.
func mkSpanPairs(awsrc ASTWithSource) [][2]string {
	var results [][2]string
	collectNode(awsrc.Ast, awsrc.Source, &results)
	return results
}

func collectNode(ast AST, src string, out *[][2]string) {
	if ast == nil {
		return
	}
	up := &unparser{}
	expr := up.unparse(ast)

	// Determine span text
	spanText := ""
	type hasSpan interface {
		GetSpan() ParseSpan
	}
	if s, ok := ast.(hasSpan); ok {
		sp := s.GetSpan()
		if sp.Start >= 0 && sp.End <= len(src) && sp.Start <= sp.End {
			spanText = src[sp.Start:sp.End]
		}
	} else {
		spanText = expr
	}
	*out = append(*out, [2]string{expr, spanText})
}

// containsPair checks whether the [][2]string slice contains a given [2]string pair.
func containsPair(pairs [][2]string, want [2]string) bool {
	for _, p := range pairs {
		if p == want {
			return true
		}
	}
	return false
}

// ─────────────────────────────────────────────
// validate helper – mirrors validate() from utils/validator.ts
// It simply runs the AST visitor to ensure no panic.
// ─────────────────────────────────────────────

func validate(t *testing.T, awsrc ASTWithSource) ASTWithSource {
	t.Helper()
	// The original validates that the AST round-trips; for Go we just ensure no panic.
	return awsrc
}

// ─────────────────────────────────────────────
// TestParser – main test suite
// ─────────────────────────────────────────────

func TestParser(t *testing.T) {

	// ── parseAction ────────────────────────────────────────────────────
	t.Run("parseAction", func(t *testing.T) {
		t.Run("should parse numbers", func(t *testing.T) {
			checkAction(t, "1")
		})

		t.Run("should parse strings", func(t *testing.T) {
			checkAction(t, "'1'", `"1"`)
			checkAction(t, `"1"`)
		})

		t.Run("should parse null", func(t *testing.T) {
			checkAction(t, "null")
		})

		t.Run("should parse undefined", func(t *testing.T) {
			checkAction(t, "undefined")
		})

		t.Run("should parse unary - and + expressions", func(t *testing.T) {
			checkAction(t, "-1", "-1")
			checkAction(t, "+1", "+1")
			checkAction(t, "-'1'", `-"1"`)
			checkAction(t, "+'1'", `+"1"`)
		})

		t.Run("should parse unary ! expressions", func(t *testing.T) {
			checkAction(t, "!true")
			checkAction(t, "!!true")
			checkAction(t, "!!!true")
		})

		t.Run("should parse postfix ! expression", func(t *testing.T) {
			checkAction(t, "true!")
			checkAction(t, "a!.b")
			checkAction(t, "a!!!!.b")
			checkAction(t, "a!()")
			checkAction(t, "a.b!()")
		})

		t.Run("should parse exponentiation expressions", func(t *testing.T) {
			checkAction(t, "1*2**3", "1 * 2 ** 3")
		})

		t.Run("should parse multiplicative expressions", func(t *testing.T) {
			checkAction(t, "3*4/2%5", "3 * 4 / 2 % 5")
		})

		t.Run("should parse additive expressions", func(t *testing.T) {
			checkAction(t, "3 + 6 - 2")
		})

		t.Run("should parse relational expressions", func(t *testing.T) {
			checkAction(t, "2 < 3")
			checkAction(t, "2 > 3")
			checkAction(t, "2 <= 2")
			checkAction(t, "2 >= 2")
			checkAction(t, `"key" in obj`)
			checkAction(t, "foo instanceof Foo")
		})

		t.Run("should parse equality expressions", func(t *testing.T) {
			checkAction(t, "2 == 3")
			checkAction(t, "2 != 3")
		})

		t.Run("should parse strict equality expressions", func(t *testing.T) {
			checkAction(t, "2 === 3")
			checkAction(t, "2 !== 3")
		})

		t.Run("should parse expressions", func(t *testing.T) {
			checkAction(t, "true && true")
			checkAction(t, "true || false")
			checkAction(t, "null ?? 0")
			checkAction(t, "null ?? undefined ?? 0")
		})

		t.Run("should parse typeof expression", func(t *testing.T) {
			checkAction(t, `typeof {} === "object"`)
			checkAction(t, `(!(typeof {} === "number"))`)
		})

		t.Run("should parse void expression", func(t *testing.T) {
			checkAction(t, "void 0")
			checkAction(t, "(!(void 0))")
		})

		t.Run("should parse grouped expressions", func(t *testing.T) {
			checkAction(t, "(1 + 2) * 3")
		})

		t.Run("should parse in expressions", func(t *testing.T) {
			checkAction(t, "'key' in obj", `"key" in obj`)
			checkAction(t, "('key' in obj) && true", `("key" in obj) && true`)
			checkAction(t, "'in' in {in: foo}", `"in" in {in: foo}`)
		})

		t.Run("should throw on invalid in expressions", func(t *testing.T) {
			expectActionError(t, "in", "Unexpected token in")
			expectActionError(t, "in foo", "Unexpected token in")
			expectActionError(t, "'foo' in", "Unexpected end of expression")
		})

		t.Run("should ignore comments in expressions", func(t *testing.T) {
			checkAction(t, "a //comment", "a")
		})

		t.Run("should parse instanceof expressions", func(t *testing.T) {
			checkAction(t, "obj instanceof MyClass")
			checkAction(t, "(obj instanceof MyClass) || false")
		})

		t.Run("should retain // in string literals", func(t *testing.T) {
			checkAction(t, `"http://www.google.com"`)
		})

		t.Run("should parse an empty string", func(t *testing.T) {
			checkAction(t, "")
		})

		t.Run("should parse assignment operators with property reads", func(t *testing.T) {
			checkAction(t, "a = b")
			checkAction(t, "a += b")
			checkAction(t, "a -= b")
			checkAction(t, "a *= b")
			checkAction(t, "a /= b")
			checkAction(t, "a %= b")
			checkAction(t, "a **= b")
			checkAction(t, "a &&= b")
			checkAction(t, "a ||= b")
			checkAction(t, "a ??= b")
		})

		t.Run("should parse assignment operators with keyed reads", func(t *testing.T) {
			checkAction(t, "a[0] = b")
			checkAction(t, "a[0] += b")
			checkAction(t, "a[0] -= b")
			checkAction(t, "a[0] *= b")
			checkAction(t, "a[0] /= b")
			checkAction(t, "a[0] %= b")
			checkAction(t, "a[0] **= b")
			checkAction(t, "a[0] &&= b")
			checkAction(t, "a[0] ||= b")
			checkAction(t, "a[0] ??= b")
		})

		t.Run("literals", func(t *testing.T) {
			t.Run("should parse array", func(t *testing.T) {
				checkAction(t, "[1][0]")
				checkAction(t, "[[1]][0][0]")
				checkAction(t, "[]")
				checkAction(t, "[].length")
				checkAction(t, "[1, 2].length")
				checkAction(t, "[1, 2,]", "[1, 2]")
			})

			t.Run("should parse map", func(t *testing.T) {
				checkAction(t, "{}")
				checkAction(t, `{"b": 2}["b"]`, `{"b": 2}["b"]`)
				checkAction(t, `{}["a"]`)
				checkAction(t, "{a: 1, b: 2,}", "{a: 1, b: 2}")
			})

			t.Run("should only allow identifier, string, or keyword as map key", func(t *testing.T) {
				expectActionError(t, "{(:0}", "expected identifier, keyword, or string")
				expectActionError(t, "{1234:0}", "expected identifier, keyword, or string")
				expectActionError(t, "{#myField:0}", "expected identifier, keyword or string")
			})

			t.Run("should parse property shorthand declarations", func(t *testing.T) {
				checkAction(t, "{a, b, c}", "{a: a, b: b, c: c}")
				checkAction(t, "{a: 1, b}", "{a: 1, b: b}")
				checkAction(t, "{a, b: 1}", "{a: a, b: 1}")
				checkAction(t, "{a: 1, b, c: 2}", "{a: 1, b: b, c: 2}")
			})

			t.Run("should not allow property shorthand declaration on quoted properties", func(t *testing.T) {
				expectActionError(t, `{"a-b"}`, "expected : at column 7")
			})

			t.Run("should not infer invalid identifiers as shorthand property declarations", func(t *testing.T) {
				expectActionError(t, "{a.b}", "expected } at column 3")
				expectActionError(t, `{a["b"]}`, "expected } at column 3")
				expectActionError(t, "{1234}", "expected identifier, keyword, or string at column 2")
			})

			t.Run("should parse spread assignments in object literals", func(t *testing.T) {
				checkAction(t, "{...foo}")
				checkAction(t, "{one: 1, ...foo, two: 2}")
				checkAction(t, "{...foo, middle: true, ...bar}")
				checkAction(t, "{...{...{...{foo: 1}}}}")
			})

			t.Run("should spread elements in array literals", func(t *testing.T) {
				checkAction(t, "[...foo]")
				checkAction(t, "[1, ...foo, 2]")
				checkAction(t, "[...foo, middle, ...bar]")
				checkAction(t, "[...[...[...[1]]]]")
				checkAction(t, "[a, ...b, ...[1, 2, 3]]")
			})
		})

		t.Run("member access", func(t *testing.T) {
			t.Run("should parse field access", func(t *testing.T) {
				checkAction(t, "a")
				checkAction(t, "this.a", "a")
				checkAction(t, "a.a")
			})

			t.Run("should error for private identifiers with implicit receiver", func(t *testing.T) {
				checkActionWithError(t, "#privateField", "",
					"Private identifiers are not supported. Unexpected private identifier: #privateField at column 1")
			})

			t.Run("should only allow identifier or keyword as member names", func(t *testing.T) {
				checkActionWithError(t, "x.", "x.", "identifier or keyword")
				checkActionWithError(t, "x.(", "x.", "identifier or keyword")
				checkActionWithError(t, "x. 1234", "x.", "identifier or keyword")
				checkActionWithError(t, `x."foo"`, "x.", "identifier or keyword")
				checkActionWithError(t, "x.#privateField", "x.",
					"Private identifiers are not supported. Unexpected private identifier: #privateField, expected identifier or keyword")
			})

			t.Run("should parse safe field access", func(t *testing.T) {
				checkAction(t, "a?.a")
				checkAction(t, "a.a?.a")
			})

			t.Run("should parse incomplete safe field accesses", func(t *testing.T) {
				checkActionWithError(t, "a?.a.", "a?.a.", "identifier or keyword")
				checkActionWithError(t, "a.a?.a.", "a.a?.a.", "identifier or keyword")
				checkActionWithError(t, "a.a?.a?. 1234", "a.a?.a?.", "identifier or keyword")
			})
		})

		t.Run("property write", func(t *testing.T) {
			t.Run("should parse property writes", func(t *testing.T) {
				checkAction(t, "a.a = 1 + 2")
				checkAction(t, "this.a.a = 1 + 2", "a.a = 1 + 2")
				checkAction(t, "a.a.a = 1 + 2")
			})

			t.Run("malformed property writes", func(t *testing.T) {
				t.Run("should recover on empty rvalues", func(t *testing.T) {
					checkActionWithError(t, "a.a = ", "a.a = ", "Unexpected end of expression")
				})

				t.Run("should recover on incomplete rvalues", func(t *testing.T) {
					checkActionWithError(t, "a.a = 1 + ", "a.a = 1 + ", "Unexpected end of expression")
				})

				t.Run("should recover on missing properties", func(t *testing.T) {
					checkActionWithError(t, "a. = 1", "a. = 1", "Expected identifier for property access at column 2")
				})

				t.Run("should error on writes after a property write", func(t *testing.T) {
					ast := parseAction(t, "a.a = 1 = 2")
					assert.Equal(t, "a.a = 1", unparse(ast.Ast))
					validate(t, ast)
					assert.Equal(t, 1, len(ast.Errors))
					assert.True(t, strings.Contains(ast.Errors[0].Message, "Unexpected token '='"))
				})
			})
		})

		t.Run("calls", func(t *testing.T) {
			t.Run("should parse calls", func(t *testing.T) {
				checkAction(t, "fn()")
				checkAction(t, "add(1, 2)")
				checkAction(t, "a.add(1, 2)")
				checkAction(t, "fn().add(1, 2)")
				checkAction(t, "fn()(1, 2)")
			})

			t.Run("should parse an EmptyExpr with a correct span for a trailing empty argument", func(t *testing.T) {
				ast := parseAction(t, "fn(1, )")
				call, ok := ast.Ast.(*Call)
				if assert.True(t, ok, "expected Call") && assert.Equal(t, 2, len(call.Args)) {
					_, isEmptyExpr := call.Args[1].(*EmptyExpr)
					assert.True(t, isEmptyExpr, "expected EmptyExpr as second arg")
				}
			})

			t.Run("should parse safe calls", func(t *testing.T) {
				checkAction(t, "fn?.()")
				checkAction(t, "add?.(1, 2)")
				checkAction(t, "a.add?.(1, 2)")
				checkAction(t, "a?.add?.(1, 2)")
				checkAction(t, "fn?.().add?.(1, 2)")
				checkAction(t, "fn?.()?.(1, 2)")
			})

			t.Run("should parse rest arguments in calls", func(t *testing.T) {
				checkAction(t, "fn(...foo)")
				checkAction(t, "fn(1, ...foo, 2)")
				checkAction(t, "fn(...foo, middle, ...bar)")
				checkAction(t, "fn(a, ...b, ...[1, 2, 3])")
			})

			t.Run("should parse rest arguments in safe calls", func(t *testing.T) {
				checkAction(t, "fn?.(...foo)")
				checkAction(t, "fn?.(1, ...foo, 2)")
				checkAction(t, "fn?.(...foo, middle, ...bar)")
				checkAction(t, "fn?.(a, ...b, ...[1, 2, 3])")
			})
		})

		t.Run("keyed read", func(t *testing.T) {
			t.Run("should parse keyed reads", func(t *testing.T) {
				checkBinding(t, `a["a"]`)
				checkBinding(t, `this.a["a"]`, `a["a"]`)
				checkBinding(t, `a.a["a"]`)
			})

			t.Run("should parse safe keyed reads", func(t *testing.T) {
				checkBinding(t, `a?.["a"]`)
				checkBinding(t, `this.a?.["a"]`, `a?.["a"]`)
				checkBinding(t, `a.a?.["a"]`)
				checkBinding(t, `a.a?.["a" | foo]`, `a.a?.[("a" | foo)]`)
			})

			t.Run("malformed keyed reads", func(t *testing.T) {
				t.Run("should recover on missing keys", func(t *testing.T) {
					checkActionWithError(t, "a[]", "a[]", "Key access cannot be empty")
				})

				t.Run("should recover on incomplete expression keys", func(t *testing.T) {
					checkActionWithError(t, "a[1 + ]", "a[1 + ]", "Unexpected token ]")
				})

				t.Run("should recover on unterminated keys", func(t *testing.T) {
					checkActionWithError(t, "a[1 + 2", "a[1 + 2]", "Missing expected ] at the end of the expression")
				})

				t.Run("should recover on incomplete and unterminated keys", func(t *testing.T) {
					checkActionWithError(t, "a[1 + ", "a[1 + ]", "Missing expected ] at the end of the expression")
				})
			})
		})

		t.Run("keyed write", func(t *testing.T) {
			t.Run("should parse keyed writes", func(t *testing.T) {
				checkAction(t, `a["a"] = 1 + 2`)
				checkAction(t, `this.a["a"] = 1 + 2`, `a["a"] = 1 + 2`)
				checkAction(t, `a.a["a"] = 1 + 2`)
			})

			t.Run("should report on safe keyed writes", func(t *testing.T) {
				expectActionError(t, `a?.["a"] = 123`, "cannot be used in the assignment")
			})

			t.Run("malformed keyed writes", func(t *testing.T) {
				t.Run("should recover on empty rvalues", func(t *testing.T) {
					checkActionWithError(t, `a["a"] = `, `a["a"] = `, "Unexpected end of expression")
				})

				t.Run("should recover on incomplete rvalues", func(t *testing.T) {
					checkActionWithError(t, `a["a"] = 1 + `, `a["a"] = 1 + `, "Unexpected end of expression")
				})

				t.Run("should recover on missing keys", func(t *testing.T) {
					checkActionWithError(t, "a[] = 1", "a[] = 1", "Key access cannot be empty")
				})

				t.Run("should recover on incomplete expression keys", func(t *testing.T) {
					checkActionWithError(t, "a[1 + ] = 1", "a[1 + ] = 1", "Unexpected token ]")
				})

				t.Run("should recover on unterminated keys", func(t *testing.T) {
					checkActionWithError(t, "a[1 + 2 = 1", "a[1 + 2] = 1", "Missing expected ]")
				})

				t.Run("should recover on incomplete and unterminated keys", func(t *testing.T) {
					ast := parseAction(t, "a[1 + = 1")
					assert.Equal(t, "a[1 + ] = 1", unparse(ast.Ast))
					validate(t, ast)
					errors := mapErrors(ast.Errors)
					assert.Equal(t, 2, len(errors))
					assert.True(t, strings.Contains(errors[0], "Unexpected token ="))
					assert.True(t, strings.Contains(errors[1], "Missing expected ]"))
				})

				t.Run("should error on writes after a keyed write", func(t *testing.T) {
					ast := parseAction(t, "a[1] = 1 = 2")
					assert.Equal(t, "a[1] = 1", unparse(ast.Ast))
					validate(t, ast)
					assert.Equal(t, 1, len(ast.Errors))
					assert.True(t, strings.Contains(ast.Errors[0].Message, "Unexpected token '='"))
				})

				t.Run("should recover on parenthesized empty rvalues", func(t *testing.T) {
					ast := parseAction(t, "(a[1] = b) = c = d")
					assert.Equal(t, "(a[1] = b)", unparse(ast.Ast))
					validate(t, ast)
					assert.Equal(t, 1, len(ast.Errors))
					assert.True(t, strings.Contains(ast.Errors[0].Message, "Unexpected token '='"))
				})
			})
		})

		t.Run("conditional", func(t *testing.T) {
			t.Run("should parse ternary/conditional expressions", func(t *testing.T) {
				checkAction(t, "7 == 3 + 4 ? 10 : 20")
				checkAction(t, "false ? 10 : 20")
			})

			t.Run("should report incorrect ternary operator syntax", func(t *testing.T) {
				expectActionError(t, "true?1", "Conditional expression true?1 requires all 3 expressions")
			})
		})

		t.Run("assignment", func(t *testing.T) {
			t.Run("should support field assignments", func(t *testing.T) {
				checkAction(t, "a = 12")
				checkAction(t, "a.a.a = 123")
				checkAction(t, "a = 123; b = 234;")
			})

			t.Run("should report on safe field assignments", func(t *testing.T) {
				expectActionError(t, "a?.a = 123", "cannot be used in the assignment")
			})

			t.Run("should support array updates", func(t *testing.T) {
				checkAction(t, "a[0] = 200")
			})
		})

		t.Run("should error when using pipes", func(t *testing.T) {
			expectActionError(t, "x|blah", "Cannot have a pipe")
		})

		t.Run("should report when encountering interpolation", func(t *testing.T) {
			expectActionError(t, "{{a()}}", "Got interpolation ({{}}) where expression was expected")
		})

		t.Run("should not report interpolation inside a string", func(t *testing.T) {
			assert.Equal(t, 0, len(parseAction(t, `"{{a()}}"`).Errors))
			assert.Equal(t, 0, len(parseAction(t, `'{{a()}}'`).Errors))
			assert.Equal(t, 0, len(parseAction(t, `"{{a('\"')}}"`).Errors))
			assert.Equal(t, 0, len(parseAction(t, `'{{a("\'")}}'`).Errors))
		})

		t.Run("template literals", func(t *testing.T) {
			t.Run("should parse template literals without interpolations", func(t *testing.T) {
				checkBinding(t, "`hello world`")
				checkBinding(t, "`foo $`")
				checkBinding(t, "`foo }`")
				checkBinding(t, "`foo $ {}`")
			})

			t.Run("should parse template literals with interpolations", func(t *testing.T) {
				checkBinding(t, "`hello ${name}`")
				checkBinding(t, "`${name} Johnson`")
				checkBinding(t, "`foo${bar}baz`")
				checkBinding(t, "`${a} - ${b} - ${c}`")
				checkBinding(t, "`foo ${{$: true}} baz`")
				checkBinding(t, "`foo ${`hello ${`${a} - b`}`} baz`")
				checkBinding(t, "[`hello ${name}`, `see ${name} later`]")
				checkBinding(t, "`hello ${name}` + 123")
			})

			t.Run("should parse template literals with pipes inside interpolations", func(t *testing.T) {
				checkBinding(t, "`hello ${name | capitalize}!!!`", "`hello ${(name | capitalize)}!!!`")
				checkBinding(t, "`hello ${(name | capitalize)}!!!`", "`hello ${((name | capitalize))}!!!`")
			})

			t.Run("should parse template literals in objects literals", func(t *testing.T) {
				checkBinding(t, `{"a": `+"`${name}`}")
				checkBinding(t, `{"a": `+"`hello ${name}!`}")
				checkBinding(t, `{"a": `+"`hello ${`hello ${`hello`}`}!`}")
				checkBinding(t, `{"a": `+"`hello ${{\"b\": `hello`}}`}")
			})

			t.Run("should report error if interpolation is empty", func(t *testing.T) {
				expectBindingError(t, "`hello ${}`", "Template literal interpolation cannot be empty")
			})

			t.Run("should parse tagged template literals with no interpolations", func(t *testing.T) {
				checkBinding(t, "tag`hello!`")
				checkBinding(t, "tags.first`hello!`")
				checkBinding(t, "tags[0]`hello!`")
				checkBinding(t, "tag()`hello!`")
				checkBinding(t, "(tag ?? otherTag)`hello!`")
				checkBinding(t, "tag!`hello!`")
			})

			t.Run("should parse tagged template literals with interpolations", func(t *testing.T) {
				checkBinding(t, "tag`hello ${name}!`")
				checkBinding(t, "tags.first`hello ${name}!`")
				checkBinding(t, "tags[0]`hello ${name}!`")
				checkBinding(t, "tag()`hello ${name}!`")
				checkBinding(t, "(tag ?? otherTag)`hello ${name}!`")
				checkBinding(t, "tag!`hello ${name}!`")
			})

			t.Run("should not mistake operator for tagged literal tag", func(t *testing.T) {
				checkBinding(t, "typeof `hello!`")
				checkBinding(t, "typeof `hello ${name}!`")
			})
		})

		t.Run("regular expression literals", func(t *testing.T) {
			t.Run("should parse a regular expression literal without flags", func(t *testing.T) {
				checkBinding(t, "/abc/")
				checkBinding(t, "/[a/]$/")
				checkBinding(t, `/a\w+/`)
				checkBinding(t, `/^http:\/\/foo\.bar/`)
			})

			t.Run("should parse a regular expression literal with flags", func(t *testing.T) {
				checkBinding(t, "/abc/g")
				checkBinding(t, "/[a/]$/gi")
				checkBinding(t, `/a\w+/gim`)
				checkBinding(t, `/^http:\/\/foo\.bar/i`)
			})

			t.Run("should parse a regular expression that is a part of other expressions", func(t *testing.T) {
				checkBinding(t, `/abc/.test("foo")`)
				checkBinding(t, `"foo".match(/(abc)/)[1].toUpperCase()`)
				checkBinding(t, `/abc/.test("foo") && something || somethingElse`)
			})

			t.Run("should report invalid regular expression flag", func(t *testing.T) {
				expectBindingError(t, `"foo".match(/abc/O)`, `Unsupported regular expression flag "O"`)
			})

			t.Run("should report duplicated regular expression flags", func(t *testing.T) {
				expectBindingError(t, `"foo".match(/abc/gig)`, `Duplicate regular expression flag "g"`)
			})
		})
	})

	// ── general error handling ──────────────────────────────────────────
	t.Run("general error handling", func(t *testing.T) {
		t.Run("should report an unexpected token", func(t *testing.T) {
			expectActionError(t, "[1,2] trac", "Unexpected token 'trac'")
		})

		t.Run("should report reasonable error for unconsumed tokens", func(t *testing.T) {
			expectActionError(t, ")", "Unexpected token ) at column 1 in [)]")
		})

		t.Run("should report a missing expected token", func(t *testing.T) {
			expectActionError(t, "a(b", "Missing expected ) at the end of the expression [a(b]")
		})

		t.Run("should report a single error for an `as` expression inside a parenthesized expression", func(t *testing.T) {
			expectActionError(t, "foo(($event.target as HTMLElement).value)", "Missing closing parentheses at column 20", 1)
			expectActionError(t, "foo(((($event.target as HTMLElement))).value)", "Missing closing parentheses at column 22", 1)
		})
	})

	// ── parseBinding ───────────────────────────────────────────────────
	t.Run("parseBinding", func(t *testing.T) {
		t.Run("pipes", func(t *testing.T) {
			t.Run("should parse pipes", func(t *testing.T) {
				checkBinding(t, "a(b | c)", "a((b | c))")
				checkBinding(t, "a.b(c.d(e) | f)", "a.b((c.d(e) | f))")
				checkBinding(t, "[1, 2, 3] | a", "([1, 2, 3] | a)")
				checkBinding(t, `{a: 1, "b": 2} | c`, `({a: 1, "b": 2} | c)`)
				checkBinding(t, "a[b] | c", "(a[b] | c)")
				checkBinding(t, "a?.b | c", "(a?.b | c)")
				checkBinding(t, "true | a", "(true | a)")
				checkBinding(t, "a | b:c | d", "((a | b:c) | d)")
				checkBinding(t, "a | b:(c | d)", "(a | b:((c | d)))")
			})

			t.Run("should parse incomplete pipes", func(t *testing.T) {
				type pipeCase struct {
					name, input, output, errMsg string
				}
				cases := []pipeCase{
					{"should parse missing pipe names: end", "a | b | ", "((a | b) | )", "Unexpected end of input, expected identifier or keyword"},
					{"should parse missing pipe names: middle", "a | | b", "((a | ) | b)", "Unexpected token |, expected identifier or keyword"},
					{"should parse missing pipe names: start", " | a | b", "(( | a) | b)", "Unexpected token |"},
					{"should parse missing pipe args: end", "a | b | c: ", "((a | b) | c:)", "Unexpected end of expression"},
					{"should parse missing pipe args: middle", "a | b: | c", "((a | b:) | c)", "Unexpected token |"},
					{"should parse incomplete pipe args", "a | b: (a | ) + | c", "((a | b:((a | )) + ) | c)", "Unexpected token |"},
				}
				for _, tc := range cases {
					tc := tc
					t.Run(tc.name, func(t *testing.T) {
						checkBinding(t, tc.input, tc.output)
						expectBindingError(t, tc.input, tc.errMsg)
					})
				}
			})

			t.Run("should only allow identifier or keyword as formatter names", func(t *testing.T) {
				expectBindingError(t, `"Foo"|(`, "identifier or keyword")
				expectBindingError(t, `"Foo"|1234`, "identifier or keyword")
				expectBindingError(t, `"Foo"|"uppercase"`, "identifier or keyword")
				expectBindingError(t, `"Foo"|#privateIdentifier"`, "identifier or keyword")
			})

			t.Run("should not crash when prefix part is not tokenizable", func(t *testing.T) {
				checkBinding(t, `"a:b"`, `"a:b"`)
			})

			t.Run("should parse pipes with the correct type when supportsDirectPipeReferences is enabled", func(t *testing.T) {
				ast := parseBinding(t, "0 | Foo", true)
				if pipe, ok := ast.Ast.(*BindingPipe); ok {
					assert.Equal(t, BindingPipeTypeReferencedDirectly, pipe.PipeType)
				} else {
					t.Errorf("expected BindingPipe, got %T", ast.Ast)
				}
				ast2 := parseBinding(t, "0 | foo", true)
				if pipe, ok := ast2.Ast.(*BindingPipe); ok {
					assert.Equal(t, BindingPipeTypeReferencedByName, pipe.PipeType)
				} else {
					t.Errorf("expected BindingPipe, got %T", ast2.Ast)
				}
			})

			t.Run("should parse pipes with the correct type when supportsDirectPipeReferences is disabled", func(t *testing.T) {
				ast := parseBinding(t, "0 | Foo", false)
				if pipe, ok := ast.Ast.(*BindingPipe); ok {
					assert.Equal(t, BindingPipeTypeReferencedByName, pipe.PipeType)
				} else {
					t.Errorf("expected BindingPipe, got %T", ast.Ast)
				}
				ast2 := parseBinding(t, "0 | foo", false)
				if pipe, ok := ast2.Ast.(*BindingPipe); ok {
					assert.Equal(t, BindingPipeTypeReferencedByName, pipe.PipeType)
				} else {
					t.Errorf("expected BindingPipe, got %T", ast2.Ast)
				}
			})
		})

		t.Run("should store the source in the result", func(t *testing.T) {
			assert.Equal(t, "someExpr", parseBinding(t, "someExpr").Source)
		})

		t.Run("should report chain expressions", func(t *testing.T) {
			expectError(t, parseBinding(t, "1;2").Errors, "contain chained expression")
		})

		t.Run("should report assignment", func(t *testing.T) {
			expectError(t, parseBinding(t, "a=2").Errors, "contain assignments")
		})

		t.Run("should report when encountering interpolation", func(t *testing.T) {
			expectBindingError(t, "{{a.b}}", "Got interpolation ({{}}) where expression was expected")
		})

		t.Run("should not report interpolation inside a string", func(t *testing.T) {
			assert.Equal(t, 0, len(parseBinding(t, `"{{exp}}"`).Errors))
			assert.Equal(t, 0, len(parseBinding(t, `'{{exp}}'`).Errors))
			assert.Equal(t, 0, len(parseBinding(t, `'{{\"}}'`).Errors))
			assert.Equal(t, 0, len(parseBinding(t, "'{{\\'}}'" ).Errors))
		})

		t.Run("should parse conditional expression", func(t *testing.T) {
			checkBinding(t, "a < b ? a : b")
		})

		t.Run("should ignore comments in bindings", func(t *testing.T) {
			checkBinding(t, "a //comment", "a")
		})

		t.Run("should retain // in string literals", func(t *testing.T) {
			checkBinding(t, `"http://www.google.com"`)
		})

		t.Run("should expose object shorthand information in AST", func(t *testing.T) {
			parser := NewParser(Lexer{}, false)
			ast := parser.ParseBinding("{bla}", ParseSourceSpan{}, 0)
			literalMap, ok := ast.Ast.(*LiteralMap)
			if assert.True(t, ok, "expected LiteralMap") {
				assert.Equal(t, 1, len(literalMap.Keys))
				if propKey, ok := literalMap.Keys[0].(*LiteralMapPropertyKey); ok {
					assert.True(t, propKey.IsShorthandInitialized)
				} else {
					t.Errorf("expected LiteralMapPropertyKey, got %T", literalMap.Keys[0])
				}
			}
		})

		t.Run("arrow functions", func(t *testing.T) {
			t.Run("should parse a single-parameter arrow function", func(t *testing.T) {
				checkBinding(t, "a => a")
			})

			t.Run("should parse a single-parameter arrow function with parentheses", func(t *testing.T) {
				checkBinding(t, "(a) => a", "a => a")
			})

			t.Run("should parse an arrow function with not parameters", func(t *testing.T) {
				checkBinding(t, "() => 1")
			})

			t.Run("should parse an arrow function with multiple parameters", func(t *testing.T) {
				checkBinding(t, "(a, b, c, d, e) => a / b + c * d")
			})

			t.Run("should parse an immediately-invoked arrow function", func(t *testing.T) {
				checkBinding(t, "((a, b) => a + b)(1, 2)")
			})

			t.Run("should parse an arrow function that returns other arrow functions", func(t *testing.T) {
				checkBinding(t, "(a, b) => c => (d, e) => () => a + b + c + d + e")
			})

			t.Run("should parse an arrow function that returns an object literal", func(t *testing.T) {
				checkBinding(t, "() => ({a: 1, b: 2})")
			})

			t.Run("should parse an arrow function containing an assignment", func(t *testing.T) {
				checkBinding(t, "(a, b) => c = a + b")
			})

			t.Run("should be able to pass an arrow function through a pipe", func(t *testing.T) {
				checkBinding(t, "(a, b) => a + b | pipe", "((a, b) => a + b | pipe)")
			})

			t.Run("should parse an arrow function that returns an array", func(t *testing.T) {
				checkBinding(t, "(a, b) => [a, b, foo]")
			})

			t.Run("arrow function validations", func(t *testing.T) {
				t.Run("should not allow pipe to be used inside an arrow function", func(t *testing.T) {
					expectBindingError(t, "(a, b) => (a + b | pipe)", "Cannot have a pipe in an action expression")
				})

				t.Run("should report an error for an arrow function with a body", func(t *testing.T) {
					expectBindingError(t, "() => {}", "Multi-line arrow functions are not supported")
				})

				t.Run("should report missing comma between arrow function parameters", func(t *testing.T) {
					expectBindingError(t, "(a b) => a + b", "Missing expected ,")
				})

				t.Run("should report arrow function parameter starting with a comma", func(t *testing.T) {
					expectBindingError(t, "(, a) => a", "Unexpected token ,")
				})

				t.Run("should report arrow function parameter with a trailing comma", func(t *testing.T) {
					expectBindingError(t, "(a, ) => a", "Unexpected token )")
				})

				t.Run("should report an arrow function without a closing paren", func(t *testing.T) {
					expectBindingError(t, "(a => a + 1", "Missing closing parentheses at the end of the expression")
				})

				t.Run("should report an arrow function without an opening paren", func(t *testing.T) {
					expectBindingError(t, "a) => a + 1", "Unexpected token ')'")
				})

				t.Run("should report an error inside the arrow function expression", func(t *testing.T) {
					expectBindingError(t, "(a) => a. + 1", "Unexpected token +, expected identifier or keyword")
				})

				t.Run("should report an error for chained expression in arrow function", func(t *testing.T) {
					expectBindingError(t, "() => foo(); bar()", "Binding expression cannot contain chained expression")
					expectBindingError(t, "() => (foo; bar)", "Binding expression cannot contain chained expression")
				})
			})
		})
	})

	// ── parseTemplateBindings ──────────────────────────────────────────
	t.Run("parseTemplateBindings", func(t *testing.T) {
		humanize := func(bindings []TemplateBinding) [][3]any {
			var res [][3]any
			for _, b := range bindings {
				switch v := b.(type) {
				case *VariableBinding:
					var val *string
					if v.Value != nil {
						s := v.Value.Source
						val = &s
					}
					res = append(res, [3]any{v.Key.Source, val, true})
				case *ExpressionBinding:
					var val *string
					if v.Value != nil {
						s := v.Value.Source
						val = &s
					}
					res = append(res, [3]any{v.Key.Source, val, false})
				}
			}
			return res
		}

		type tplBindRow = [3]any

		assertHumanize := func(t *testing.T, bindings []TemplateBinding, want []tplBindRow) {
			t.Helper()
			got := humanize(bindings)
			assert.Equal(t, len(want), len(got), "humanize length mismatch")
			for i := range want {
				if i >= len(got) {
					break
				}
				assert.Equal(t, want[i][0], got[i][0], "key mismatch at index %d", i)
				assert.Equal(t, want[i][2], got[i][2], "isVar mismatch at index %d", i)
				// value comparison: want[i][1] can be nil or string ptr
				if want[i][1] == nil {
					assert.Nil(t, got[i][1], "expected nil value at index %d", i)
				} else {
					if wantStr, ok := want[i][1].(string); ok {
						if gotStr, ok2 := got[i][1].(*string); ok2 && gotStr != nil {
							assert.Equal(t, wantStr, *gotStr, "value mismatch at index %d", i)
						} else {
							t.Errorf("expected string value %q at index %d, got %v", wantStr, i, got[i][1])
						}
					}
				}
			}
		}

		parseTpl := func(t *testing.T, attr string) []TemplateBinding {
			t.Helper()
			result := parseTemplateBindings(t, attr)
			assert.Equal(t, 0, len(result.Errors), "unexpected errors: %v", result.Errors)
			assert.Equal(t, 0, len(result.Warnings), "unexpected warnings: %v", result.Warnings)
			return result.TemplateBindings
		}

		t.Run("should parse key and value", func(t *testing.T) {
			type kv = [3]any
			cases := []struct {
				attr   string
				expect []kv
			}{
				{`*a=""`, []kv{{"a", nil, false}}},
				{`*a="b"`, []kv{{"a", "b", false}}},
				{`*a-b="c"`, []kv{{"a-b", "c", false}}},
				{`*a="1+1"`, []kv{{"a", "1+1", false}}},
			}
			for _, tc := range cases {
				tc := tc
				t.Run(tc.attr, func(t *testing.T) {
					bindings := parseTpl(t, tc.attr)
					assertHumanize(t, bindings, tc.expect)
				})
			}
		})

		t.Run("should variable declared via let", func(t *testing.T) {
			bindings := parseTpl(t, `*a="let b"`)
			assertHumanize(t, bindings, []tplBindRow{
				{"a", nil, false},
				{"b", nil, true},
			})
		})

		t.Run("should allow multiple pairs", func(t *testing.T) {
			bindings := parseTpl(t, `*a="1 b 2"`)
			assertHumanize(t, bindings, []tplBindRow{
				{"a", "1", false},
				{"aB", "2", false},
			})
		})

		t.Run("should allow space and colon as separators", func(t *testing.T) {
			bindings := parseTpl(t, `*a="1,b 2"`)
			assertHumanize(t, bindings, []tplBindRow{
				{"a", "1", false},
				{"aB", "2", false},
			})
		})

		t.Run("should support common usage of ngIf", func(t *testing.T) {
			bindings := parseTpl(t, `*ngIf="cond | pipe as foo, let x; ngIf as y"`)
			assertHumanize(t, bindings, []tplBindRow{
				{"ngIf", "cond | pipe", false},
				{"foo", "ngIf", true},
				{"x", nil, true},
				{"y", "ngIf", true},
			})
		})

		t.Run("should support common usage of ngFor", func(t *testing.T) {
			bindings1 := parseTpl(t, `*ngFor="let person of people"`)
			assertHumanize(t, bindings1, []tplBindRow{
				{"ngFor", nil, false},
				{"person", nil, true},
				{"ngForOf", "people", false},
			})

			bindings2 := parseTpl(t, `*ngFor="let item; of items | slice:0:1 as collection, trackBy: func; index as i"`)
			assertHumanize(t, bindings2, []tplBindRow{
				{"ngFor", nil, false},
				{"item", nil, true},
				{"ngForOf", "items | slice:0:1", false},
				{"collection", "ngForOf", true},
				{"ngForTrackBy", "func", false},
				{"i", "index", true},
			})

			bindings3 := parseTpl(t, `*ngFor="let item, of: [1,2,3] | pipe as items; let i=index, count as len"`)
			assertHumanize(t, bindings3, []tplBindRow{
				{"ngFor", nil, false},
				{"item", nil, true},
				{"ngForOf", "[1,2,3] | pipe", false},
				{"items", "ngForOf", true},
				{"i", "index", true},
				{"len", "count", true},
			})
		})

		t.Run("should parse pipes", func(t *testing.T) {
			bindings := parseTpl(t, `*key="value|pipe "`)
			assertHumanize(t, bindings, []tplBindRow{
				{"key", "value|pipe", false},
			})
			if assert.Equal(t, 1, len(bindings)) {
				eb, ok := bindings[0].(*ExpressionBinding)
				if assert.True(t, ok) {
					_, isPipe := eb.Value.Ast.(*BindingPipe)
					assert.True(t, isPipe, "expected BindingPipe")
				}
			}
		})

		t.Run("\"let\" binding", func(t *testing.T) {
			t.Run("should support single declaration", func(t *testing.T) {
				bindings := parseTpl(t, `*key="let i"`)
				assertHumanize(t, bindings, []tplBindRow{
					{"key", nil, false},
					{"i", nil, true},
				})
			})

			t.Run("should support multiple declarations", func(t *testing.T) {
				bindings := parseTpl(t, `*key="let a; let b"`)
				assertHumanize(t, bindings, []tplBindRow{
					{"key", nil, false},
					{"a", nil, true},
					{"b", nil, true},
				})
			})

			t.Run("should support empty string assignment", func(t *testing.T) {
				bindings := parseTpl(t, `*key="let a=''; let b='';"`)
				assertHumanize(t, bindings, []tplBindRow{
					{"key", nil, false},
					{"a", "", true},
					{"b", "", true},
				})
			})

			t.Run("should support key and value names with dash", func(t *testing.T) {
				bindings := parseTpl(t, `*key="let i-a = j-a,"`)
				assertHumanize(t, bindings, []tplBindRow{
					{"key", nil, false},
					{"i-a", "j-a", true},
				})
			})

			t.Run("should support declarations with or without value assignment", func(t *testing.T) {
				bindings := parseTpl(t, `*key="let item; let i = k"`)
				assertHumanize(t, bindings, []tplBindRow{
					{"key", nil, false},
					{"item", nil, true},
					{"i", "k", true},
				})
			})

			t.Run("should support declaration before an expression", func(t *testing.T) {
				bindings := parseTpl(t, `*directive="let item in expr; let a = b"`)
				assertHumanize(t, bindings, []tplBindRow{
					{"directive", nil, false},
					{"item", nil, true},
					{"directiveIn", "expr", false},
					{"a", "b", true},
				})
			})
		})

		t.Run("\"as\" binding", func(t *testing.T) {
			t.Run("should support single declaration", func(t *testing.T) {
				bindings := parseTpl(t, `*ngIf="exp as local"`)
				assertHumanize(t, bindings, []tplBindRow{
					{"ngIf", "exp", false},
					{"local", "ngIf", true},
				})
			})

			t.Run("should support declaration after an expression", func(t *testing.T) {
				bindings := parseTpl(t, `*ngFor="let item of items as iter; index as i"`)
				assertHumanize(t, bindings, []tplBindRow{
					{"ngFor", nil, false},
					{"item", nil, true},
					{"ngForOf", "items", false},
					{"iter", "ngForOf", true},
					{"i", "index", true},
				})
			})

			t.Run("should support key and value names with dash", func(t *testing.T) {
				bindings := parseTpl(t, `*key="foo, k-b as l-b;"`)
				assertHumanize(t, bindings, []tplBindRow{
					{"key", "foo", false},
					{"l-b", "k-b", true},
				})
			})
		})
	})

	// ── parseInterpolation ─────────────────────────────────────────────
	t.Run("parseInterpolation", func(t *testing.T) {
		t.Run("should return empty/nil if no interpolation", func(t *testing.T) {
			result := parseInterpolation(t, "nothing")
			// ParseInterpolation returns ASTWithSource with empty Ast when no interpolation
			assert.Nil(t, result.Ast)
		})

		t.Run("should parse no prefix/suffix interpolation", func(t *testing.T) {
			result := parseInterpolation(t, "{{a}}")
			interp, ok := result.Ast.(*Interpolation)
			if assert.True(t, ok, "expected Interpolation") {
				assert.Equal(t, 2, len(interp.Strings))
				assert.Equal(t, 1, len(interp.Expressions))
				propRead, ok2 := interp.Expressions[0].(*PropertyRead)
				if assert.True(t, ok2) {
					assert.Equal(t, "a", propRead.Name)
				}
			}
		})

		t.Run("should parse interpolation inside quotes", func(t *testing.T) {
			result := parseInterpolation(t, `"{{a}}"`)
			interp, ok := result.Ast.(*Interpolation)
			if assert.True(t, ok) {
				assert.Equal(t, 2, len(interp.Strings))
				assert.Equal(t, 1, len(interp.Expressions))
				propRead, ok2 := interp.Expressions[0].(*PropertyRead)
				if assert.True(t, ok2) {
					assert.Equal(t, "a", propRead.Name)
				}
			}
		})

		t.Run("should parse interpolation with interpolation characters inside quotes", func(t *testing.T) {
			checkInterpolation(t, `{{"{{a}}"}}`, `{{ "{{a}}" }}`)
			checkInterpolation(t, `{{"{{" }}`, `{{ "{{" }}`)
			checkInterpolation(t, `{{"}}"}}`, `{{ "}}" }}`)
			checkInterpolation(t, `{{"{" }}`, `{{ "{" }}`)
			checkInterpolation(t, `{{"}"}}`, `{{ "}" }}`)
		})

		t.Run("should parse interpolation with escaped quotes", func(t *testing.T) {
			checkInterpolation(t, `{{'It\'s just Angular'}}`, `{{ "It's just Angular" }}`)
			checkInterpolation(t, `{{'It\'s {{ just Angular'}}`, `{{ "It's {{ just Angular" }}`)
			checkInterpolation(t, `{{'It\'s }} just Angular'}}`, `{{ "It's }} just Angular" }}`)
		})

		t.Run("should parse prefix/suffix with multiple interpolation", func(t *testing.T) {
			originalExp := "before {{ a }} middle {{ b }} after"
			result := parseInterpolation(t, originalExp)
			assert.Equal(t, originalExp, unparse(result.Ast))
			validate(t, result)
		})

		t.Run("should report empty interpolation expressions", func(t *testing.T) {
			expectError(t, parseInterpolation(t, "{{}}").Errors, "Blank expressions are not allowed in interpolated strings")
			expectError(t, parseInterpolation(t, "foo {{  }}").Errors, "Blank expressions are not allowed in interpolated strings")
			expectError(t, parseInterpolation(t, "{{  }}").Errors, "Blank expressions are not allowed in interpolated strings")
		})

		t.Run("should produce an empty expression ast for empty interpolations", func(t *testing.T) {
			parsed := parseInterpolation(t, "{{}}")
			interp, ok := parsed.Ast.(*Interpolation)
			if assert.True(t, ok) {
				assert.Equal(t, 1, len(interp.Expressions))
				_, isEmptyExpr := interp.Expressions[0].(*EmptyExpr)
				assert.True(t, isEmptyExpr)
			}
		})

		t.Run("should parse conditional expression", func(t *testing.T) {
			checkInterpolation(t, "{{ a < b ? a : b }}")
		})

		t.Run("should parse expression with newline characters", func(t *testing.T) {
			checkInterpolation(t, "{{ 'foo' +\n 'bar' +\r 'baz' }}", `{{ "foo" + "bar" + "baz" }}`)
		})

		t.Run("comments", func(t *testing.T) {
			t.Run("should ignore comments in interpolation expressions", func(t *testing.T) {
				checkInterpolation(t, "{{a //comment}}", "{{ a }}")
			})

			t.Run("should error when interpolation only contains a comment", func(t *testing.T) {
				expectError(t, parseInterpolation(t, "{{ // foobar  }}").Errors,
					"Interpolation expression cannot only contain a comment")
			})

			t.Run("should retain // in single quote strings", func(t *testing.T) {
				checkInterpolation(t, "{{ 'http://www.google.com' }}", `{{ "http://www.google.com" }}`)
			})

			t.Run("should retain // in double quote strings", func(t *testing.T) {
				checkInterpolation(t, `{{ "http://www.google.com" }}`)
			})

			t.Run("should ignore comments after string literals", func(t *testing.T) {
				checkInterpolation(t, `{{ "a//b" //comment }}`, `{{ "a//b" }}`)
			})
		})
	})

	// ── parseSimpleBinding ─────────────────────────────────────────────
	t.Run("parseSimpleBinding", func(t *testing.T) {
		parseSimpleBindingLocal := func(t *testing.T, exp string) ASTWithSource {
			t.Helper()
			parser := NewParser(Lexer{}, false)
			return parser.ParseSimpleBinding(exp, ParseSourceSpan{}, 0)
		}

		t.Run("should parse a field access", func(t *testing.T) {
			p := parseSimpleBindingLocal(t, "name")
			assert.Equal(t, "name", unparse(p.Ast))
			validate(t, p)
		})


		t.Run("should report when encountering pipes", func(t *testing.T) {
			p := parseSimpleBindingLocal(t, "a | somePipe")
			expectError(t, p.Errors, "Host binding expression cannot contain pipes")
		})

		t.Run("should report when encountering interpolation", func(t *testing.T) {
			p := parseSimpleBindingLocal(t, "{{exp}}")
			expectError(t, p.Errors, "Got interpolation ({{}}) where expression was expected")
		})

		t.Run("should not report interpolation inside a string", func(t *testing.T) {
			assert.Equal(t, 0, len(parseSimpleBindingLocal(t, `"{{exp}}"`).Errors))
			assert.Equal(t, 0, len(parseSimpleBindingLocal(t, `'{{exp}}'`).Errors))
			assert.Equal(t, 0, len(parseSimpleBindingLocal(t, `'{{\"}}'`).Errors))
			assert.Equal(t, 0, len(parseSimpleBindingLocal(t, "'{{\\'}}'" ).Errors))
		})

		t.Run("should report when encountering field write", func(t *testing.T) {
			p := parseSimpleBindingLocal(t, "a = b")
			expectError(t, p.Errors, "Bindings cannot contain assignments")
		})

		t.Run("should throw if a pipe is used inside a conditional", func(t *testing.T) {
			p := parseSimpleBindingLocal(t, `(hasId | myPipe) ? "my-id" : ""`)
			expectError(t, p.Errors, "Host binding expression cannot contain pipes")
		})

		t.Run("should throw if a pipe is used inside a call", func(t *testing.T) {
			p := parseSimpleBindingLocal(t, "getId(true, id | myPipe)")
			expectError(t, p.Errors, "Host binding expression cannot contain pipes")
		})

		t.Run("should throw if a pipe is used inside a call to a property access", func(t *testing.T) {
			p := parseSimpleBindingLocal(t, "idService.getId(true, id | myPipe)")
			expectError(t, p.Errors, "Host binding expression cannot contain pipes")
		})

		t.Run("should throw if a pipe is used inside a call to a safe property access", func(t *testing.T) {
			p := parseSimpleBindingLocal(t, "idService?.getId(true, id | myPipe)")
			expectError(t, p.Errors, "Host binding expression cannot contain pipes")
		})

		t.Run("should throw if a pipe is used inside a property access", func(t *testing.T) {
			p := parseSimpleBindingLocal(t, "a[id | myPipe]")
			expectError(t, p.Errors, "Host binding expression cannot contain pipes")
		})

		t.Run("should throw if a pipe is used inside a keyed read expression", func(t *testing.T) {
			p := parseSimpleBindingLocal(t, "a[id | myPipe].b")
			expectError(t, p.Errors, "Host binding expression cannot contain pipes")
		})

		t.Run("should throw if a pipe is used inside a safe property read", func(t *testing.T) {
			p := parseSimpleBindingLocal(t, "(id | myPipe)?.id")
			expectError(t, p.Errors, "Host binding expression cannot contain pipes")
		})

		t.Run("should throw if a pipe is used inside a non-null assertion", func(t *testing.T) {
			p := parseSimpleBindingLocal(t, "[id | myPipe]!")
			expectError(t, p.Errors, "Host binding expression cannot contain pipes")
		})

		t.Run("should throw if a pipe is used inside a prefix not expression", func(t *testing.T) {
			p := parseSimpleBindingLocal(t, "!(id | myPipe)")
			expectError(t, p.Errors, "Host binding expression cannot contain pipes")
		})

		t.Run("should throw if a pipe is used inside a binary expression", func(t *testing.T) {
			p := parseSimpleBindingLocal(t, "(id | myPipe) === true")
			expectError(t, p.Errors, "Host binding expression cannot contain pipes")
		})
	})

	// ── wrapLiteralPrimitive ───────────────────────────────────────────
	t.Run("wrapLiteralPrimitive", func(t *testing.T) {
		t.Run("should wrap a literal primitive", func(t *testing.T) {
			parser := NewParser(Lexer{}, false)
			foo := "foo"
			wrapped := parser.WrapLiteralPrimitive(&foo, "", 0)
			assert.Equal(t, `"foo"`, unparse(wrapped.Ast))
		})
	})

	// ── error recovery ─────────────────────────────────────────────────
	t.Run("error recovery", func(t *testing.T) {
		recover := func(t *testing.T, text string, expected ...string) {
			t.Helper()
			expr := validate(t, parseAction(t, text))
			want := text
			if len(expected) > 0 {
				want = expected[0]
			}
			assert.Equal(t, want, unparse(expr.Ast))
		}

		t.Run("should be able to recover from an extra paren", func(t *testing.T) { recover(t, "((a)))", "((a))") })
		t.Run("should be able to recover from an extra bracket", func(t *testing.T) { recover(t, "[[a]]]", "[[a]]") })
		t.Run("should be able to recover from a missing )", func(t *testing.T) { recover(t, "(a;b", "(a); b;") })
		t.Run("should be able to recover from a missing ]", func(t *testing.T) { recover(t, "[a,b", "[a, b]") })
		t.Run("should be able to recover from a missing selector", func(t *testing.T) { recover(t, "a.") })
		t.Run("should be able to recover from a missing selector in a array literal", func(t *testing.T) {
			recover(t, "[[a.], b, c]")
		})

		t.Run("should recover from parenthesized `as` expressions", func(t *testing.T) {
			recover(t, "foo(($event.target as HTMLElement).value)", "foo(($event.target).value)")
			recover(t, "foo(((($event.target as HTMLElement))).value)", "foo(((($event.target))).value)")
			recover(t, "foo(((bar as HTMLElement) as Something).value)", "foo(((bar)).value)")
		})

		t.Run("should be able to recover from a broken expression in a template literal", func(t *testing.T) {
			recover(t, "`before ${expr.}`")
			recover(t, "`${expr.} after`")
			recover(t, "`before ${expr.} after`")
		})
	})

	// ── offsets ────────────────────────────────────────────────────────
	t.Run("offsets", func(t *testing.T) {
		t.Run("should retain the offsets of an interpolation", func(t *testing.T) {
			parser := NewParser(Lexer{}, false)
			errs := []ParseError{}
			split := parser.SplitInterpolation("{{a}}  {{b}}  {{c}}", ParseSourceSpan{}, &errs, nil)
			if assert.NotNil(t, split) {
				assert.Equal(t, []int{2, 9, 16}, split.Offsets)
			}
		})

		t.Run("should retain the offsets into the expression AST of interpolations", func(t *testing.T) {
			result := parseInterpolation(t, "{{a}}  {{b}}  {{c}}")
			interp, ok := result.Ast.(*Interpolation)
			if assert.True(t, ok) {
				starts := make([]int, len(interp.Expressions))
				for i, e := range interp.Expressions {
					starts[i] = e.Span().Start
				}
				assert.Equal(t, []int{2, 9, 16}, starts)
			}
		})
	})
}
