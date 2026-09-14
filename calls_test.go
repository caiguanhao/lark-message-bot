package main

import (
	"strings"
	"testing"
)

type (
	test struct{}

	FId  string
	F2Id string
)

func (t test) A() string               { return "A" }
func (t test) B(a string) string       { return "B" + a }
func (t test) C(a, b string) string    { return "C" + a + b }
func (t test) D(a, b, c string) string { return "D" + a + b + c }
func (t test) E(a ...string) string    { return "E" + strings.Join(a, ",") }
func (t test) F(a FId, _b ...F2Id) string {
	var b []string
	for i := range _b {
		b = append(b, string(_b[i]))
	}
	return "F" + string(a) + "/" + strings.Join(b, ",")
}
func (t test) G(a, b string, c ...string) string {
	return "G" + a + "/" + b + "/" + strings.Join(c, ",")
}
func (t test) LongName(a string) string { return "L" + a }
func (t test) Help() string             { return getMethods(t) }

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
		{"f()", callTooFewArguments},
		{"f(a, bc, d)", "Fa/bc,d"},
		{"g(a)", callTooFewArguments},
		{"g(a, bc, d)", "Ga/bc/d"},
		{"g(a, bc, d, 12, 34)", "Ga/bc/d,12,34"},
		{"longname(mdycgl.fs)", "Lmdycgl.fs"},
		{"Long_Name(mdycgl.fs)", "Lmdycgl.fs"},
		{"LONG_NAME(mdycgl.fs)", "Lmdycgl.fs"},
		{"b(127.0.0.1)", "B127.0.0.1"},
		{"c(1.2.3.4, 5.6.7.8)", "C1.2.3.45.6.7.8"},
		{"b(1.2.3.4.5)", callUnknownExpression},
		{"b(room1.2.3.4)", callUnknownExpression},
		{`b(<a href="http://test.wanico.net">test.wanico.net</a>)`, "Btest.wanico.net"},
		{`b([test.wanico.net](http://test.wanico.net/))`, "Btest.wanico.net"},
		{`b([test.wanico.net](https://test.wanico.net))`, "Btest.wanico.net"},
		{`c([a.example.com](http://a.example.com/), 1.2.3.4)`, "Ca.example.com1.2.3.4"},
		{`b(<a href="https://test.wanico.net/">test.wanico.net</a>)`, "Btest.wanico.net"},
		{`b(<a href="http://test.wanico.net/">http://test.wanico.net</a>)`, "Btest.wanico.net"},
		{`long_name(<a href="http://test.wanico.net">test.wanico.net</a>)`, "Ltest.wanico.net"},
		{`c(<a href="http://a.example.com">a.example.com</a>, 1.2.3.4)`, "Ca.example.com1.2.3.4"},
		{"b(http://test.wanico.net)", "Btest.wanico.net"},
		{"b(https://test.wanico.net/)", "Btest.wanico.net"},
		{"b(a&amp;b)", "Ba&b"},
		{"help", strings.Join([]string{
			"A()", "B(string)", "C(string, string)",
			"D(string, string, string)", "E(string...)",
			"F(FId, F2Id...)", "G(string, string, string...)",
			"Help()", "LongName(string)",
			"For more info, visit github.com/caiguanhao/lark-message-bot",
		}, "\n")},
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

func Test_callCreateSubdomain(t *testing.T) {
	o := MastersCall{Call{senderOpenId: "test", http: &httpHandler{}}}
	notConfigured := "aliyun dns is not configured, " +
		"please specify aliyun access key id, secret, main domain and default ip"
	for _, expression := range []string{"CREATE_SUBDOMAIN(mdycgl.fs)", "LIST_SUBDOMAINS"} {
		if got := call(o, expression); got != notConfigured {
			t.Errorf(`call(o, "%s") should be "%s", got "%s"`, expression, notConfigured, got)
		}
	}
}

func Test_callCreateSubdomainValidation(t *testing.T) {
	o := MastersCall{Call{senderOpenId: "test", http: &httpHandler{
		aliyun: &aliyunClient{
			AccessKeyId:     "id",
			AccessKeySecret: "secret",
			MainDomain:      "example.com",
			DefaultIP:       "8.8.8.8",
		},
	}}}
	cases := [][]string{
		{"CreateSubdomain(a_b)", `"a_b" is not a valid subdomain name`},
		{"CreateSubdomain(abc, 999.1.1.1)", `"999.1.1.1" is not a valid IPv4 address`},
		{"CreateSubdomain(abc, 12)", `"12" is not a valid IPv4 address`},
	}
	for _, c := range cases {
		if got := call(o, c[0]); got != c[1] {
			t.Errorf(`call(o, "%s") should be "%s", got "%s"`, c[0], c[1], got)
		}
	}
}
