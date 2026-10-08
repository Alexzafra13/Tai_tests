package lawtext

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/alexzafra13/tai_tests/internal/content"
)

// EU regulations come from the consolidated texts of EUR-Lex, the
// Publications Office's official consolidation. They are named by their
// CELEX number, e.g. 32016R0679 for the GDPR.

var celexRe = regexp.MustCompile(`^3\d{4}R\d{4}$`)

// IsCELEX tells an EU regulation's CELEX number from a BOE reference.
func IsCELEX(ref string) bool { return celexRe.MatchString(ref) }

const eurLexBase = "https://eur-lex.europa.eu/legal-content/ES/"

// EURLexVersionsURL is the page listing a regulation's consolidated versions.
func EURLexVersionsURL(celex string) string { return eurLexBase + "ALL/?uri=CELEX:" + celex }

// EURLexTextURL is the consolidated text as of version (YYYYMMDD).
func EURLexTextURL(celex, version string) string {
	return eurLexBase + "TXT/HTML/?uri=CELEX:0" + celex[1:] + "-" + version
}

// EURLexPageURL is the regulation's page on EUR-Lex, for people.
func EURLexPageURL(celex string) string { return eurLexBase + "TXT/?uri=CELEX:" + celex }

// LatestEURLexVersion finds the latest consolidated version (YYYYMMDD) in
// the page EURLexVersionsURL serves.
func LatestEURLexVersion(page []byte, celex string) (string, error) {
	var version string
	for _, v := range regexp.MustCompile(`0`+celex[1:]+`-(\d{8})`).FindAllSubmatch(page, -1) {
		if d := string(v[1]); d > version {
			version = d
		}
	}
	if version == "" {
		return "", errors.New("eur-lex: no consolidated version")
	}
	return version, nil
}

// VersionDay turns a YYYYMMDD version into YYYY-MM-DD.
func VersionDay(version string) (string, error) {
	t, err := time.Parse("20060102", version)
	if err != nil {
		return "", fmt.Errorf("version %q", version)
	}
	return t.Format("2006-01-02"), nil
}

// node is a minimal HTML tree: EUR-Lex serves well-formed XHTML.
type node struct {
	tag      string
	id       string
	class    string
	text     string // character data, for text nodes (tag == "")
	children []*node
}

func parseTree(page []byte) (*node, error) {
	dec := xml.NewDecoder(bytes.NewReader(page))
	dec.Strict = false
	dec.AutoClose = xml.HTMLAutoClose
	dec.Entity = xml.HTMLEntity
	root := &node{}
	stack := []*node{root}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("eur-lex: %w", err)
		}
		top := stack[len(stack)-1]
		switch t := tok.(type) {
		case xml.StartElement:
			n := &node{tag: t.Name.Local}
			for _, a := range t.Attr {
				switch a.Name.Local {
				case "id":
					n.id = a.Value
				case "class":
					n.class = a.Value
				}
			}
			top.children = append(top.children, n)
			stack = append(stack, n)
		case xml.EndElement:
			// Unbalanced end tags close up to their element, if open.
			for i := len(stack) - 1; i > 0; i-- {
				if stack[i].tag == t.Name.Local {
					stack = stack[:i]
					break
				}
			}
		case xml.CharData:
			top.children = append(top.children, &node{text: string(t)})
		}
	}
	return root, nil
}

func (n *node) find(match func(*node) bool) *node {
	if match(n) {
		return n
	}
	for _, c := range n.children {
		if f := c.find(match); f != nil {
			return f
		}
	}
	return nil
}

// plain is the node's text without the consolidation's markers (▼M2, ►C2,
// ◄) and footnote calls.
func (n *node) plain() string {
	var b strings.Builder
	var walk func(*node)
	walk = func(n *node) {
		if n.tag == "" {
			b.WriteString(n.text)
			return
		}
		if n.tag == "a" && (strings.HasPrefix(n.id, "src.") || n.find(func(c *node) bool { return c.tag == "" && strings.ContainsAny(c.text, "▼►") }) != nil) {
			return
		}
		if n.tag == "span" && isMarker(n) {
			return
		}
		for _, c := range n.children {
			walk(c)
		}
	}
	walk(n)
	s := strings.NewReplacer("◄", " ", "▼", " ", "►", " ").Replace(b.String())
	s = emptyCall.ReplaceAllString(s, "") // what a footnote call leaves: "Consejo (1)" → "Consejo"
	return strings.TrimSpace(spaces.ReplaceAllString(s, " "))
}

