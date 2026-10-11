// SPDX-License-Identifier: AGPL-3.0
package git

import (
	"context"
	"fmt"
	"strings"
)

// GraphNode is a commit with its parent hashes for graph rendering.
type GraphNode struct {
	Hash      string   `json:"hash"`
	ShortHash string   `json:"short"`
	Subject   string   `json:"subject"`
	Author    string   `json:"author"`
	Date      string   `json:"date"`
	Refs      []string `json:"refs"`
	Parents   []string `json:"parents"`
}

// Graph returns up to limit commits reachable from ref, each annotated with
// the refs (branches/tags) pointing at it, suitable for a commit graph.
func (s *Service) Graph(ctx context.Context, path, ref string, limit int) ([]GraphNode, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if ref == "" {
		ref = "HEAD"
	}
	const fs = "%H%x1f%h%x1f%an%x1f%aI%x1f%P%x1f%D%x1f%s%x1e"
	out, err := s.runner.Run(ctx, path, "log", fmt.Sprintf("-%d", limit), "--format="+fs, ref)
	if err != nil {
		return nil, err
	}
	var nodes []GraphNode
	for _, rec := range strings.Split(out, "\x1e") {
		rec = strings.Trim(rec, "\n")
		if rec == "" {
			continue
		}
		f := strings.Split(rec, "\x1f")
		if len(f) < 7 {
			continue
		}
		n := GraphNode{
			Hash:      f[0],
			ShortHash: f[1],
			Author:    f[2],
			Date:      f[3],
			Subject:   f[6],
		}
		if strings.TrimSpace(f[4]) != "" {
			n.Parents = strings.Fields(f[4])
		}
		if strings.TrimSpace(f[5]) != "" {
			for _, r := range strings.Split(f[5], ",") {
				if r = strings.TrimSpace(r); r != "" {
					n.Refs = append(n.Refs, r)
				}
			}
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}
