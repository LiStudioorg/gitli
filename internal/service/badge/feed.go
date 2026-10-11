// SPDX-License-Identifier: AGPL-3.0
package badge

import (
	"encoding/xml"
	"time"
)

// FeedItem is a single entry in an RSS or Atom feed.
type FeedItem struct {
	Title       string
	Link        string
	Description string
	GUID        string
	PubDate     time.Time
}

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description,omitempty"`
	GUID        string `xml:"guid,omitempty"`
	PubDate     string `xml:"pubDate,omitempty"`
}

// RSS renders a valid RSS 2.0 document for the given feed.
func RSS(title, link, description string, items []FeedItem) ([]byte, error) {
	feed := rssFeed{
		Version: "2.0",
		Channel: rssChannel{
			Title:       title,
			Link:        link,
			Description: description,
		},
	}
	for _, it := range items {
		feed.Channel.Items = append(feed.Channel.Items, rssItem{
			Title:       it.Title,
			Link:        it.Link,
			Description: it.Description,
			GUID:        it.GUID,
			PubDate:     it.PubDate.UTC().Format(time.RFC1123Z),
		})
	}
	body, err := xml.Marshal(feed)
	if err != nil {
		return nil, err
	}
	out := append([]byte(xml.Header), body...)
	return out, nil
}

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Xmlns   string      `xml:"xmlns,attr"`
	Title   string      `xml:"title"`
	Link    atomLink    `xml:"link"`
	Updated string      `xml:"updated"`
	Entries []atomEntry `xml:"entry"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
}

type atomEntry struct {
	Title   string   `xml:"title"`
	Link    atomLink `xml:"link"`
	ID      string   `xml:"id"`
	Updated string   `xml:"updated"`
	Summary string   `xml:"summary,omitempty"`
}

// Atom renders a valid Atom 1.0 document for the given feed.
func Atom(title, link string, items []FeedItem) ([]byte, error) {
	updated := time.Now().UTC()
	if len(items) > 0 {
		updated = items[0].PubDate.UTC()
	}
	feed := atomFeed{
		Xmlns:   "http://www.w3.org/2005/Atom",
		Title:   title,
		Link:    atomLink{Href: link, Rel: "alternate"},
		Updated: updated.Format(time.RFC3339),
	}
	for _, it := range items {
		id := it.GUID
		if id == "" {
			id = it.Link
		}
		feed.Entries = append(feed.Entries, atomEntry{
			Title:   it.Title,
			Link:    atomLink{Href: it.Link, Rel: "alternate"},
			ID:      id,
			Updated: it.PubDate.UTC().Format(time.RFC3339),
			Summary: it.Description,
		})
	}
	body, err := xml.Marshal(feed)
	if err != nil {
		return nil, err
	}
	out := append([]byte(xml.Header), body...)
	return out, nil
}
