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
2. Descargarla de la API de datos abiertos del BOE:

   ```sh
   go run ./cmd/tai fetch-law -ref BOE-A-2015-10565 -alias "LPACAP"
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

Hecho: las 22 normas españolas del bloque I, la LO 10/2022 (tema 5) y las
12 Normas Técnicas de Interoperabilidad vigentes (tema 8; las dos
derogadas se dejan fuera). Comprobado con la web del BOE: el texto de cada
artículo coincide. Pendiente: el RGPD y el reglamento eIDAS, que solo están
completos y vigentes en EUR-Lex (hay que permitir `eur-lex.europa.eu` y
`publications.europa.eu` en la red del entorno).

Es la base para generar después preguntas de ley con cita literal (fase 6).

## Comprobar

- `make test` en verde.
- Arrancar la app con una base de prueba, importar un examen y hacer un
  test con él: enunciados, opciones, respuesta correcta y fuente visibles.
