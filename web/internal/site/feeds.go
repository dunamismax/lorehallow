package site

import (
	"encoding/xml"
	"fmt"
	"time"
)

// RSS 2.0 and Atom feeds for the devlog, and the sitemap. All written with
// encoding/xml so every value is escaped.

type rssDoc struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Atom    string     `xml:"xmlns:atom,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Self        atomLink  `xml:"atom:link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	LastBuild   string    `xml:"lastBuildDate,omitempty"`
	Items       []rssItem `xml:"item"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr,omitempty"`
}

type rssItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	GUID        rssGUID  `xml:"guid"`
	PubDate     string   `xml:"pubDate"`
	Description string   `xml:"description"`
	Categories  []string `xml:"category"`
}

type rssGUID struct {
	Value       string `xml:",chardata"`
	IsPermaLink bool   `xml:"isPermaLink,attr"`
}

func rssFeed(base string, posts []*Post) ([]byte, error) {
	ch := rssChannel{
		Title:       "Lorehallow devlog",
		Link:        base + "/devlog/",
		Self:        atomLink{Href: base + "/devlog/rss.xml", Rel: "self", Type: "application/rss+xml"},
		Description: "Notes from building Lorehallow, a small pixel-art MMO with a Go server, in the open.",
		Language:    "en",
		Items:       []rssItem{},
	}
	if len(posts) > 0 {
		ch.LastBuild = posts[0].Date.Format(time.RFC1123Z)
	}
	for _, p := range posts {
		ch.Items = append(ch.Items, rssItem{
			Title: p.Title, Link: base + p.URL, GUID: rssGUID{base + p.URL, true},
			PubDate: p.Date.Format(time.RFC1123Z), Description: string(p.Body), Categories: p.Tags,
		})
	}
	return marshalXML(rssDoc{Version: "2.0", Atom: "http://www.w3.org/2005/Atom", Channel: ch})
}

type atomFeed struct {
	XMLName xml.Name    `xml:"http://www.w3.org/2005/Atom feed"`
	Title   string      `xml:"title"`
	ID      string      `xml:"id"`
	Updated string      `xml:"updated"`
	Links   []atomLink  `xml:"link"`
	Author  atomAuthor  `xml:"author"`
	Entries []atomEntry `xml:"entry"`
}

type atomAuthor struct {
	Name string `xml:"name"`
	URI  string `xml:"uri"`
}

type atomEntry struct {
	Title     string      `xml:"title"`
	ID        string      `xml:"id"`
	Link      atomLink    `xml:"link"`
	Published string      `xml:"published"`
	Updated   string      `xml:"updated"`
	Summary   string      `xml:"summary"`
	Content   atomContent `xml:"content"`
	Category  []atomCat   `xml:"category"`
}

type atomContent struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type atomCat struct {
	Term string `xml:"term,attr"`
}

func atomFeedXML(base string, posts []*Post, built time.Time) ([]byte, error) {
	updated := built
	if len(posts) > 0 {
		updated = posts[0].Date
	}
	f := atomFeed{
		Title: "Lorehallow devlog", ID: base + "/devlog/", Updated: updated.UTC().Format(time.RFC3339),
		Links: []atomLink{
			{Href: base + "/devlog/atom.xml", Rel: "self", Type: "application/atom+xml"},
			{Href: base + "/devlog/", Rel: "alternate", Type: "text/html"},
		},
		Author:  atomAuthor{Name: "Lorehallow", URI: base + "/"},
		Entries: []atomEntry{},
	}
	for _, p := range posts {
		e := atomEntry{
			Title: p.Title, ID: base + p.URL, Link: atomLink{Href: base + p.URL, Rel: "alternate", Type: "text/html"},
			Published: p.Date.UTC().Format(time.RFC3339), Updated: p.Date.UTC().Format(time.RFC3339),
			Summary: p.Summary, Content: atomContent{"html", string(p.Body)},
		}
		for _, t := range p.Tags {
			e.Category = append(e.Category, atomCat{t})
		}
		f.Entries = append(f.Entries, e)
	}
	return marshalXML(f)
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"http://www.sitemaps.org/schemas/sitemap/0.9 urlset"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

func sitemap(base string, pages []sitemapURL) ([]byte, error) {
	for i := range pages {
		pages[i].Loc = base + pages[i].Loc
	}
	return marshalXML(sitemapURLSet{URLs: pages})
}

func marshalXML(v any) ([]byte, error) {
	out, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("xml: %w", err)
	}
	return append([]byte(xml.Header), append(out, '\n')...), nil
}
