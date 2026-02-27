package browser

import (
	"fmt"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

var skips = []string{
	"script", "style", "noscript", "svg", "iframe", "canvas", "video", "audio", "nav", "header", "footer", "aside", "form", "button", "input", "select", "textarea", "label", "link", "meta",
}

var blocks = []string{
	"div", "section", "article", "main", "p", "ul", "ol", "li", "blockquote", "pre", "table", "tr", "td", "th",
}

func extract(raw, title, url string) (string, error) {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("html.Parse: %w", err)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "---\ntitle: %s\nurl: %s\n---\n\n", title, url)
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			sb.WriteString(n.Data)
			return

		case html.ElementNode:
			tag := strings.ToLower(n.Data)

			if slices.Contains(skips, tag) {
				return
			}

			switch tag {
			case "h1", "h2", "h3", "h4", "h5", "h6":
				level := int(tag[1] - '0')
				sb.WriteString("\n" + strings.Repeat("#", level) + " ")
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
				sb.WriteString("\n")
				return

			case "a":
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
				return

			case "strong", "b":
				sb.WriteString("**")
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
				sb.WriteString("**")
				return

			case "em", "i":
				sb.WriteString("*")
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
				sb.WriteString("*")
				return

			case "br":
				sb.WriteString("\n")
				return

			case "li":
				sb.WriteString("\n- ")
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
				return
			}

			if slices.Contains(blocks, tag) {
				sb.WriteString("\n")
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
				sb.WriteString("\n")
				return
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return strings.TrimSpace(collapse(sb.String())), nil
}


// * remove empty line like [\n]{2,} to be [\n]{2}
func collapse(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	blanks := 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			blanks++
			if blanks <= 1 {
				out = append(out, "")
			}
		} else {
			blanks = 0
			out = append(out, strings.TrimSpace(l))
		}
	}
	return strings.Join(out, "\n")
}
