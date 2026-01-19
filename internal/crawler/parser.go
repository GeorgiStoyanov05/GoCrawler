package crawler

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
)

func ExtractLinks(htmlBody []byte) ([]string, error) {
	doc, err := html.Parse(bytes.NewReader(htmlBody))
	if err != nil {
		return nil, err
	}

	var links []string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" && strings.TrimSpace(a.Val) != "" {
					links = append(links, strings.TrimSpace(a.Val))
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)
	return links, nil
}

func ExtractImages(htmlBody []byte) ([]string, error) {
	doc, err := html.Parse(bytes.NewReader(htmlBody))
	if err != nil {
		return nil, err
	}

	var images []string

	addSrcset := func(v string) {
		for _, u := range parseSrcset(v) {
			if u != "" {
				images = append(images, u)
			}
		}
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "img":
				for _, a := range n.Attr {
					key := strings.ToLower(a.Key)
					val := strings.TrimSpace(a.Val)
					if val == "" {
						continue
					}
					switch key {
					case "src", "data-src", "data-original", "data-lazy-src", "data-url":
						images = append(images, val)
					case "srcset", "data-srcset":
						addSrcset(val)
					}
				}

			case "source":
				for _, a := range n.Attr {
					if strings.ToLower(a.Key) == "srcset" {
						addSrcset(strings.TrimSpace(a.Val))
					}
				}

			case "image":
				for _, a := range n.Attr {
					key := strings.ToLower(a.Key)
					val := strings.TrimSpace(a.Val)
					if val == "" {
						continue
					}
					if key == "href" || key == "xlink:href" {
						images = append(images, val)
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)
	return images, nil
}

func parseSrcset(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		fields := strings.Fields(p)
		if len(fields) > 0 {
			out = append(out, fields[0])
		}
	}
	return out
}
