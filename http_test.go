package main

import (
	"testing"
)

func Test_parseCall(t *testing.T) {
	cases := []struct {
		input            string
		expectedFuncName string
		expectedArgs     []string
	}{
		{"", "", []string{}},
		{"a", "a", []string{}},
		{"a(", "a", []string{}},
		{"a()", "a", []string{}},
		{" b ", "b", []string{}},
		{" b (", "b", []string{}},
		{" b ()", "b", []string{}},
		{" b (  ) ", "b", []string{}},
		{" cd (1) ", "cd", []string{"1"}},
		{" cd ( 2 ) ", "cd", []string{"2"}},
		{" efg ( hij , , klmn ) ", "efg", []string{"hij", "", "klmn"}},
	}
	var h *httpHandler
	for _, kase := range cases {
		f, a := h.parseCall(kase.input)
		if f != kase.expectedFuncName {
			t.Errorf(`parseCall("%s") func name should be "%s", got "%s"`, kase.input, kase.expectedFuncName, f)
		}
		if len(a) != len(kase.expectedArgs) {
			t.Errorf(`parseCall("%s") args size should be %d, got %d`, kase.input, len(kase.expectedArgs), len(a))
		}
		for i := range kase.expectedArgs {
			if a[i] != kase.expectedArgs[i] {
				t.Errorf(`parseCall("%s") args #%d should be %s, got %s`, kase.input, i, kase.expectedArgs[i], a[i])
			}
		}
	}
}
