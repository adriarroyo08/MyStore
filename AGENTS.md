# AGENTS.md

Instrucciones comunes para cualquier agente (Claude Code, Codex, Gemini/Antigravity). Claude las lee desde `CLAUDE.md`.

## Proyecto

Tienda de APKs autoalojada: servidor Go (`server/`, un binario en Docker, sin base de datos) y apps Android y Wear OS en Kotlin + Compose (`android/mobile`, `android/wear`, lógica común en `android/core`). Detalle en `README.md`.

## Verificar

```bash
cd server && go vet ./... && go test ./...                      # siempre que toques server/
cd android && ./gradlew :core:testDebugUnitTest assembleDebug   # si tocas android/ (Windows: .\gradlew.bat)
docker build -t mystore ./server                                 # si tocas el Dockerfile o las dependencias
```

El CI (`.github/workflows/ci.yml`) ejecuta lo mismo y una prueba de humo del contenedor.

## Reglas

- Una issue = una rama = un PR contra `master`. Rama `feat/<corto>` o `fix/<corto>`; el PR lleva `Closes #<n>`.
- Haz solo lo que pide la issue. Si falta información o la duda cambia el diseño, pregunta en vez de inventar.
- Toda lógica nueva lleva test (`*_test.go` en el servidor, `android/core` en las apps).
- El repo es público: nada de tokens, dominios ni rutas de servidores personales en el código ni en los PR.
- No toques `action.yml` (la GitHub Action que usan otros repos) sin que la issue lo pida: un cambio ahí rompe sus publicaciones.

## Etiquetas

| Etiqueta | Significado |
| --- | --- |
| `agent:claude` | Arquitectura y diseño de módulos, interfaz y sistema de diseño, seguridad (auth, API keys, permisos), bugs difíciles o de concurrencia, refactors que cruzan el proyecto. |
| `agent:codex` | Implementar con la spec clara: lógica, endpoints, pantallas ya diseñadas, tests, migraciones de dependencias, arreglos de CI y build, scripts. También revisa los PR de Claude. |
| `agent:gemini` | Investigar con fuentes web (estudios, noticias, competidores), contenido y datos con citas, documentación, resumir mucho código, revisar capturas e imágenes, generar assets. |
| `necesita:pc` | Para verificarla hay que ver la interfaz, sacar capturas, usar emulador o dispositivo, o compilar la app Android: solo en el PC, no en el servidor. |
| `needs-spec` | Le faltan criterios o cómo verificar: no se reparte. |
| `epic` | Objetivo grande: se trocea en issues hijas (`Parte de #n`). |
| `en-curso` | Un agente la está haciendo. |
| `size:s` / `size:m` / `size:l` | Tamaño estimado. |
