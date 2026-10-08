package lawtext

import (
	"strings"
	"testing"
	"time"

	"github.com/alexzafra13/tai_tests/internal/content"
)

// Shortened from the BOE answer for the Constitution and the dependency law.
const sampleText = `<?xml version="1.0" encoding="utf-8"?>
<response><status><code>200</code></status><data><texto>
  <bloque id="preambulo" tipo="preambulo" titulo="preambulo">
    <version id_norma="X" fecha_publicacion="19781229" fecha_vigencia="19781229">
      <p class="parrafo_2">DON JUAN CARLOS I, REY DE ESPAÑA,</p>
    </version>
  </bloque>
  <bloque id="ti" tipo="encabezado" titulo="TÍTULO I">
    <version id_norma="X" fecha_publicacion="19781229" fecha_vigencia="19781229">
      <p class="titulo_num">TÍTULO I</p>
      <p class="titulo_tit">De los derechos y deberes fundamentales</p>
    </version>
  </bloque>
  <bloque id="cprimero" tipo="encabezado" titulo="CAPÍTULO PRIMERO" fecha_caducidad="19850609">
    <version id_norma="X" fecha_publicacion="19781229" fecha_vigencia="19781229">
      <p class="capitulo_num">CAPÍTULO PRIMERO</p>
    </version>
  </bloque>
  <bloque id="a13" tipo="precepto" titulo="Artículo 13">
    <version id_norma="X" fecha_publicacion="19781229" fecha_vigencia="19781229">
      <p class="articulo">Artículo 13</p>
      <p class="parrafo">1. Los extranjeros gozarán   en España de las libertades públicas.</p>
    </version>
    <version id_norma="Y" fecha_publicacion="19920828" fecha_vigencia="19920828">
      <p class="articulo">Artículo 13</p>
      <p class="parrafo">1. Los extranjeros gozarán en España de las libertades públicas.</p>
      <p class="parrafo">2. Solamente los españoles serán titulares de los derechos.</p>
      <blockquote><p class="nota_pie">Se modifica el apartado 2. <a class="refPost">Ref. BOE-A-1992-20403</a></p></blockquote>
    </version>
  </bloque>
  <bloque id="a4-2" tipo="precepto" titulo="Artículo 4 bis">
    <version id_norma="Z" fecha_publicacion="20251022" fecha_vigencia="20261023">
      <p class="articulo">Artículo 4 bis. Derechos de las personas cuidadoras.</p>
      <p class="parrafo">1. Tendrán la consideración de personas cuidadoras.</p>
    </version>
  </bloque>
  <bloque id="ai" tipo="encabezado" titulo="ANEXO I">
    <version id_norma="X" fecha_publicacion="20220504" fecha_vigencia="20220505">
      <p class="anexo_num">ANEXO I</p>
      <p class="anexo_tit">Categorías de seguridad</p>
      <p class="parrafo">1. Fundamentos.</p>
      <table><tr><th><p>Nivel</p></th><th><p>Valor</p></th></tr><tr><td><p>Alto</p></td><td><p>3</p></td></tr></table>
    </version>
  </bloque>
  <bloque id="firma" tipo="firma" titulo="firma">
    <version id_norma="X" fecha_publicacion="19781229" fecha_vigencia="19781229"><p class="firma_rey">JUAN CARLOS</p></version>
  </bloque>
</texto></data></response>`

func TestParseText(t *testing.T) {
	secs, err := ParseText([]byte(sampleText), time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, s := range secs {
		ids = append(ids, s.ID)
	}
	// The expired chapter and the signature are left out.
	if strings.Join(ids, ",") != "preambulo,ti,a13,a4-2,ai" {
		t.Fatalf("sections %v", ids)
	}

	if s := secs[1]; s.Kind != content.SectionHeading || s.Level != 1 || s.Title != "TÍTULO I. De los derechos y deberes fundamentales" {
		t.Errorf("heading %+v", s)
	}
	a13 := secs[2]
	if a13.Title != "Artículo 13" || a13.Body != "1. Los extranjeros gozarán en España de las libertades públicas.\n\n2. Solamente los españoles serán titulares de los derechos." {
		t.Errorf("article in force %+v", a13)
	}
	if a13.Notes != "Se modifica el apartado 2. Ref. BOE-A-1992-20403" || a13.Upcoming != nil {
		t.Errorf("notes %q upcoming %v", a13.Notes, a13.Upcoming)
	}

	// Published but not in force yet: no body, the wording waits in Upcoming.
	bis := secs[3]
	if bis.Body != "" || bis.Upcoming == nil || bis.Upcoming.Date != "2026-10-23" ||
		bis.Title != "Artículo 4 bis. Derechos de las personas cuidadoras." {
		t.Errorf("upcoming article %+v", bis)
	}

	// An annex heading with text becomes a text section.
	if s := secs[4]; s.Kind != content.SectionArticle || s.Title != "ANEXO I. Categorías de seguridad" ||
		s.Body != "1. Fundamentos.\n\nNivel | Valor\nAlto | 3" {
		t.Errorf("annex %+v", s)
	}
}

func TestParseMetadata(t *testing.T) {
	m, err := ParseMetadata([]byte(`{"data":[{"titulo":"Ley 39/2015, de 1 de octubre.","fecha_actualizacion":"20260925T080815Z",
		"estado_consolidacion":{"codigo":"3","texto":"Finalizado"}}]}`))
	if err != nil || m.Title != "Ley 39/2015, de 1 de octubre" || m.VersionDate != "2026-09-25" {
		t.Fatalf("%+v %v", m, err)
	}
	if _, err := ParseMetadata([]byte(`{"data":[{"titulo":"x","fecha_actualizacion":"20260925T0","estado_consolidacion":{"texto":"En proceso"}}]}`)); err == nil {
		t.Error("accepted an unfinished consolidation")
	}
}

func TestAliases(t *testing.T) {
	got := Aliases("Ley Orgánica 3/2018, de 5 de diciembre", []string{"LOPDGDD", " "})
	if strings.Join(got, "|") != "Ley Orgánica 3/2018|3/2018|LOPDGDD" {
		t.Errorf("%q", got)
	}
}

func TestAliasesEURegulation(t *testing.T) {
	got := Aliases("Reglamento (UE) n.º 910/2014 del Parlamento Europeo y del Consejo, de 23 de julio de 2014", []string{"eIDAS"})
	if strings.Join(got, "|") != "Reglamento (UE) n.º 910/2014|910/2014|Reglamento (UE) 910/2014|eIDAS" {
		t.Errorf("%q", got)
	}
	got = Aliases("Reglamento (UE) 2016/679 del Parlamento Europeo y del Consejo", nil)
	if strings.Join(got, "|") != "Reglamento (UE) 2016/679|2016/679" {
		t.Errorf("%q", got)
	}
}
