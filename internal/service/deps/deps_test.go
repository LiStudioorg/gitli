// SPDX-License-Identifier: AGPL-3.0
package deps

import "testing"

func TestParseGoMod(t *testing.T) {
	src := `module example.com/x

go 1.22

require (
	github.com/go-chi/chi/v5 v5.0.10
	golang.org/x/crypto v0.20.0 // indirect
)

require github.com/BurntSushi/toml v1.3.2
`
	got := parseGoMod(src)
	if len(got) != 3 {
		t.Fatalf("expected 3 deps, got %d: %+v", len(got), got)
	}
	if got[0].Name != "github.com/go-chi/chi/v5" || got[0].Version != "v5.0.10" {
		t.Fatalf("dep0 = %+v", got[0])
	}
	if got[2].Name != "github.com/BurntSushi/toml" {
		t.Fatalf("single-line require not parsed: %+v", got)
	}
}

func TestParsePackageJSON(t *testing.T) {
	data := []byte(`{"dependencies":{"vue":"^3.4.0"},"devDependencies":{"vite":"^5.0.0"}}`)
	got := parsePackageJSON(data)
	if len(got) != 2 {
		t.Fatalf("expected 2 npm deps, got %d", len(got))
	}
}

func TestParsePOM(t *testing.T) {
	data := []byte(`<project><dependencies>
		<dependency><groupId>org.slf4j</groupId><artifactId>slf4j-api</artifactId><version>2.0.9</version></dependency>
	</dependencies></project>`)
	got := parsePOM(data)
	if len(got) != 1 || got[0].Name != "org.slf4j:slf4j-api" || got[0].Version != "2.0.9" {
		t.Fatalf("pom deps = %+v", got)
	}
}

func TestParseRequirements(t *testing.T) {
	src := `# comment
requests==2.31.0
flask>=2.0
numpy; python_version >= "3.8"
-r other.txt
`
	got := parseRequirements(src)
	if len(got) != 3 {
		t.Fatalf("expected 3 pip deps, got %d: %+v", len(got), got)
	}
	if got[0].Name != "requests" || got[0].Version != "2.31.0" {
		t.Fatalf("requests = %+v", got[0])
	}
	if got[1].Name != "flask" {
		t.Fatalf("flask = %+v", got[1])
	}
}

func TestParseManifestsEmpty(t *testing.T) {
	if got := ParseManifests(nil, nil, nil, nil); len(got) != 0 {
		t.Fatalf("expected no deps, got %d", len(got))
	}
}
