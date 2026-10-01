# TAI · Estudio

Aplicación web para preparar la oposición de **Técnico Auxiliar de
Informática de la Administración del Estado (TAI, C1)**, con administradores y
usuarios. Un solo binario y un contenedor.

Principio básico: **nada inventado**. Toda pregunta guarda su fuente
verificable (examen oficial del INAP, ley del BOE o documentación técnica). La
IA solo transforma textos aportados y todo lo que genera pasa por revisión.

## Instalación rápida

Solo hace falta Docker (con Compose 2.24 o posterior). En el servidor:

```sh
curl -O https://raw.githubusercontent.com/alexzafra13/Tai_tests/main/docker-compose.yml
docker compose up -d
```

Abre `http://<tu-servidor>:8080`: la primera vez aparece la pantalla
**Bienvenido** para crear la cuenta de administrador. No hay que configurar
nada más; el resto (usuarios, nota, temario…) se gestiona desde la app.

- **Actualizar:** `docker compose pull && docker compose up -d`.
- **Datos:** todo vive en el volumen `tai-data` (`/data/tai.db`). Las
  migraciones se aplican solas al arrancar.
- **Imagen:** GitHub la compila automáticamente para amd64 y arm64 (sirve en un
  PC o en una Raspberry Pi) y la publica en `ghcr.io/alexzafra13/tai_tests`.
  Si el repositorio es privado, la imagen también: hazla pública en GitHub
  (paquete → *Package settings* → *Change visibility*) o inicia sesión con
  `docker login ghcr.io`.
- **Compilar desde el código** en vez de descargar la imagen:
  `docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build`.

### Configuración opcional

Nada es obligatorio. Para cambiar algo, crea un `.env` junto al
`docker-compose.yml` (ver [`.env.example`](.env.example)):

| Variable | Por defecto | Para qué |
|---|---|---|
| `TAI_PORT` | `8080` | Puerto en el servidor |
| `TAI_COOKIE_SECURE` | `false` | `true` si se sirve por HTTPS |
| `TAI_SESSION_TTL` | `720h` | Duración de la sesión |
| `TAI_ADMIN_USER` / `TAI_ADMIN_PASSWORD` | — | Crear el administrador sin la pantalla de bienvenida (instalaciones sin navegador). Se ignoran si ya existe |
| `TAI_ADDR` / `TAI_DB_PATH` | `:8080` / `tai.db` | Solo al ejecutar el binario fuera de Docker |

### Acceso desde el móvil fuera de casa

No expongas el puerto directamente a internet. Opciones recomendadas:

- **Tailscale** (o otra VPN) en el servidor y en el móvil. Con
  `tailscale serve` obtienes además HTTPS.
- Un **proxy inverso con HTTPS** (Caddy, Nginx Proxy Manager…).

Si la app se sirve por HTTPS, pon `TAI_COOKIE_SECURE=true`.

### Copias de seguridad

Todo el estado está en `tai.db`. Hasta que exista el comando `tai backup`,
para el contenedor y copia el fichero del volumen:

```sh
docker compose stop
docker run --rm -v tai_tai-data:/data -v "$PWD":/backup alpine cp /data/tai.db /backup/
docker compose start
```

## Stack

- **Backend:** Go (`net/http`), SQLite con `modernc.org/sqlite` (sin CGO),
  migraciones SQL embebidas, FTS5.
- **Frontend:** React + TypeScript + Vite, embebido en el binario con `embed`.
- **Despliegue:** Dockerfile multi-stage (Node → Go → Alpine con
  `poppler-utils`) y Docker Compose con un volumen para la base de datos.

## Estructura

