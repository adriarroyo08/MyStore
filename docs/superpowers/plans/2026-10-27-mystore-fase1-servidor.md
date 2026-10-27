# MyStore Fase 1 (prueba en reloj + servidor) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Confirmar que el TicWatch puede instalar APKs desde una app y construir el servidor de MyStore (API + web de subida + Docker + GitHub Action).

**Architecture:** Un binario Go sin base de datos: cada app vive en `DATA/apps/<id>/` con `app.apk`, `icon.png` y `meta.json`; los metadatos se extraen de la APK al subirla. Rutas con `net/http` (patrones de Go 1.22+), web de administración embebida con `go:embed`. La Action es un `action.yml` compuesto que hace `curl`.

**Tech Stack:** Go (última estable vía winget), `github.com/shogo82148/androidbinary` (manifest, label, icono), `github.com/avast/apkverifier` (certificado de firma), Docker, GitHub Actions. Fixtures generadas con `aapt2` + `apksigner` del SDK Android local.

**Spec:** `docs/superpowers/specs/2026-10-27-mystore-design.md`

La Fase 2 (apps Android móvil y reloj) tendrá su propio plan, escrito tras el resultado de la Task 2.

## Global Constraints

- Una instancia por persona; un único token de administrador en la variable `MYSTORE_TOKEN` (obligatoria: el servidor no arranca sin ella).
- Identidad de una app = package name + tipo. `id = <package>-<phone|watch>` (un package Java nunca contiene `-`).
- Tipo `watch` si el manifest declara `<uses-feature android:name="android.hardware.type.watch">`; si no, `phone`.
- Subir, editar y borrar exigen `Authorization: Bearer <token>`. Listar, icono y descarga son públicos.
- Rechazar subida si: la APK no es válida (400), `versionCode` ≤ guardado (409), certificado distinto al guardado (409). El JSON de error `{"error": "..."}` explica el motivo en español.
- Solo se guarda la última versión de cada app.
- Variables: `MYSTORE_TOKEN`, `MYSTORE_DATA` (defecto `/data`), `MYSTORE_ADDR` (defecto `:8080`), `MYSTORE_MAX_UPLOAD_MB` (defecto `300`).
- Módulo Go: `github.com/adriarroyo08/mystore/server`. Licencia MIT. Textos visibles en español.
- SDK Android: `%LOCALAPPDATA%\Android\Sdk`, build-tools `36.0.0`, plataforma `android-36`. `adb` está en `platform-tools` (no en el PATH).

## Review Focus

1. Fichero que no es una APK o APK corrupta → 400 con mensaje claro y nada guardado (test en Task 3 y Task 5).
2. Subida que falla a mitad o dos subidas a la vez de la misma app → la versión anterior queda intacta; se escribe en temporal y se renombra (test en Task 4).
3. Token ausente o incorrecto → 401 y sin cambios; comparación en tiempo constante (test en Task 5).
4. `id` manipulado en la URL (`..%2F`, barras, mayúsculas raras) → 404, nunca se toca nada fuera de `DATA/apps` (test en Task 4).
5. APK cuyo icono no se puede extraer (icono adaptativo XML o ausente) → la subida funciona y se sirve un icono genérico (test en Task 3 y Task 5).

---

### Task 1: Herramientas, módulo Go y APKs de prueba

**Files:**
- Create: `server/go.mod`, `server/testdata/make-fixtures.sh`, `server/testdata/src/` (manifests + icono), `server/testdata/*.apk` (generadas y commiteadas), `server/testdata/keys/` (keystores de prueba, commiteados: son solo para tests)

**Interfaces:**
- Produces: fixtures en `server/testdata/`:
  - `phone-v1.apk`: package `com.example.demo`, versionCode 1, versionName `1.0`, label `Demo`, icono PNG en `mipmap`, firmada con `keys/a.jks`
  - `phone-v2.apk`: igual con versionCode 2, versionName `2.0`
  - `phone-v2-otherkey.apk`: como v2 firmada con `keys/b.jks`
  - `watch-v1.apk`: mismo package `com.example.demo`, versionCode 1, con `uses-feature android.hardware.type.watch` y `uses-library com.google.android.wearable required=false`, firmada con `a.jks`
  - `noicon-v1.apk`: package `com.example.noicon`, versionCode 1, sin `android:icon`
  - `not-an-apk.apk`: 1 KB de texto
  - Todas con `android:hasCode="false"` y `minSdkVersion` 30 (instalables, sin dex).

