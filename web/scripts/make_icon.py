# Builds web/public/icon.svg: the answer-sheet mark (A B C D, B marked) with
# the letters as outlines, so the icon does not depend on installed fonts.
# Run from web/:  pip install fonttools brotli && python3 scripts/make_icon.py
# then render the PNG sizes with scripts/render_icons.mjs.
from fontTools.ttLib import TTFont
from fontTools.varLib.instancer import instantiateVariableFont
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.boundsPen import BoundsPen

font = instantiateVariableFont(TTFont("node_modules/@fontsource-variable/figtree/files/figtree-latin-wght-normal.woff2"), {"wght": 700})
gs = font.getGlyphSet()
cmap = font.getBestCmap()
upm = font["head"].unitsPerEm
SIZE = 76  # letter size in icon units

def glyph(ch, cx, cy, fill):
    name = cmap[ord(ch)]
    bp = BoundsPen(gs); gs[name].draw(bp)
    x0, y0, x1, y1 = bp.bounds
    pen = SVGPathPen(gs); gs[name].draw(pen)
    s = SIZE / upm
    # Centre the glyph's outline on (cx, cy); font y grows upwards.
    tx = cx - (x0 + x1) / 2 * s
    ty = cy + (y0 + y1) / 2 * s
    return f'<path transform="translate({tx:.1f} {ty:.1f}) scale({s:.4f} {-s:.4f})" d="{pen.getCommands()}" fill="{fill}"/>'

ink, amber, dark = "#ece6da", "#f0b03f", "#1b1407"
parts = []
for i, ch in enumerate("ABCD"):
    cx, cy = 176 + (i % 2) * 160, 176 + (i // 2) * 160
    on = ch == "B"
    parts.append(f'<circle cx="{cx}" cy="{cy}" r="62" fill="{amber if on else "none"}" stroke="{amber if on else ink}" stroke-width="14"/>')
    parts.append(glyph(ch, cx, cy, dark if on else ink))

svg = ('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">'
       '<defs><radialGradient id="g"><stop offset="0" stop-color="#f0b03f" stop-opacity=".3"/>'
       '<stop offset="1" stop-color="#f0b03f" stop-opacity="0"/></radialGradient></defs>'
       '<rect width="512" height="512" rx="112" fill="#111318"/>'
       '<circle cx="256" cy="256" r="236" fill="url(#g)"/>' + "".join(parts) + '</svg>\n')
open("public/icon.svg", "w").write(svg)
print(len(svg), "bytes")
