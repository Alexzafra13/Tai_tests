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
3. Revisar el JSON (enunciados, opciones y letras) y `make test`: un test
   carga todo el banco y exige que cada pregunta pase la validación.
4. Subirlo. Al arrancar, cada instalación añade como borrador lo que aún no
   tiene.

Hecho: OEP 2019, 2022 y 2024 (ingreso libre). Pendiente: la OEP 2025, que
solo tiene plantilla provisional, y la OEP 2018, cuyo cuestionario está
escaneado (sin texto: habría que pasar OCR y revisarlo a mano).

### 4. Revisar y publicar

En la app, con el administrador: **Revisión → Aceptar en bloque** para cada
examen. Las preguntas necesitan un tema para publicarse; la IA puede
proponerlo según el temario, pero se confirma en la revisión.

### 5. Banco de preguntas incluido en la app

Hecho: `data/bank/` va en el binario y se carga al arrancar (ver el paso 3).
Pendiente: exportar al banco los temas asignados en la revisión, para que
las instalaciones nuevas reciban las preguntas ya clasificadas.

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