- [ ] **Step 1: Instalar Go**

Run: `winget install --id GoLang.Go -e --accept-source-agreements --accept-package-agreements` y abrir una shell nueva.
Expected: `go version` muestra go1.2x.

- [ ] **Step 2: `go mod init github.com/adriarroyo08/mystore/server` en `server/`**

- [ ] **Step 3: Escribir `server/testdata/make-fixtures.sh`**

Usa `aapt2 compile` (icono) + `aapt2 link -I android.jar --manifest ... --min-sdk-version 30 --version-code N --version-name X` + `zipalign -p 4` + `apksigner sign --ks keys/X.jks`. Si no existen, crea `keys/a.jks` y `keys/b.jks` con `keytool -genkeypair` (contraseña `android`, alias `key`). Rutas del SDK desde `$LOCALAPPDATA/Android/Sdk`.

- [ ] **Step 4: Ejecutar y verificar**

Run: `bash server/testdata/make-fixtures.sh && "$LOCALAPPDATA/Android/Sdk/build-tools/36.0.0/aapt2" dump badging server/testdata/watch-v1.apk`
Expected: `package: name='com.example.demo' versionCode='1'` y `uses-feature: name='android.hardware.type.watch'`. `apksigner verify --print-certs` muestra certificados distintos para `phone-v2.apk` y `phone-v2-otherkey.apk`.

- [ ] **Step 5: Commit**

```bash
git add server && git commit -m "chore: go module and APK test fixtures"
```

---

### Task 2: Prueba desechable de instalación en el TicWatch (PUERTA)

**Files:**
- Create: `spikes/wear-install/` (proyecto Android Studio, plantilla "Wear OS → Empty Wear App", Kotlin, minSdk 30). Desechable: no se reutiliza en la Fase 2.

**Interfaces:**
- Consumes: `server/testdata/watch-v1.apk` (Task 1).

Sin tests: es una prueba manual en el reloj real.

- [ ] **Step 1: App mínima con un botón "Instalar"** que:
  1. Si `packageManager.canRequestPackageInstalls()` es false, abre `Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES` con `package:<propio>`.
  2. Descarga por HTTP la URL fija `http://<IP-del-PC>:8000/watch-v1.apk` (en `BuildConfig`/constante).
  3. Instala con `PackageInstaller` (sesión `MODE_FULL_INSTALL`, `commit` con `PendingIntent` a un `BroadcastReceiver`; en `STATUS_PENDING_USER_ACTION` lanza `EXTRA_INTENT`) y muestra el estado final en pantalla.
  Manifest: `INTERNET`, `REQUEST_INSTALL_PACKAGES`, `usesCleartextTraffic="true"`.

- [ ] **Step 2: Servir la fixture desde el PC**

Run: `cd server/testdata && python -m http.server 8000`

- [ ] **Step 3: Instalar la prueba en el reloj por depuración inalámbrica**

En el reloj: Ajustes → Opciones de desarrollador → Depuración inalámbrica. Run: `adb pair <ip:puerto>`, `adb connect <ip:puerto>`, `adb install app-debug.apk`.

- [ ] **Step 4: Pulsar "Instalar" en el reloj** (con la ayuda del usuario)

Expected: aparece el diálogo del sistema, se confirma y `adb shell pm list packages com.example.demo` muestra el paquete.
**Si falla** (no existe la pantalla de permisos, el instalador rechaza la sesión, etc.): anotar el error exacto en `spikes/wear-install/RESULTADO.md`, PARAR y hablar con el usuario antes de seguir con la app del reloj. Las Tasks 3–7 no dependen de esto y pueden continuar.

- [ ] **Step 5: Anotar resultado y commit**

`spikes/wear-install/RESULTADO.md`: funciona sí/no, versión de Wear OS del reloj, pasos que pidió el sistema.

```bash
git add spikes && git commit -m "spike: install APK from a Wear OS app on TicWatch"
```

---

### Task 3: Leer metadatos de una APK

**Files:**
- Create: `server/apk.go`, `server/apk_test.go`

**Interfaces:**
- Consumes: fixtures (Task 1).
- Produces:
```go
type Meta struct {
    Package     string `json:"package"`
    Type        string `json:"type"`        // "phone" | "watch"
    Label       string `json:"label"`
    VersionCode int64  `json:"versionCode"`
    VersionName string `json:"versionName"`
    MinSDK      int    `json:"minSdk"`
    CertSHA256  string `json:"certSha256"`  // hex del primer certificado de firma
    Size        int64  `json:"size"`
}
func ParseAPK(path string) (Meta, []byte /*icon PNG, nil si no hay*/, error)
var ErrInvalidAPK = errors.New("el fichero no es una APK válida")
```

