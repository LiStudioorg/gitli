// SPDX-License-Identifier: AGPL-3.0
// Package deps parses dependency manifests and stores a repository dependency
// graph.
package deps

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"strings"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Dependency is a single parsed dependency.
type Dependency struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
	Version   string `json:"version"`
}

// Ecosystem keys.
const (
	EcoGo    = "go"
	EcoNPM   = "npm"
	EcoMaven = "maven"
	EcoPip   = "pip"
)

// Service stores and queries dependency graphs.
type Service struct {
	st *store.Store
}

// New builds a deps Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Replace stores the dependency list for a repo, replacing prior rows.
func (s *Service) Replace(ctx context.Context, repoID int64, list []Dependency) error {
	tx, err := s.st.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, s.st.Rebind(`DELETE FROM repo_dependencies WHERE repo_id = ?`), repoID); err != nil {
		return err
	}
	ins := s.st.Rebind(`INSERT INTO repo_dependencies (repo_id, ecosystem, name, version) VALUES (?, ?, ?, ?)`)
	for _, d := range list {
		if _, err := tx.ExecContext(ctx, ins, repoID, d.Ecosystem, d.Name, d.Version); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// List returns dependencies for a repo.
func (s *Service) List(ctx context.Context, repoID int64) ([]Dependency, error) {
	q := s.st.Rebind(`SELECT ecosystem, name, version FROM repo_dependencies WHERE repo_id = ? ORDER BY ecosystem, name`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []Dependency{}
	for rows.Next() {
		var d Dependency
		if err := rows.Scan(&d.Ecosystem, &d.Name, &d.Version); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ParseManifests parses any provided manifest contents into dependencies.
// Missing (empty) manifests are skipped.
func ParseManifests(goMod, packageJSON, pomXML, requirementsTxt []byte) []Dependency {
	var out []Dependency
	if len(goMod) > 0 {
		out = append(out, parseGoMod(string(goMod))...)
	}
	if len(packageJSON) > 0 {
		out = append(out, parsePackageJSON(packageJSON)...)
	}
	if len(pomXML) > 0 {
		out = append(out, parsePOM(pomXML)...)
	}
	if len(requirementsTxt) > 0 {
		out = append(out, parseRequirements(string(requirementsTxt))...)
	}
	return out
}

func parseGoMod(s string) []Dependency {
	var out []Dependency
	inBlock := false
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "require (") {
			inBlock = true
			continue
		}
		if inBlock && t == ")" {
			inBlock = false
			continue
		}
		var modLine string
		switch {
		case inBlock && t != "" && !strings.HasPrefix(t, "//"):
			modLine = t
		case strings.HasPrefix(t, "require "):
			modLine = strings.TrimPrefix(t, "require ")
		default:
			continue
		}
		modLine = strings.SplitN(modLine, "//", 2)[0]
		f := strings.Fields(modLine)
		if len(f) >= 2 && strings.HasPrefix(f[1], "v") {
			out = append(out, Dependency{Ecosystem: EcoGo, Name: f[0], Version: f[1]})
		}
	}
	return out
}

func parsePackageJSON(data []byte) []Dependency {
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}
	var out []Dependency
	seen := map[string]bool{}
	add := func(m map[string]string) {
		for name, ver := range m {
			if seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, Dependency{Ecosystem: EcoNPM, Name: name, Version: ver})
		}
	}
	add(pkg.Dependencies)
	add(pkg.DevDependencies)
	return out
}

type pomProject struct {
	Dependencies struct {
		Items []struct {
			GroupID    string `xml:"groupId"`
			ArtifactID string `xml:"artifactId"`
			Version    string `xml:"version"`
		} `xml:"dependency"`
	} `xml:"dependencies"`
	DepMgmt struct {
		Dependencies struct {
			Items []struct {
				GroupID    string `xml:"groupId"`
				ArtifactID string `xml:"artifactId"`
				Version    string `xml:"version"`
			} `xml:"dependency"`
		} `xml:"dependencies"`
	} `xml:"dependencyManagement"`
}

func parsePOM(data []byte) []Dependency {
	var p pomProject
	if err := xml.Unmarshal(data, &p); err != nil {
		return nil
	}
	var out []Dependency
	seen := map[string]bool{}
	emit := func(g, a, v string) {
		if g == "" || a == "" {
			return
		}
		key := g + ":" + a
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, Dependency{Ecosystem: EcoMaven, Name: key, Version: v})
	}
	for _, d := range p.Dependencies.Items {
		emit(d.GroupID, d.ArtifactID, d.Version)
	}
	for _, d := range p.DepMgmt.Dependencies.Items {
		emit(d.GroupID, d.ArtifactID, d.Version)
	}
	return out
}

func parseRequirements(s string) []Dependency {
	var out []Dependency
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "-") {
			continue
		}
		// Strip environment markers.
		if i := strings.Index(t, ";"); i >= 0 {
			t = t[:i]
		}
		name, ver := t, ""
		if i := strings.Index(t, "=="); i >= 0 {
			name, ver = strings.TrimSpace(t[:i]), strings.TrimSpace(t[i+2:])
		} else if i := strings.IndexAny(t, "<>=!"); i >= 0 {
			name = strings.TrimSpace(t[:i])
		}
		if name == "" {
			continue
		}
		out = append(out, Dependency{Ecosystem: EcoPip, Name: name, Version: ver})
	}
	return out
}
