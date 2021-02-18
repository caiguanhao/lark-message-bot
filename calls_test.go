package main

import (
	"strings"
	"testing"
)

type (
	test struct{}
)

func (t test) A() string                      { return "A" }
func (t test) B(a string) string              { return "B" + a }
func (t test) C(a, b string) string           { return "C" + a + b }
func (t test) D(a, b, c string) string        { return "D" + a + b + c }
func (t test) E(a ...string) string           { return "E" + strings.Join(a, ",") }
func (t test) F(a string, b ...string) string { return "F" + a + "/" + strings.Join(b, ",") }
func (t test) G(a, b string, c ...string) string {
	return "G" + a + "/" + b + "/" + strings.Join(c, ",")
}

func Test_call(t *testing.T) {
	o := test{}
	cases := [][]string{
		{"+()", callUnknownExpression},
		{"0()", callUnknownFunction},
		{"+", callUnknownExpression},
		{"foobar", callUnknownFunction},
		{"A", "A"},
		{"a", "A"},
		{"A()", "A"},
		{"a()", "A"},
		{"b()", callTooFewArguments},
		{"b(1)", "B1"},
		{"b(1,a)", callTooManyArguments},
		{"c()", callTooFewArguments},
		{"c(1)", callTooFewArguments},
		{"c(1,a)", "C1a"},
		{"c(1,a,b)", callTooManyArguments},
		{"d()", callTooFewArguments},
		{"d(1)", callTooFewArguments},
		{"d(1,a)", callTooFewArguments},
		{"d(1,a,b)", "D1ab"},
		{"d(1,a,b,x)", callTooManyArguments},
		{"e()", "E"},
		{"e(a)", "Ea"},
		{"e(a, bc, d)", "Ea,bc,d"},
		{"f(a, bc, d)", "Fa/bc,d"},
		{"g(a, bc, d)", "Ga/bc/d"},
		{"g(a, bc, d, 12, 34)", "Ga/bc/d,12,34"},
		{"help", strings.Join([]string{"A", "B", "C", "D", "E", "F", "G"}, "\n")},
	}
	for _, c := range cases {
		val := call(o, c[0])
		if val == c[1] {
			t.Logf(`call(o, "%s") == "%s" passed`, c[0], c[1])
		} else {
			t.Errorf(`call(o, "%s") should be "%s", got "%s"`, c[0], c[1], val)
		}
	}
}