- [ ] **Step 1: Tests que fallan**

```go
func TestParsePhone(t *testing.T) {
    m, icon, err := ParseAPK("testdata/phone-v1.apk")
    // err == nil; m.Package == "com.example.demo"; m.Type == "phone"; m.Label == "Demo"
    // m.VersionCode == 1; m.VersionName == "1.0"; m.MinSDK == 30; len(m.CertSHA256) == 64
    // icon empieza por "\x89PNG"; m.Size == tamaño del fichero
}
func TestParseWatch(t *testing.T)        // watch-v1.apk → Type == "watch"
func TestCertDiffers(t *testing.T)       // phone-v2 y phone-v2-otherkey → CertSHA256 distintos; phone-v1 y phone-v2 iguales
func TestNoIcon(t *testing.T)            // noicon-v1.apk → err == nil, icon == nil
func TestInvalid(t *testing.T)           // not-an-apk.apk → errors.Is(err, ErrInvalidAPK)
```

- [ ] **Step 2: Run** `cd server && go test ./...` → FAIL (ParseAPK no definido).

- [ ] **Step 3: Implementar `ParseAPK`**

`androidbinary/apk.OpenFile` para manifest, label e `Icon(nil)` (codificar a PNG; si el icono es XML adaptativo o da error, devolver `nil` sin fallar). `apkverifier.ExtractCerts` para el certificado; SHA-256 del `Raw` del primer certificado. Cualquier fallo al abrir/leer manifest o firma → envolver con `ErrInvalidAPK`.

- [ ] **Step 4: Run** `go test ./...` → PASS.

- [ ] **Step 5: Commit** `git commit -am "feat(server): parse APK metadata"` (añadiendo los ficheros nuevos y `go.sum`).

---

### Task 4: Almacén en disco

**Files:**
- Create: `server/store.go`, `server/store_test.go`

**Interfaces:**
- Consumes: `Meta`, `ParseAPK` (Task 3).
- Produces:
```go
type App struct {
    Meta
    ID          string    `json:"id"`
    Description string    `json:"description"`
    Notes       string    `json:"notes"`
    HasIcon     bool      `json:"hasIcon"`
    UpdatedAt   time.Time `json:"updatedAt"`
}
type Store struct{ /* dir, sync.Mutex */ }
func NewStore(dir string) (*Store, error)          // crea dir/apps
func (s *Store) Put(tmpAPK string, notes, description string) (App, error) // mueve tmpAPK dentro del store
func (s *Store) List(typ string) ([]App, error)    // typ "" = todas; orden por Label
func (s *Store) Get(id string) (App, error)
func (s *Store) Update(id string, description, notes *string) (App, error)
func (s *Store) Delete(id string) error
func (s *Store) APKPath(id string) (string, error)
func (s *Store) IconPath(id string) (string, bool, error)
var ErrNotFound    = errors.New("app no encontrada")
var ErrNotNewer    = errors.New("ya hay una versión igual o más nueva de esta app")
var ErrCertMismatch = errors.New("la APK está firmada con otra clave; borra la app si quieres cambiar de firma")
```
Disco: `dir/apps/<id>/{app.apk, icon.png, meta.json}`. `meta.json` = `App` serializado. `id` válido solo si casa `^[a-zA-Z][a-zA-Z0-9_.]*-(phone|watch)$`; si no, `ErrNotFound`.

- [ ] **Step 1: Tests que fallan** (cada test copia fixtures a `t.TempDir()` porque `Put` mueve el fichero)

```go
func TestPutAndList(t *testing.T)      // Put phone-v1 → App.ID == "com.example.demo-phone"; List("phone") len 1; List("watch") len 0
func TestPhoneAndWatchCoexist(t *testing.T) // Put phone-v1 y watch-v1 → List("") len 2, IDs distintos
func TestRejectNotNewer(t *testing.T)  // Put phone-v2 y luego phone-v1 → ErrNotNewer; Get sigue con VersionCode 2
func TestRejectOtherKey(t *testing.T)  // Put phone-v1 y luego phone-v2-otherkey → ErrCertMismatch; sigue v1
func TestUpgradeKeepsDescription(t *testing.T) // Put v1 con description "d"; Put v2 con description "" → Description == "d"; Notes == las nuevas
func TestDeleteAllowsNewKey(t *testing.T) // Put v1, Delete, Put v2-otherkey → ok
func TestBadIDs(t *testing.T)          // Get/Delete/APKPath con "../x-phone", "a/b-phone", "com.example.demo", "" → ErrNotFound
func TestFailedPutKeepsOld(t *testing.T) // Put v1; Put not-an-apk → ErrInvalidAPK; APKPath sigue existiendo y ParseAPK de él da VersionCode 1
func TestConcurrentPuts(t *testing.T)  // 2 goroutines Put phone-v2 a la vez → una ok y otra ErrNotNewer; meta.json válido con VersionCode 2
```

