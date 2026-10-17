// SPDX-License-Identifier: AGPL-3.0
package store

import "testing"

func TestEscapeLikeNeutralisesWildcards(t *testing.T) {
	cases := map[string]string{
		"normal":       "normal",
		"100%":         `100\%`,
		"a_b":          `a\_b`,
		`back\slash`:   `back\\slash`,
		"%' OR 1=1 --": `\%' OR 1=1 --`,
	}
	for in, want := range cases {
		if got := EscapeLike(in); got != want {
			t.Errorf("EscapeLike(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPatternsWrapEscapedValue(t *testing.T) {
	if got := ContainsPattern("50%"); got != `%50\%%` {
		t.Errorf("ContainsPattern = %q", got)
	}
	if got := PrefixPattern("a_b"); got != `a\_b%` {
		t.Errorf("PrefixPattern = %q", got)
	}
}
