// Package lawtext reads the consolidated legislation of the BOE open data
// API (https://www.boe.es/datosabiertos/api/legislacion-consolidada) into
// study texts.
package lawtext

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/alexzafra13/tai_tests/internal/content"
)

const apiBase = "https://www.boe.es/datosabiertos/api/legislacion-consolidada/id/"

// MetadataURL and TextURL are the API endpoints of a law (ask for JSON and
// XML respectively).
func MetadataURL(ref string) string { return apiBase + ref + "/metadatos" }
func TextURL(ref string) string     { return apiBase + ref + "/texto" }

// PageURL is the law's page on boe.es, for people.
func PageURL(ref string) string { return "https://www.boe.es/buscar/act.php?id=" + ref }

type Metadata struct {
	Title       string
	VersionDate string // YYYY-MM-DD
}

// ParseMetadata reads the JSON answer of the metadata endpoint.
func ParseMetadata(b []byte) (Metadata, error) {
	var r struct {
		Data []struct {
			Titulo              string `json:"titulo"`
			FechaActualizacion  string `json:"fecha_actualizacion"` // 20260520T074424Z
			EstadoConsolidacion struct {
				Texto string `json:"texto"`
			} `json:"estado_consolidacion"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return Metadata{}, fmt.Errorf("metadata: %w", err)
	}
	if len(r.Data) == 0 || r.Data[0].Titulo == "" {
		return Metadata{}, errors.New("metadata: no title")
	}
	d := r.Data[0]
	if d.EstadoConsolidacion.Texto != "Finalizado" {
		return Metadata{}, fmt.Errorf("metadata: consolidation state %q, not finished", d.EstadoConsolidacion.Texto)
	}
	t, err := time.Parse("20060102", strings.SplitN(d.FechaActualizacion, "T", 2)[0])
	if err != nil {
		return Metadata{}, fmt.Errorf("metadata: fecha_actualizacion %q", d.FechaActualizacion)
	}
	return Metadata{Title: strings.TrimSuffix(d.Titulo, "."), VersionDate: t.Format("2006-01-02")}, nil
}

type textDoc struct {
	Blocks []block `xml:"data>texto>bloque"`
}

type block struct {
	ID   string `xml:"id,attr"`
	Tipo string `xml:"tipo,attr"`
	// Caducidad is the date a block stopped being part of the law, such as
	// a chapter division removed by a reform (YYYYMMDD).
	Caducidad string    `xml:"fecha_caducidad,attr"`
	Versions  []version `xml:"version"`
}

type version struct {
	Vigencia string    `xml:"fecha_vigencia,attr"` // YYYYMMDD
	Items    []element `xml:",any"`
}

type element struct {
	XMLName xml.Name
	Class   string `xml:"class,attr"`
	Inner   string `xml:",innerxml"`
}

// headingLevels maps the class of a heading's first paragraph to its depth.
var headingLevels = map[string]int{
	"libro": 1, "titulo": 1, "titulo_num": 1, "anexo": 1, "anexo_num": 1,
	"capitulo": 2, "capitulo_num": 2,
	"seccion":    3,
	"subseccion": 4,
}

// ParseText reads the XML answer of the text endpoint. Each block takes the
// wording in force on today; a later wording already published goes to
// Upcoming. Expired blocks, signatures and editorial notes are left out.
func ParseText(b []byte, today time.Time) ([]content.LawSection, error) {
	var doc textDoc
	if err := xml.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("text: %w", err)
	}
	day := today.Format("20060102")
	var out []content.LawSection
	for _, bl := range doc.Blocks {
		if bl.Caducidad != "" && bl.Caducidad <= day {
			continue
		}
		var kind content.LawSectionKind
		switch bl.Tipo {
		case "encabezado":
			kind = content.SectionHeading
		case "precepto", "anexo":
			kind = content.SectionArticle
		case "preambulo":
			kind = content.SectionText
		default: // firma, nota_inicial
			continue
		}
		var current, upcoming *version
		for i := range bl.Versions {
			v := &bl.Versions[i]
			if v.Vigencia <= day {
				current = v
			} else {
				upcoming = v
			}
		}
		sec := content.LawSection{ID: bl.ID, Kind: kind}
		if current != nil {
			sec.Level, sec.Title, sec.Body, sec.Notes = readVersion(kind, *current)
		}
		if upcoming != nil {
			lvl, title, body, _ := readVersion(kind, *upcoming)
			date, err := time.Parse("20060102", upcoming.Vigencia)
			if err != nil {
				return nil, fmt.Errorf("text: block %s: date %q", bl.ID, upcoming.Vigencia)
			}
			sec.Upcoming = &content.LawChange{Date: date.Format("2006-01-02"), Title: title, Body: body}
			if current == nil {
				sec.Level, sec.Title = lvl, title
			}
		}
		// Some annexes are headings with text: show them as text sections.
		if kind == content.SectionHeading && (sec.Body != "" || sec.Upcoming != nil && sec.Upcoming.Body != "") {
			sec.Kind, sec.Level = content.SectionArticle, 0
		}
		if kind == content.SectionText && sec.Title == "" {
			sec.Title = "Preámbulo"
		}
		if sec.Title == "" && sec.Body == "" && sec.Upcoming == nil {
			continue
		}
		out = append(out, sec)
	}
	if len(out) == 0 {
		return nil, errors.New("text: no blocks")
	}
	return out, nil
}

// readVersion splits a version into title (heading lines or the article's
// name), body paragraphs and the BOE's notes on changes.
func readVersion(kind content.LawSectionKind, v version) (level int, title, body, notes string) {
	var titles, paras, noteLines []string
	for _, el := range v.Items {
		switch el.XMLName.Local {
		case "p":
			text := cleanText(el.Inner)
			if text == "" {
				continue
			}
			switch {
			case kind == content.SectionHeading && len(paras) == 0 && isHeadingClass(el.Class):
				if level == 0 {
					level = headingLevels[el.Class]
				}
				titles = append(titles, text)
			case kind == content.SectionArticle && len(titles) == 0 && len(paras) == 0 &&
				(el.Class == "articulo" || strings.HasPrefix(el.Class, "anexo")):
				titles = append(titles, text)
			case kind == content.SectionArticle && strings.HasPrefix(el.Class, "anexo_tit"):
				titles = append(titles, text)
			case el.Class == "imagen":
				paras = append(paras, "[Imagen en el texto original del BOE]")
			default:
				paras = append(paras, text)
			}
		case "blockquote":
			for _, line := range strings.Split(cleanText(strings.ReplaceAll(el.Inner, "</p>", "</p>\n")), "\n") {
				if line = strings.TrimSpace(line); line != "" {
					noteLines = append(noteLines, line)
				}
			}
		case "table":
			paras = append(paras, tableText(el.Inner))
		}
	}
	if kind == content.SectionHeading && level == 0 {
		level = 1
	}
	return level, strings.Join(titles, ". "), strings.Join(paras, "\n\n"), strings.Join(noteLines, "\n")
}

// isHeadingClass tells the lines naming a heading ("TÍTULO I", "De los
// derechos…") from text some headings carry, such as annexes.
func isHeadingClass(class string) bool {
	if _, ok := headingLevels[class]; ok {
		return true
	}
	return strings.HasSuffix(class, "_tit") || strings.HasSuffix(class, "_num")
}

var (
	tags   = regexp.MustCompile(`<[^>]*>`)
	spaces = regexp.MustCompile(`[ \t\r\n\x{00a0}]+`)
	rowRe  = regexp.MustCompile(`(?s)<tr[^>]*>(.*?)</tr>`)
	cellRe = regexp.MustCompile(`(?s)<t[hd][^>]*>(.*?)</t[hd]>`)
)

func cleanText(inner string) string {
	return strings.TrimSpace(spaces.ReplaceAllString(html.UnescapeString(tags.ReplaceAllString(inner, " ")), " "))
}

// tableText writes a table one row per line, cells separated by " | ".
func tableText(inner string) string {
	var rows []string
	for _, r := range rowRe.FindAllStringSubmatch(inner, -1) {
		var cells []string
		for _, c := range cellRe.FindAllStringSubmatch(r[1], -1) {
			cells = append(cells, cleanText(c[1]))
		}
		rows = append(rows, strings.Join(cells, " | "))
	}
	return strings.Join(rows, "\n")
}

var lawNumber = regexp.MustCompile(`^(Ley Orgánica|Ley|Real Decreto Legislativo|Real Decreto-ley|Real Decreto) (\d+/\d{4})`)

// Aliases are the names exam questions use for a law: its number with and
// without the kind ("Ley 39/2015", "39/2015"), plus any given.
func Aliases(title string, extra []string) []string {
	var out []string
	if m := lawNumber.FindStringSubmatch(title); m != nil {
		out = append(out, m[1]+" "+m[2], m[2])
	}
	for _, e := range extra {
		if e = strings.TrimSpace(e); e != "" {
			out = append(out, e)
		}
	}
	return out
}
