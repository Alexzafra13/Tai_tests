# Cargar el contenido real

Guía para la sesión que mete en TAI Go el temario, los exámenes oficiales y
las leyes. Está pensada para que Claude Code la siga paso a paso; las reglas
generales del proyecto están en `CLAUDE.md`.

## Dónde trabajar

**Recomendado: Claude Code en la web** (claude.ai/code), en este mismo
repositorio. No hay que instalar nada: el entorno ya trae Go, Node y git, y
la sesión instala `pdftotext` sola (`apt-get install -y poppler-utils`).

Lo único que hace falta, una vez: permitir en la red del entorno los
dominios del BOE y del INAP. En la sesión, menú del entorno en la barra de
título → **Edit** → **Network access**, y añadir a los dominios permitidos:

```
boe.es
www.boe.es
sede.inap.gob.es
www.inap.es
```

Alternativa: Claude Code en tu PC, en una copia del repositorio. Necesita
`git`, Go 1.26+, Node 22+ y `pdftotext` (macOS: `brew install poppler`;
Ubuntu/WSL: `sudo apt install poppler-utils`).

## Reglas de esta tarea

- **Nada inventado.** Solo se carga lo que está en un documento oficial,
  con su referencia: convocatoria y año, número de pregunta, artículo.
- **Plantilla definitiva**, nunca la provisional: es la que recoge las
  preguntas anuladas y las respuestas corregidas tras las reclamaciones. Las
  anuladas se importan marcadas como anuladas (no salen en los tests).
- Todo lo importado entra como **borrador** y se publica desde Revisión.
- Si un PDF no se puede leer bien (escaneado, columnas raras), se para y se
  dice; no se rellenan huecos a ojo.
- Los PDF descargados van a `data/raw/` (fuera de git).

## Pasos

### 1. Temario

1. Buscar en el BOE la convocatoria vigente de Técnicos Auxiliares de
   Informática de la Administración del Estado. El temario está en su anexo.
2. Pasarlo **literalmente** a `data/syllabus.json` con el formato de
   `data/syllabus.example.json`. Los `code` (B1-T01…) son estables: al
   recargar se actualizan títulos y los temas que desaparecen se desactivan.
3. Va dentro del binario: una instalación sin temario lo carga al
   arrancar. En una ya en marcha: `go run ./cmd/tai load-syllabus -file data/syllabus.json`.

Hecho: convocatoria 2025 (BOE-A-2025-26262, anexo V), 4 bloques y 33 temas.

### 2. Nota de la convocatoria

Leer en las bases el sistema de calificación (puntuación máxima, nota para
aprobar y penalización por error) y ponerlo como valor por defecto en
`internal/quiz/scoring.go`, citando el apartado de las bases en el comentario.
En una instalación ya en marcha se cambia en Ajustes → Nota.

### 3. Exámenes del INAP

Están en la sede electrónica del INAP: Procedimientos y servicios →
Selección → Cuerpo de Técnicos Auxiliares de Informática, una página por
proceso selectivo, con el cuestionario y las plantillas de respuestas.

1. Descargar a `data/raw/inap/<año>/` el cuestionario, la plantilla
   **definitiva** y la provisional (solo sirve para dar respuesta a las
   anuladas). Si el proceso aún no tiene definitiva, esperar.
2. Convertirlo en un fichero del banco:

   ```sh
   go run ./cmd/tai import-exam -title "TAI ingreso libre · OEP 2024 · ejercicio único (modelo A)" \
     -ref INAP-TAI-L-OEP2024 -label "OEP 2024" -url <página del proceso> \
     -questions data/raw/inap/2024/cuestionario-A.pdf \
     -answers data/raw/inap/2024/plantilla-definitiva-A.pdf \
     -provisional data/raw/inap/2024/plantilla-provisional-A.pdf \
     -out data/bank/inap-tai-l-oep2024.json
   ```

   Lee el cuestionario con `pdftotext -layout` (primera parte, supuestos I
   y II y sus preguntas de reserva), lo cruza con las plantillas y escribe
   `data/bank/<ref>.json`. Al final informa de lo que no ha podido leer, con
   el número de pregunta. La `-ref` no se cambia nunca: identifica las
   preguntas del examen en todas las instalaciones.
