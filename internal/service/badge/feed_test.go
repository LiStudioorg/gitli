// SPDX-License-Identifier: AGPL-3.0
package badge

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

func sampleItems() []FeedItem {
	return []FeedItem{
		{
			Title:       "Fix <bug> & improve",
			Link:        "https://example.com/commit/1",
			Description: "A fix for <stuff>",
			GUID:        "commit-1",
			PubDate:     time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
		},
	}
}

func TestRSSWellFormed(t *testing.T) {
	out, err := RSS("My Feed", "https://example.com", "desc", sampleItems())
	if err != nil {
		t.Fatalf("RSS error: %v", err)
	}
	if !strings.HasPrefix(string(out), xml.Header) {
		t.Fatalf("RSS missing xml.Header: %s", out)
	}
	var doc struct {
		XMLName xml.Name `xml:"rss"`
		Version string   `xml:"version,attr"`
		Channel struct {
			Items []struct {
				Title string `xml:"title"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("RSS not well-formed: %v\n%s", err, out)
	}
	if doc.Version != "2.0" {
		t.Fatalf("RSS version = %q, want 2.0", doc.Version)
	}
	if len(doc.Channel.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(doc.Channel.Items))
	}
	if doc.Channel.Items[0].Title != "Fix <bug> & improve" {
		t.Fatalf("title not decoded correctly: %q", doc.Channel.Items[0].Title)
	}
	if strings.Contains(string(out), "Fix <bug>") {
		t.Fatalf("RSS emitted unescaped title: %s", out)
	}
}

func TestAtomWellFormed(t *testing.T) {
	out, err := Atom("My Feed", "https://example.com", sampleItems())
	if err != nil {
		t.Fatalf("Atom error: %v", err)
	}
	if !strings.HasPrefix(string(out), xml.Header) {
		t.Fatalf("Atom missing xml.Header: %s", out)
	}
	var doc struct {
		XMLName xml.Name `xml:"feed"`
		Title   string   `xml:"title"`
		Entries []struct {
			Title string `xml:"title"`
		} `xml:"entry"`
	}
	if err := xml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("Atom not well-formed: %v\n%s", err, out)
	}
	if len(doc.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(doc.Entries))
	}
	if doc.Entries[0].Title != "Fix <bug> & improve" {
		t.Fatalf("title not decoded correctly: %q", doc.Entries[0].Title)
	}
	if strings.Contains(string(out), "Fix <bug>") {
		t.Fatalf("Atom emitted unescaped title: %s", out)
	}
}

func TestAtomEmptyItems(t *testing.T) {
	out, err := Atom("Empty", "https://example.com", nil)
	if err != nil {
		t.Fatalf("Atom error: %v", err)
	}
	var doc struct {
		XMLName xml.Name `xml:"feed"`
	}
	if err := xml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("Atom not well-formed: %v\n%s", err, out)
	}
}