```
cmd/tai/               binario y subcomandos (serve, migrate, version…)
internal/config/       variables de entorno
internal/db/           apertura de SQLite y migrador
internal/db/migrations/  migraciones SQL versionadas (NNNN_nombre.sql)
internal/users/        cuentas y roles (admin / usuario), contraseñas con bcrypt
internal/auth/         login, sesiones y usuario actual de cada petición
internal/validate/     errores de validación por campo, comunes a todos los paquetes
internal/content/      temario, fuentes y preguntas con sus reglas de validación
internal/quiz/         sesiones de test: creación, respuestas, historial, nota y barajado
internal/settings/     ajustes del usuario (clave → JSON)
internal/textmatch/    comprobación de citas literales (normaliza espacios y tipografía)
internal/server/       API HTTP y servidor de la SPA (routes.go: rutas y permisos)
web/                   frontend (Vite); web/dist se embebe en el binario
data/                  syllabus.example.json (formato del temario)
```

## Usuarios y permisos

No hay registro abierto: el administrador crea las cuentas en **Ajustes →
Usuarios**.

| | Administrador | Usuario |
|---|---|---|
| Tests, resultados, temario, marcar dudas | ✅ | ✅ |
| Cambiar su propia contraseña | ✅ | ✅ |
| Preguntas, fuentes, revisión | ✅ | ❌ |
| Reglas de la nota, cuentas de usuario | ✅ | ❌ |

- Los permisos se comprueban **en el servidor** ruta a ruta
  (`internal/server/routes.go`); la interfaz solo oculta lo que no se puede
  usar.
- Cada test y cada respuesta pertenece a su usuario: nadie ve los tests de
  otro, tampoco el administrador.
- Las **dudas** son avisos de cada usuario con su nota. El administrador los ve
  todos en Revisión, con quién los marcó; aceptar o descartar la pregunta los
  cierra (y deshacer los reabre).
- Los intentos fallidos de login se limitan por cuenta (10 cada 15 min).
  Desactivar una cuenta le quita el acceso al momento; cambiar una contraseña
  cierra las demás sesiones.
- Siempre queda al menos un administrador activo.

**Primer arranque:** la app muestra la pantalla de bienvenida para crear el
administrador (o lo crea desde `TAI_ADMIN_USER` / `TAI_ADMIN_PASSWORD` si se
dan). La pantalla desaparece en cuanto existe. Si se pierde la contraseña del
administrador:

```sh
docker compose exec -T tai tai user passwd -username admin <<< 'nueva-contraseña'
docker compose exec tai tai user list
```

## Reglas de contenido

Se aplican en el backend al guardar, vengan de la interfaz, de la línea de
comandos o del futuro pipeline de IA:

- Toda pregunta tiene **fuente** (`source_id`) y **referencia** (año y nº de
  pregunta, artículo…).
- El origen fija el tipo de fuente: `official` → examen INAP, `law` → ley,
  `technical` → documentación técnica.
- Las preguntas `law` y `technical` necesitan una **cita literal** de al menos
  20 caracteres que **aparezca en el texto completo de la fuente**. Solo se
  normalizan espacios, saltos de línea, comillas y guiones tipográficos;
  nunca palabras ni mayúsculas.
- Para publicar una pregunta hace falta al menos un tema.
- No se puede modificar el texto de una fuente si alguna cita de sus
  preguntas deja de aparecer en él, ni borrar una fuente con preguntas.

## Tests

- **Práctica:** corrección inmediata con explicación, fuente y cita tras cada
  pregunta.
- **Examen:** sin corrección hasta entregar. Las respuestas se pueden cambiar;
  pulsar otra vez la opción marcada la deja en blanco. Tiempo límite opcional
  (por defecto 1,2 min por pregunta, como el examen real).
- Solo entran preguntas **publicadas y no anuladas**.
- Cada respuesta se guarda al momento en el servidor: si cierras la app o se
  bloquea el móvil, el test se retoma donde estaba. El reloj de un examen
  sigue corriendo, como en el examen real; si vuelves pasado el límite, el
  examen se corrige a la hora límite.
