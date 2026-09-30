# TAI · Estudio

Aplicación web personal para preparar la oposición de **Técnico Auxiliar de
Informática de la Administración del Estado (TAI, C1)**. Un solo usuario, un
solo binario y un contenedor.

Principio básico: **nada inventado**. Toda pregunta guarda su fuente
verificable (examen oficial del INAP, ley del BOE o documentación técnica). La
IA solo transforma textos aportados y todo lo que genera pasa por revisión.

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
internal/auth/         login de un solo usuario y sesiones
internal/server/       API HTTP y servidor de la SPA
web/                   frontend (Vite); web/dist se embebe en el binario
```

## Desarrollo local

Requisitos: Go 1.26+ y Node 22+.

```sh
cd web && npm ci && cd ..

# Terminal 1: API en :8080 (contraseña por defecto: devpassword)
make dev-api

# Terminal 2: frontend con recarga en caliente en :5173 (redirige /api a :8080)
make dev-web
```

Abre http://localhost:5173.

Otros comandos:

```sh
make test    # go vet, tests de Go y typecheck del frontend
make build   # compila el frontend y genera bin/tai con todo embebido
```

## Despliegue con Docker Compose

```sh
cp .env.example .env
# Edita .env: como mínimo cambia TAI_PASSWORD
docker compose up -d --build
```

La app queda en `http://<tu-servidor>:8080`. La base de datos vive en el
volumen `tai-data` (`/data/tai.db` dentro del contenedor). Las migraciones se
aplican solas al arrancar.

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

## Variables de entorno

Ver [`.env.example`](.env.example).

| Variable | Por defecto | Descripción |
|---|---|---|
| `TAI_PASSWORD` | — (obligatoria) | Contraseña de acceso, mínimo 8 caracteres |
| `TAI_ADDR` | `:8080` | Dirección de escucha |
| `TAI_DB_PATH` | `tai.db` (`/data/tai.db` en Docker) | Ruta de la base de datos |
| `TAI_COOKIE_SECURE` | `false` | `true` si se sirve por HTTPS |
| `TAI_SESSION_TTL` | `720h` | Duración de la sesión |
| `ANTHROPIC_API_KEY` | — | Solo para los comandos de generación (fase 5+) |
| `TAI_LLM_MODEL` | `claude-sonnet-5` | Modelo para la generación (fase 5+) |

## Hoja de ruta

1. ✅ Esqueleto: Go + SQLite + migraciones, SPA embebida, login, Docker.
2. Temario y modelo de preguntas; alta y edición manual.
3. Tests (práctica y examen) y registro de intentos.
4. Cola de revisión.
5. Importador de exámenes del INAP y modo simulacro.
6. Generación del bloque legal con validación.
7. FSRS, falladas, estadísticas y búsqueda.
8. Generación del bloque técnico, PWA y pulido.