// isMarker tells the end mark of a change, "◄", with the space before it.
func isMarker(n *node) bool {
	var b strings.Builder
	n.find(func(c *node) bool {
		b.WriteString(c.text)
		return false
	})
	t := b.String()
	return strings.Contains(t, "◄") && strings.Trim(t, " \t\r\n\u00a0◄") == ""
}

var emptyCall = regexp.MustCompile(`[\s\x{00a0}]*\([\s\x{00a0}]*\)`)

var (
	divisionID = regexp.MustCompile(`^cpt_[^.]+(\.sct_\d+)?$|^tis_[^.]+$`)
	articleID  = regexp.MustCompile(`^art_\w+$`)
	annexID    = regexp.MustCompile(`^anx_\w+$`)
)

// ParseEURLexText reads a consolidated text served by EURLexTextURL into
// sections: chapters and sections as headings, articles and annexes as
// articles. Recitals are not part of consolidated texts. It also returns
// the title as consolidated (corrections included).
func ParseEURLexText(page []byte) (string, []content.LawSection, error) {
	root, err := parseTree(page)
	if err != nil {
		return "", nil, err
	}
	title, err := consolidatedTitle(root)
	if err != nil {
		return "", nil, err
	}
	var out []content.LawSection
	var walk func(*node)
	walk = func(n *node) {
		switch {
		case n.tag == "div" && divisionID.MatchString(n.id):
			var titles []string
			for _, c := range n.children {
				if strings.HasPrefix(c.class, "title-division") {
					titles = append(titles, c.plain())
				}
			}
			out = append(out, content.LawSection{
				ID: sectionID(n.id), Kind: content.SectionHeading, Level: strings.Count(n.id, ".") + 1,
				Title: strings.Join(titles, ". "),
			})
		case n.tag == "div" && articleID.MatchString(n.id):
			out = append(out, readEUArticle(n))
			return
		case n.tag == "div" && annexID.MatchString(n.id):
			out = append(out, readEUAnnex(n))
			return
		}
		for _, c := range n.children {
			walk(c)
		}
	}
	walk(root)
	if len(out) == 0 {
		return "", nil, errors.New("eur-lex: no articles")
	}
	for _, s := range out {
		if s.Title == "" {
			return "", nil, fmt.Errorf("eur-lex: section %s has no title", s.ID)
		}
	}
	return title, out, nil
}

var euNumber = regexp.MustCompile(`^REGLAMENTO \(UE\) (?:N\s*o\s+)?(\d+/\d+) DEL PARLAMENTO EUROPEO Y DEL CONSEJO$`)

// consolidatedTitle writes the title lines ("REGLAMENTO (UE) 2016/679 DEL
// PARLAMENTO EUROPEO Y DEL CONSEJO", "de 27 de abril de 2016", "relativo
// a…") the way the BOE names EU acts.
func consolidatedTitle(root *node) (string, error) {
	main := root.find(func(n *node) bool { return n.class == "eli-main-title" })
	if main == nil {
		return "", errors.New("eur-lex: no title")
	}
	var lines []string
	main.find(func(n *node) bool {
		if strings.HasPrefix(n.class, "title-doc-") {
			if t := n.plain(); t != "(Texto pertinente a efectos del EEE)" {
				lines = append(lines, t)
			}
		}
		return false
	})
	if len(lines) < 3 {
		return "", fmt.Errorf("eur-lex: title %q", lines)
	}
	m := euNumber.FindStringSubmatch(lines[0])
	if m == nil {
		return "", fmt.Errorf("eur-lex: title %q", lines[0])
	}
	name := "Reglamento (UE) " + m[1]
	if strings.Contains(lines[0], " N") {
		name = "Reglamento (UE) n.º " + m[1]
	}
	return name + " del Parlamento Europeo y del Consejo, " + lines[1] + ", " + strings.Join(lines[2:], " "), nil
}

// sectionID keeps ids usable in a URL fragment: cpt_III.sct_1 → cpt_III-sct_1.
func sectionID(id string) string { return strings.ReplaceAll(id, ".", "-") }