- **Nota** en la escala de la convocatoria:
  `(aciertos − fallos × penalización) / preguntas × puntuación máxima`, con
  aprobado. Puntuación máxima, aprobado y penalización por defecto se
  configuran en **Ajustes** (por defecto, provisionalmente: sobre 100,
  aprobado en 50 y −1/3 por fallo). Cada test guarda sus netas, así que al
  cambiar la escala se recalculan también los resultados anteriores. El
  resultado muestra la nota sin penalización, para ver cuánto cuestan los
  fallos.
- **Opciones barajadas** en cada test (el orden se guarda, así que al
  retomar un test las ves igual). No se mueve nada que pueda romper la
  pregunta: si alguna opción nombra letras («A y B son correctas», «solo la
  A») la pregunta mantiene su orden, y las opciones del tipo «todas/ninguna
  de las anteriores» se quedan en su sitio. Cada pregunta tiene además la
  casilla «No barajar las opciones».
- Desde el resultado puedes **repetir las falladas** en un test de práctica.
- Las preguntas con respuestas no se pueden borrar: se marcan como
  **descartadas** para no perder el historial. Si editas el enunciado, las
  opciones o la respuesta correcta, sube su número de revisión y cada intento
  guarda la revisión que viste.

## Cola de revisión

La pestaña **Revisión** reúne lo que aún no sale en los tests: borradores
(importados o generados) y preguntas marcadas como dudosas durante un test,
estas primero. Cada pregunta se muestra con su respuesta, la fuente y la cita
**resaltada dentro del texto de la ley**. Acciones: **Aceptar** (publica y
quita la marca de dudosa; si no tiene tema se asigna ahí mismo),
**Descartar**, **Saltar** y editar en el formulario completo. Aceptar y
descartar se pueden **deshacer** durante unos segundos.

**Aceptar en bloque** sirve para exámenes importados: lista los borradores
de una fuente, preselecciona los que se pueden publicar tal cual y valida
cada uno por separado (los que fallan quedan en la cola con su motivo).

## Cargar el temario y las fuentes

El temario se define en `data/syllabus.json` (formato en
[`data/syllabus.example.json`](data/syllabus.example.json)). Cada bloque y
tema tiene un `code` estable: al recargar el fichero se actualizan títulos y
orden, y los temas que desaparezcan se desactivan sin perder sus preguntas.

```sh
# En local
go run ./cmd/tai load-syllabus -file data/syllabus.json

# Con Docker (el fichero se pasa por la entrada estándar)
docker compose exec -T tai tai load-syllabus -file - < data/syllabus.json
```

Las fuentes se pueden crear desde la interfaz (Fuentes → Nueva) o, para
textos largos, desde la línea de comandos:

```sh
docker compose exec -T tai tai add-source -kind law \
  -title "Ley 39/2015, del Procedimiento Administrativo Común" \
  -ref BOE-A-2015-10565 -version 2024-01-01 -text - < ley39.txt
```

## Desarrollo local

Requisitos: Go 1.26+ y Node 22+.

```sh
cd web && npm ci && cd ..

# Terminal 1: API en :8080 (crea el admin "admin" / "devpassword")
make dev-api

# Terminal 2: frontend con recarga en caliente en :5173 (redirige /api a :8080)
make dev-web
```

Abre http://localhost:5173.

Otros comandos:

```sh
make test    # go vet, tests de Go y typecheck del frontend (lo mismo que la CI)
make build   # compila el frontend y genera bin/tai con todo embebido
```

## Hoja de ruta

1. ✅ Esqueleto: Go + SQLite + migraciones, SPA embebida, login, Docker.
2. ✅ Temario y modelo de preguntas; alta y edición manual.
3. ✅ Tests (práctica y examen) y registro de intentos.
4. ✅ Cola de revisión.
5. Importador de exámenes del INAP y modo simulacro.
6. Generación del bloque legal con validación.
7. FSRS, falladas, estadísticas y búsqueda.
8. Generación del bloque técnico, PWA y pulido.
