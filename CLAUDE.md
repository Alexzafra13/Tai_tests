# TAI Go — guía para trabajar en el repo

App web para preparar la oposición TAI (C1). Go + SQLite en el servidor,
React + TypeScript en el cliente, todo en un binario. El README explica qué
hace la app; aquí van las reglas para cambiarla.

## Reglas que no se rompen

- **Nada inventado.** Toda pregunta guarda su fuente verificable (examen
  oficial del INAP, ley del BOE o documentación técnica) y su referencia.
  Las preguntas de ley y técnicas llevan una cita literal que debe aparecer
  en el texto completo de la fuente. Se valida en el servidor
  (`internal/content/questions.go`, `validate`), venga de donde venga: no
  añadas caminos que se lo salten.
- La IA solo transforma textos aportados, y lo que genera entra como
  borrador (`status = draft`) para pasar por Revisión.
- Permisos en el servidor, ruta a ruta, en `internal/server/routes.go`.

## Idiomas

- Interfaz y mensajes al usuario: español.
- Código, identificadores, comentarios, tablas, columnas y commits: inglés.

## Arquitectura

- Un paquete por área en `internal/` (content, quiz, srs, stats, users,
  auth, settings). Solo `server` los conoce a todos; las dependencias van en
  un sentido. La lógica va en los paquetes, no en los handlers.
- Base de datos: migraciones SQL versionadas en `internal/db/migrations`
  (`NNNN_nombre.sql`), nunca se editan para cambiar el esquema: se añade una
  nueva. `-- migrate:rebuild` en la primera línea para reconstruir tablas.
  Si una migración reconstruye `questions`, recrea los triggers de
  `questions_fts`.
- Fechas en la base: `db.Timestamp(t)` (UTC, milisegundos). Errores de
  restricción: `db.IsUnique` / `db.IsForeignKey`, nunca comparar texto.
- Cliente: `web/src/types` refleja el JSON de la API; estilos en capas CSS
  (`base` < `components` < `pages`), cada CSS empieza con
  `@layer base, components, pages;` y los de una pantalla van junto a ella.
- Colores solo con los tokens de `web/src/styles/tokens.css` (tema oscuro
  por defecto, claro y automático).
- App instalable: el service worker sale de la plantilla `web/sw.js`
  (lo completa `vite.config.ts`). Nunca debe cachear `/api`: los datos son
  de cada usuario y tienen que estar al día.

## Comentarios

Solo los necesarios: el porqué, una regla de negocio, una trampa o un
formato. Concretos y cortos. Nada que repita el código ni que cuente la
historia del cambio.

## Comprobar antes de subir

```sh
make test     # go vet, tests de Go y typecheck del frontend (igual que la CI)
make build    # binario con el frontend embebido en bin/tai
```

Para ver cambios de interfaz, arranca `bin/tai serve` con una base de datos
de prueba y revisa capturas en móvil, en tema oscuro y claro.
