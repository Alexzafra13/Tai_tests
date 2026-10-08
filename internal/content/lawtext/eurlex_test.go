package lawtext

import (
	"strings"
	"testing"

	"github.com/alexzafra13/tai_tests/internal/content"
)

// Shortened from the EUR-Lex consolidated text of the eIDAS regulation,
// with the GDPR's way of laying out definitions.
const sampleEURLex = `<!DOCTYPE html>
<html><head><meta charset="utf-8"/><title>TEXTO consolidado</title></head><body>
<div class="eli-container">
  <p class="hd-modifiers">►C2 Rectificado por: …</p>
  <div class="eli-main-title" id="tit_1">
    <p class="title-doc-first">REGLAMENTO (UE) N
      <span class="superscript">o</span> 910/2014 DEL PARLAMENTO EUROPEO Y DEL CONSEJO</p>
    <p class="title-doc-first">de&nbsp;23 de julio de 2014</p>
    <p class="title-doc-last"><a href="x" title="32014R0910">relativo a la identificación electrónica</a>
      <span><a href="y" title="32014R0910R(08): REPLACED"><span class="boldface">►C2</span></a></span> y por el que se deroga la Directiva 1999/93/CE<span class="boldface"> ◄ </span></p>
    <p class="title-doc-last">(Texto pertinente a efectos del EEE)</p>
  </div>
  <div class="eli-subdivision" id="enc_1">
    <p><br/><br/></p>
    <div id="cpt_I">
      <p class="title-division-1">CAPÍTULO I</p>
      <p class="title-division-2"><span class="boldface">DISPOSICIONES GENERALES</span></p>
      <p class="modref"><a href="z" title="32024R1183: REPLACED">▼M2</a></p>
      <div class="eli-subdivision" id="art_3">
        <p class="title-article-norm">Artículo 3</p>
        <div class="eli-title" id="art_3.tit_1"><p class="stitle-article-norm">Definiciones</p></div>
        <p class="norm">A efectos del presente Reglamento, se aplicarán las siguientes definiciones:</p>
        <div class="grid-container grid-list">
          <div class="list grid-list-column-1"><span>1) </span></div>
          <div class="grid-list-column-2"><p class="norm">«identificación electrónica», proceso;</p></div>
        </div>
        <table width="100%"><col width="5%"/><col width="95%"/><tr>
          <td valign="top"><p class="dlist-term">2)</p></td>
          <td valign="top"><p class="dlist-definition">«tratamiento» : cualquier operación;</p></td>
        </tr></table>
      </div>
    </div>
    <div id="cpt_II">
      <p class="title-division-1">CAPÍTULO II</p>
      <p class="title-division-2">IDENTIFICACIÓN ELECTRÓNICA</p>
      <div id="cpt_II.sct_1">
        <p class="title-division-1">SECCIÓN 1</p>
        <p class="title-division-2">Cartera europea de identidad digital</p>
        <div class="eli-subdivision" id="art_5a">
          <p class="title-article-norm">Artículo 5 bis</p>
          <div class="eli-title"><p class="stitle-article-norm">Carteras europeas de identidad digital</p></div>
          <div class="norm">
            <span class="no-parag">1.  </span>
            <div class="norm inline-element">Cada Estado miembro proporcionará una cartera (<a href="#E0001" id="src.E0001"><span class="superscript">1</span></a>).</div>
          </div>
          <div class="norm">
            <span class="no-parag">2.  </span>
            <div class="norm inline-element">Las carteras se proporcionarán:</div>
            <div class="grid-container grid-list">
              <div class="list grid-list-column-1"><span>a) </span></div>
              <div class="grid-list-column-2"><p class="norm">directamente por un Estado miembro;</p></div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
  <div id="anx_I">
    <hr class="separator-annex"/>
    <p class="title-annex-1">ANEXO I</p>
    <p class="title-gr-seq-level-1"><span class="boldface">REQUISITOS DE LOS CERTIFICADOS</span></p>
    <p class="norm">Los certificados contendrán:</p>
  </div>
</div>
<hr class="separator-short"/>
<p class="footnote">(<a href="#src.E0001" id="E0001"><span class="superscript">1</span></a>) Directiva (UE) 2015/1535.</p>
</body></html>`

func TestParseEURLexText(t *testing.T) {
	title, secs, err := ParseEURLexText([]byte(sampleEURLex))
	if err != nil {
		t.Fatal(err)
	}
	if want := "Reglamento (UE) n.º 910/2014 del Parlamento Europeo y del Consejo, de 23 de julio de 2014, relativo a la identificación electrónica y por el que se deroga la Directiva 1999/93/CE"; title != want {
		t.Errorf("title %q", title)
	}
	want := []content.LawSection{
		{ID: "cpt_I", Kind: content.SectionHeading, Level: 1, Title: "CAPÍTULO I. DISPOSICIONES GENERALES"},
		{ID: "art_3", Kind: content.SectionArticle, Title: "Artículo 3. Definiciones",
			Body: "A efectos del presente Reglamento, se aplicarán las siguientes definiciones:\n\n1) «identificación electrónica», proceso;\n\n2) «tratamiento» : cualquier operación;"},
		{ID: "cpt_II", Kind: content.SectionHeading, Level: 1, Title: "CAPÍTULO II. IDENTIFICACIÓN ELECTRÓNICA"},
		{ID: "cpt_II-sct_1", Kind: content.SectionHeading, Level: 2, Title: "SECCIÓN 1. Cartera europea de identidad digital"},
		{ID: "art_5a", Kind: content.SectionArticle, Title: "Artículo 5 bis. Carteras europeas de identidad digital",
			Body: "1. Cada Estado miembro proporcionará una cartera.\n\n2. Las carteras se proporcionarán:\n\na) directamente por un Estado miembro;"},
		{ID: "anx_I", Kind: content.SectionArticle, Title: "ANEXO I. REQUISITOS DE LOS CERTIFICADOS", Body: "Los certificados contendrán:"},
	}
	if len(secs) != len(want) {
		t.Fatalf("got %d sections: %+v", len(secs), secs)
	}
	for i := range want {
		if secs[i] != want[i] {
			t.Errorf("section %d\n got %+v\nwant %+v", i, secs[i], want[i])
		}
	}
}

func TestLatestEURLexVersion(t *testing.T) {
	page := `<a href="?uri=CELEX:02014R0910-20240520">x</a> <a href="?uri=CELEX:02014R0910-20241018">y</a>
		<a href="?uri=CELEX:02014R0910-20140917">z</a> <a href="?uri=CELEX:02016R0679-20991231">other</a>`
	v, err := LatestEURLexVersion([]byte(page), "32014R0910")
	if err != nil || v != "20241018" {
		t.Errorf("%q %v", v, err)
	}
	if !IsCELEX("32016R0679") || IsCELEX("BOE-A-2015-10565") {
		t.Error("IsCELEX")
	}
	if got := EURLexTextURL("32014R0910", "20241018"); !strings.HasSuffix(got, "CELEX:02014R0910-20241018") {
		t.Error(got)
	}
}