- [ ] **Step 2: Run** `go test ./...` → FAIL.

- [ ] **Step 3: Implementar.** `Put`: `ParseAPK` fuera del lock; con el lock, comparar con lo guardado, escribir en `dir/apps/.tmp-<random>/` y `os.Rename` del directorio entero sobre el destino (borrando el antiguo justo antes). Al arrancar, `NewStore` borra restos `.tmp-*`.

- [ ] **Step 4: Run** `go test ./...` → PASS.

- [ ] **Step 5: Commit** `git commit -m "feat(server): disk store with version and signature checks"`

---

### Task 5: API HTTP

**Files:**
- Create: `server/api.go`, `server/api_test.go`, `server/main.go`, `server/web/generic-icon.png`

**Interfaces:**
- Consumes: `Store` (Task 4).
- Produces:
```go
func NewHandler(s *Store, token string, maxUploadBytes int64) http.Handler
```
Rutas:
| Método y ruta | Auth | Respuesta |
|---|---|---|
| `GET /api/apps?type=phone\|watch` | no | `200 [App + iconUrl + apkUrl]` (`iconUrl` = `/api/apps/{id}/icon`, `apkUrl` = `/api/apps/{id}/apk`) |
| `GET /api/apps/{id}/icon` | no | PNG; icono genérico embebido si `!HasIcon` |
| `GET /api/apps/{id}/apk` | no | `application/vnd.android.package-archive`, `Content-Disposition: attachment; filename="<package>-<versionName>.apk"`, `http.ServeContent` (permite reanudar) |
| `POST /api/apps` multipart `apk`, `notes?`, `description?` | sí | `201 App` / `400` / `409` / `413` |
| `PATCH /api/apps/{id}` JSON `{description?, notes?}` | sí | `200 App` |
| `DELETE /api/apps/{id}` | sí | `204` |
Errores siempre `{"error": "<mensaje en español>"}`. Token con `subtle.ConstantTimeCompare`. `main.go` lee las variables de Global Constraints, sale con error si falta `MYSTORE_TOKEN`, y registra un log por petición de escritura.

- [ ] **Step 1: Tests que fallan** (`httptest`, store en `t.TempDir()`, token `"secret"`)

```go
func TestUploadRequiresToken(t *testing.T) // POST sin header y con "Bearer wrong" → 401; List vacío
func TestUploadAndList(t *testing.T)       // POST phone-v1 con token → 201; GET /api/apps?type=phone → 1 app con apkUrl "/api/apps/com.example.demo-phone/apk"
func TestUploadErrors(t *testing.T)        // not-an-apk → 400; phone-v1 dos veces → 409 y body.error no vacío; v2-otherkey tras v1 → 409
func TestUploadTooBig(t *testing.T)        // NewHandler con maxUploadBytes=1024 y phone-v1 → 413
func TestDownloadAPK(t *testing.T)         // GET apk → 200, Content-Type correcto, bytes == fixture
func TestGenericIcon(t *testing.T)         // subir noicon-v1 → GET icon → 200, empieza por "\x89PNG"
func TestPatchAndDelete(t *testing.T)      // PATCH description "hola" → 200 y GET lista la muestra; DELETE sin token 401; con token 204; GET apk → 404
func TestTraversal(t *testing.T)           // GET /api/apps/..%2F..%2Fetc-phone/apk → 404
```

- [ ] **Step 2: Run** `go test ./...` → FAIL.

- [ ] **Step 3: Implementar** con `http.NewServeMux` y patrones `"GET /api/apps/{id}/apk"` etc.; `http.MaxBytesReader` para el límite; el multipart se guarda en un temporal dentro de `MYSTORE_DATA` (mismo disco, para que `Rename` funcione).

- [ ] **Step 4: Run** `go test ./...` → PASS.

- [ ] **Step 5: Probar a mano**

