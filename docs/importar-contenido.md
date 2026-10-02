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
3. Cargarlo: `go run ./cmd/tai load-syllabus -file data/syllabus.json`.

### 2. Nota de la convocatoria

Leer en las bases el sistema de calificación (puntuación máxima, nota para
aprobar y penalización por error) y ponerlo como valor por defecto en
`internal/quiz/scoring.go`, citando el apartado de las bases en el comentario.
En una instalación ya en marcha se cambia en Ajustes → Nota.

### 3. Importador de exámenes del INAP

Es código nuevo (fase 5 de la hoja de ruta). Comando propuesto:

```sh
tai import-exam -title "TAI ingreso libre · OEP 2024" -ref "OEP-2024-LI" \
  -questions data/raw/inap/2024/cuestionario.pdf \
  -answers data/raw/inap/2024/plantilla-definitiva.pdf
```

Qué debe hacer:

- Crear (o reutilizar) la fuente de tipo `inap_exam` con título, referencia
  y URL de descarga.
- Extraer las preguntas con `pdftotext -layout`: enunciado y cuatro
  opciones. Cruzarlas con la plantilla para la respuesta correcta y las
  anuladas.
- Guardar cada una con `origin = official`, `author = import`,
  `status = draft` y `source_ref` como `2024 · nº 37`, usando
  `content.Store.CreateQuestion`, para que pasen las mismas validaciones
  que todo lo demás.
- Ser **idempotente**: la clave es fuente + número de pregunta. Al
  reimportar no duplica, y nunca resucita una pregunta descartada.
- Terminar con un informe: importadas, ya existentes y con problemas (por
  qué y qué número).
- Tests con un fragmento de texto de muestra del PDF real.

Los exámenes están en la sede electrónica del INAP, en el proceso selectivo
del cuerpo TAI de cada año: el cuestionario y la plantilla de respuestas.
Empezar por los más recientes. Si la web es difícil de recorrer, descargar
los PDF a mano a `data/raw/inap/<año>/` y seguir desde ahí.

En Docker, el contenedor ya trae `pdftotext`; los PDF se pasan con un
volumen o `docker compose cp`.

### 4. Revisar y publicar

En la app, con el administrador: **Revisión → Aceptar en bloque** para cada
examen. Las preguntas necesitan un tema para publicarse; la IA puede
proponerlo según el temario, pero se confirma en la revisión.

### 5. Banco de preguntas incluido en la app

Para que cualquier instalación nueva empiece con contenido:

- Exportar las preguntas publicadas y sus fuentes a `data/bank/` (JSON,
  un fichero por fuente, con clave estable por pregunta).
- Incluir `data/bank/` en el binario con `embed` y cargarlo al arrancar,
  de forma idempotente: añade lo que falta y respeta lo descartado o
  editado en esa instalación.

### 6. Leyes del temario

Añadir como fuentes las leyes que cita el temario, con su **texto
consolidado del BOE** (referencia BOE-A-…, fecha de la versión):

```sh
go run ./cmd/tai add-source -kind law -title "Ley 39/2015, del Procedimiento Administrativo Común" \
  -ref BOE-A-2015-10565 -version 2024-01-01 -text - < data/raw/boe/ley39.txt
```

Es la base para generar después preguntas de ley con cita literal (fase 6).

## Comprobar

- `make test` en verde.
- Arrancar la app con una base de prueba, importar un examen y hacer un
  test con él: enunciados, opciones, respuesta correcta y fuente visibles.
