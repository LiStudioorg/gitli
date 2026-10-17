// SPDX-License-Identifier: AGPL-3.0
package codeowners

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	body := `# comment
*       @alice
/docs/  @bob @carol
*.go    alice
/root   @dave  # trailing comment

`
	rules, err := Parse([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 4 {
		t.Fatalf("expected 4 rules, got %d: %+v", len(rules), rules)
	}
	if !reflect.DeepEqual(rules[0].Owners, []string{"alice"}) {
		t.Fatalf("rule0 owners = %v", rules[0].Owners)
	}
	if !reflect.DeepEqual(rules[2].Owners, []string{"alice"}) {
		t.Fatalf("bare owner should drop '@': %v", rules[2].Owners)
	}
}

func TestOwnersForLastMatchWins(t *testing.T) {
	rules, _ := Parse([]byte("* @default\ndocs/ @docs-team\n"))
	// A file under docs/ is matched by both; the later rule wins.
	if got := OwnersFor(rules, "docs/guide.md"); !reflect.DeepEqual(got, []string{"docs-team"}) {
		t.Fatalf("docs/guide.md = %v", got)
	}
	if got := OwnersFor(rules, "main.go"); !reflect.DeepEqual(got, []string{"default"}) {
		t.Fatalf("main.go = %v", got)
	}
}

func TestOwnersForPatterns(t *testing.T) {
	rules, _ := Parse([]byte("*.go @gophers\n/src/ @src-team\n"))
	if got := OwnersFor(rules, "internal/app/app.go"); !reflect.DeepEqual(got, []string{"gophers"}) {
		t.Fatalf("*.go match = %v", got)
	}
	if got := OwnersFor(rules, "src/lib/x.ts"); !reflect.DeepEqual(got, []string{"src-team"}) {
		t.Fatalf("/src/ match = %v", got)
	}
}