Run: `cd server && MYSTORE_TOKEN=secret MYSTORE_DATA=./tmpdata go run .` y en otra shell `curl -H "Authorization: Bearer secret" -F apk=@testdata/phone-v1.apk localhost:8080/api/apps` → JSON con `"versionCode":1`. Añadir `server/tmpdata/` al `.gitignore`.

- [ ] **Step 6: Commit** `git commit -m "feat(server): HTTP API"`

---

### Task 6: Web de administración

**Files:**
- Create: `server/web/index.html` (HTML + CSS + JS en un solo fichero, sin frameworks)
- Modify: `server/api.go` (servir `GET /` con `go:embed`)
- Test: `server/api_test.go`

**Interfaces:**
- Consumes: la API de la Task 5.

Página única en español: campo para el token (se guarda en `localStorage`), zona de arrastrar y soltar (o elegir fichero) para una o varias APKs con barra de progreso (`XMLHttpRequest.upload.onprogress`), campo opcional de notas, y lista de apps en el estilo "lista" del boceto (icono, nombre, versión, tipo móvil/reloj, tamaño, fecha) con botones Editar descripción/notas y Borrar (confirmación en la propia página, sin `confirm()`). Los errores de la API se muestran tal cual llegan. Enlace de descarga directa de cada APK (sirve para instalar la tienda en el móvil la primera vez).

- [ ] **Step 1: Test que falla**

```go
func TestServesAdminPage(t *testing.T) // GET / → 200, Content-Type text/html, body contiene "MyStore"
```

- [ ] **Step 2: Run** → FAIL. **Step 3: Implementar.** **Step 4: Run** → PASS.

- [ ] **Step 5: Probar a mano en el navegador** con el servidor de la Task 5: subir `phone-v1.apk` y `watch-v1.apk`, ver las dos filas, subir `phone-v1.apk` otra vez y ver el error 409 en español, borrar una.

- [ ] **Step 6: Commit** `git commit -m "feat(server): admin web page"`

---

### Task 7: Docker, GitHub Action, CI y documentación

**Files:**
- Create: `server/Dockerfile`, `docker-compose.yml`, `action.yml`, `.github/workflows/ci.yml`, `README.md`, `LICENSE` (MIT, "Adrian", 2026)

**Interfaces:**
- Consumes: servidor completo (Tasks 3–6).
- Produces: Action usable como `uses: adriarroyo08/mystore@v1` con inputs `store-url` (req), `token` (req), `apk` (req, ruta o glob con un solo resultado), `notes` (opcional). Falla el job si la respuesta no es 2xx e imprime `error` del JSON.

- [ ] **Step 1: `server/Dockerfile`** multi-stage: `golang:<versión>-alpine` compila con `CGO_ENABLED=0`; final `gcr.io/distroless/static`; `VOLUME /data`; `EXPOSE 8080`.

- [ ] **Step 2: `docker-compose.yml`** de ejemplo: servicio `mystore`, build `./server`, `MYSTORE_TOKEN: ${MYSTORE_TOKEN}`, volumen `./data:/data`, puerto `8080:8080`.

- [ ] **Step 3: `action.yml`** compuesto (`runs.using: composite`) con un paso `bash`: `curl --fail-with-body -H "Authorization: Bearer $TOKEN" -F apk=@"$APK" -F notes="$NOTES" "$STORE_URL/api/apps"`; inputs pasados por `env:` (nunca interpolados en el script).

- [ ] **Step 4: `.github/workflows/ci.yml`**: en push/PR, job 1 `go test ./...` en `server/`; job 2 construye la imagen, la arranca con `MYSTORE_TOKEN=ci`, usa la Action local (`uses: ./`) para subir `server/testdata/phone-v1.apk` y comprueba con `curl` que `/api/apps` contiene `com.example.demo`.

- [ ] **Step 5: Verificar en local**

Run: `docker compose up -d --build` con `MYSTORE_TOKEN=secret` y repetir el `curl` de la Task 5.
Expected: 201 y la app en `http://localhost:8080/`.

- [ ] **Step 6: `README.md`** en español: qué es, desplegar con Docker (incl. poner HTTPS delante con un proxy inverso), subir desde la web, ejemplo de workflow con la Action (release → compilar → subir), primera instalación de la tienda (móvil: enlace de descarga; reloj: `adb pair/connect/install` una vez) y una nota de que las apps móvil/reloj llegan en la Fase 2.

- [ ] **Step 7: Commit** `git commit -m "feat: docker image, GitHub Action, CI and README"`