func readEUArticle(n *node) content.LawSection {
	var title, name string
	var paras []string
	for _, c := range n.children {
		switch {
		case c.class == "title-article-norm":
			title = c.plain()
		case c.class == "eli-title":
			name = c.plain()
		default:
			paras = append(paras, blockText(c)...)
		}
	}
	if name != "" {
		title += ". " + name
	}
	return content.LawSection{ID: sectionID(n.id), Kind: content.SectionArticle, Title: title, Body: strings.Join(paras, "\n\n")}
}

func readEUAnnex(n *node) content.LawSection {
	var titles, paras []string
	for _, c := range n.children {
		switch {
		case strings.HasPrefix(c.class, "title-annex"), len(paras) == 0 && strings.HasPrefix(c.class, "title-gr-seq"):
			titles = append(titles, c.plain())
		default:
			paras = append(paras, blockText(c)...)
		}
	}
	return content.LawSection{ID: sectionID(n.id), Kind: content.SectionArticle, Title: strings.Join(titles, ". "), Body: strings.Join(paras, "\n\n")}
}

// blockText turns a piece of an article into paragraphs. A numbered
// paragraph keeps its number ("1. El presente…"); a list item keeps its
// letter or number and nested items follow it.
func blockText(n *node) []string {
	switch {
	case n.tag == "":
		return nil
	case n.class == "modref" || n.tag == "hr" || n.tag == "br":
		return nil
	case strings.Contains(n.class, "grid-container"):
		var label string
		var out []string
		for _, c := range n.children {
			switch {
			case strings.Contains(c.class, "grid-list-column-1"):
				label = c.plain()
			case strings.Contains(c.class, "grid-list-column-2"):
				out = append(out, blockChildren(c)...)
			}
		}
		if len(out) > 0 && label != "" {
			out[0] = label + " " + out[0]
		}
		return out
	case n.tag == "div" && n.class == "norm":
		// A numbered paragraph: <span class="no-parag">1. </span> then the text,
		// inline or in nested blocks.
		var label string
		var out []string
		var inline []*node
		for _, c := range n.children {
			switch {
			case c.class == "no-parag":
				label = c.plain()
			case c.tag == "div" || c.tag == "p" || c.tag == "table":
				if c.class == "norm inline-element" {
					inline = append(inline, c)
					continue
				}
				out = append(out, blockText(c)...)
			}
		}
		var lead []string
		for _, c := range inline {
			lead = append(lead, blockChildren(c)...)
		}
		out = append(lead, out...)
		if len(out) > 0 && label != "" {
			out[0] = label + " " + out[0]
		}
		return out
	case n.tag == "table" && n.find(func(c *node) bool { return c.class == "dlist-term" }) != nil:
		// A numbered definition laid out as a two-cell table.
		var cells []*node
		n.find(func(c *node) bool {
			if c.tag == "td" {
				cells = append(cells, c)
			}
			return false
		})
		if len(cells) != 2 {
			return []string{treeTableText(n)}
		}
		out := blockChildren(cells[1])
		if len(out) > 0 {
			out[0] = cells[0].plain() + " " + out[0]
		}
		return out
	case n.tag == "table":
		return []string{treeTableText(n)}
	case n.tag == "p" || n.tag == "span":
		if t := n.plain(); t != "" {
			return []string{t}
		}
		return nil
	case n.tag == "dl":
		var out []string
		var term string
		for _, c := range n.children {
			switch c.tag {
			case "dt":
				term = c.plain()
			case "dd":
				out = append(out, strings.TrimSpace(term+" "+strings.Join(blockChildren(c), " ")))
			}
		}
		return out
	default:
		return blockChildren(n)
	}
}

// blockChildren reads a container: its child blocks, or its own text when
// it has only inline content.
func blockChildren(n *node) []string {
	block := slices.ContainsFunc(n.children, func(c *node) bool {
		return c.tag == "p" || c.tag == "div" || c.tag == "table" || c.tag == "dl"
	})
	if !block {
		if t := n.plain(); t != "" {
			return []string{t}
		}
		return nil
	}
	var out []string
	for _, c := range n.children {
		out = append(out, blockText(c)...)
	}
	return out
}

func treeTableText(n *node) string {
	var rows []string
	var walk func(*node)
	walk = func(n *node) {
		if n.tag == "tr" {
			var cells []string
			for _, c := range n.children {
				if c.tag == "td" || c.tag == "th" {
					cells = append(cells, c.plain())
				}
			}
			rows = append(rows, strings.Join(cells, " | "))
			return
		}
		for _, c := range n.children {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(rows, "\n")
}