3. Comprobar cada pregunta contra el PDF (enunciado, opciones y letra de
   cada opción) y las respuestas contra la plantilla definitiva. A las
   comprobadas, ponerles `topics` (códigos de `data/syllabus.json`) y
   `"status": "published"`; las que dependen de una figura que no se
   reproduce se dejan sin estado, para Revisión. Al reimportar, se conserva
   en las preguntas que no cambian.
   Con las leyes del tema ya en `data/laws` (paso 6), cada pregunta lleva
   en `articles` lo que la responde, comprobado contra el texto vigente:
   `{"law": "BOE-A-1978-31229", "section": "a54"}` (sin `section`, la norma
   entera) o, si no la responde ninguna ley, la página oficial que sí, con
   la frase literal que respalda la respuesta: `{"title": "eVisor · Centro
   de Transferencia de Tecnología", "url": "https://…", "quote": "…"}`.
   Vale la documentación técnica oficial (Microsoft Learn, páginas de
   manual de Linux, POSIX, documentación de PostgreSQL u Oracle, RFC, W3C,
   USB-IF…), nunca apuntes, academias ni Wikipedia. Sustituye a los
   artículos que la app deduce del enunciado y llega también a las
   instalaciones que ya tenían la pregunta. Al arrancar, las preguntas que
   nadie ha editado en la instalación también toman del banco sus temas,
   explicación y estado.
4. `make test`: un test carga todo el banco con el temario incluido y exige
   que cada pregunta pase la validación y que las comprobadas se publiquen.
5. Subirlo. Al arrancar, cada instalación añade lo que aún no tiene.

Hecho: ingreso libre OEP 2019, 2022 y 2024, y promoción interna OEP 2019,
2022, 2024 y 2025: 745 preguntas, 741 publicadas y 4 en Revisión. Las
respuestas se comprueban con una segunda lectura independiente de la
plantilla (por coordenadas: `pdftotext -bbox`). Pendiente: ingreso libre
OEP 2025, que solo tiene plantilla provisional, y la OEP 2018 (ingreso
libre y promoción interna), cuyos cuestionarios están escaneados (sin
texto: habría que pasar OCR y revisarlo a mano).

### 4. Revisar y publicar

En la app, con el administrador: **Revisión → Aceptar en bloque** para cada
examen. Las preguntas necesitan un tema para publicarse; la IA puede
proponerlo según el temario, pero se confirma en la revisión.

### 5. Banco de preguntas incluido en la app

Hecho: `data/bank/` va en el binario y se carga al arrancar (ver el paso 3),
con los temas asignados.

### 6. Leyes del temario

Las leyes de cada tema van en `data/laws/`, dentro del binario: un JSON por
norma con su **texto consolidado del BOE** tal cual, por títulos, capítulos
y artículos, y `topics.json` con las normas de cada tema (`parts` limita una
norma a algunos de sus títulos). Al arrancar se cargan como fuentes de tipo
ley, con su texto completo para validar citas, y se leen en
**Temario → Leyes**.

1. Elegir las normas de un tema por lo que nombra el temario y lo que citan
   las preguntas oficiales del banco.
2. Descargarla de la API de datos abiertos del BOE o, si es un reglamento
   europeo, del texto consolidado de EUR-Lex (por su número CELEX):

   ```sh
   go run ./cmd/tai fetch-law -ref BOE-A-2015-10565 -alias "LPACAP"
   go run ./cmd/tai fetch-law -ref 32016R0679 -alias "RGPD"
   ```

   Guarda la redacción vigente de cada artículo y, si ya hay publicada una
   reforma que entra en vigor más tarde, también esa con su fecha; la app
   muestra la que toque cada día. Deja fuera lo caducado (capítulos
   suprimidos) y las firmas. `-alias` añade nombres con que las preguntas
   citan la norma, además de su número.
3. Añadirla a `topics.json` y `make test`.
4. Para actualizar, `go run ./cmd/tai refresh-laws` descarga de nuevo
   todas (conserva los `-alias`). Lo hace cada mes la tarea
   `.github/workflows/laws.yml`, que abre una PR con los cambios (en
   GitHub hay que permitir que Actions cree PR: Settings → Actions →
   General → *Allow GitHub Actions to create and approve pull requests*).
   Si el texto nuevo deja sin respaldo la cita de alguna pregunta, la
   instalación conserva el anterior y lo avisa al arrancar.
5. Una norma derogada no se descarga: `fetch-law` la rechaza y
   `refresh-laws` actualiza las demás y termina en error con la norma y la
   fecha de derogación, así que la tarea mensual sale en rojo. Hay que
   descargar la norma que la sustituye, cambiarla en `topics.json`, llevar
   a ella los `articles` del banco (con una `explanation` si la respuesta
   del examen ya no es la vigente) y borrar el fichero de la derogada.

Hecho, bloque I completo: 29 normas españolas (con los reglamentos de
ingreso y provisión, situaciones administrativas y régimen disciplinario,
la Ley 56/2007, el RD 209/2003 y la resolución de Cl@ve), las 12 Normas
Técnicas de Interoperabilidad vigentes (las dos derogadas se dejan fuera)
y, de EUR-Lex, el RGPD y el reglamento eIDAS. Comprobado con la web del BOE
y EUR-Lex: el texto de cada artículo coincide. Las 109 preguntas oficiales
del bloque están revisadas una a una contra el texto vigente: todas
mantienen su respuesta y 108 llevan lo que las responde (88 un artículo,
20 una página oficial: fichas del CTT, AEPD, Agenda 2030…); falta la
nº 16 de 2019 (algoritmos del DNIe 3.0), sin fuente oficial publicada.

Bloque II (sin leyes): sus 80 preguntas oficiales están revisadas contra
la documentación técnica oficial; 77 llevan la página que las responde con
su frase literal, comprobada descargando de nuevo cada página. Las tres
desfasadas (USB4 y la versión de macOS) explican en qué ha cambiado. Sin
fuente quedan las dos anuladas y la nº 34 de 2022 (bases de datos
orientadas a objetos: el estándar ODMG no está publicado en línea).

Es la base para generar después preguntas de ley con cita literal (fase 6).

### 7. Apuntes

Un JSON por tema en `data/notes/` (`B1-T01.json`): secciones con puntos y,
como mucho, un nivel de subpuntos. Cada punto es una frase de estudio
(`**negrita**` para lo clave) y lleva en `refs` lo que la respalda, con la
cita literal: `{"law": "BOE-A-1978-31229", "section": "a62", "quote": "…"}`
o `{"title": "…", "url": "https://…", "quote": "…"}`. Nada sin cita.

1. Escribirlo leyendo el texto vigente de las normas del tema y las
   preguntas oficiales del banco (dicen qué se pregunta).
2. Contrastar cada punto con su artículo: que diga exactamente lo mismo,
   sin perder matices («en su caso», «previa autorización», «salvo»).
3. `make test`: un test carga todos los apuntes y exige que cada cita de
   una ley esté en la sección que nombra. Al arrancar, un apunte con una
   cita que ya no está (porque la ley cambió) no se carga y se avisa.

Salen publicados (ver CLAUDE.md). Los avisos de los usuarios llegan a
Revisión.

Hecho: tema 1 del bloque I (Constitución: título preliminar, derechos y
deberes, garantías, suspensión y Corona).

## Comprobar

- `make test` en verde.
- Arrancar la app con una base de prueba, importar un examen y hacer un
  test con él: enunciados, opciones, respuesta correcta y fuente visibles.
