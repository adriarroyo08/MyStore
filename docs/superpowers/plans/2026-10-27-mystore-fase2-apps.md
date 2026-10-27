# MyStore Fase 2 (iconos adaptativos + apps móvil y reloj) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Que el servidor muestre los iconos reales de las apps (adaptativos) y construir las apps tienda para móvil y reloj que listan, instalan y actualizan APKs desde la tienda.

**Architecture:** Servidor: al subir, si el icono es un `<adaptive-icon>`, se componen fondo + primer plano (color, PNG, WebP o `<vector>`) en un PNG de 192 px con `golang.org/x/image/vector`. Android: un proyecto Gradle en `android/` con `:core` (modelos, cliente HTTP con `HttpURLConnection` + `org.json`, lógica de estado, descarga e instalación con `PackageInstaller`), `:mobile` (Compose Material 3) y `:wear` (Compose for Wear OS). Sin librerías de red ni de imágenes.

**Tech Stack:** Go + `golang.org/x/image` (vector, webp). Android: AGP 9.4.1, Kotlin 2.4.21 (Kotlin integrado de AGP 9), Compose BOM 2026.09.00, `androidx.wear.compose` 1.4.0, coroutines 1.11.0, Gradle 9.8.2 (wrapper copiado de `spikes/wear-install`).

**Spec:** `docs/superpowers/specs/2026-10-27-mystore-design.md` · Resultado de la prueba del reloj: `spikes/wear-install/RESULTADO.md`

## Global Constraints

- applicationId de las dos apps tienda: `dev.mystore` (mismo package; el servidor las guarda como `dev.mystore-phone` y `dev.mystore-watch`). minSdk 30, compileSdk 36, targetSdk 35.
- Botón por app: **Instalar** (no instalada) / **Actualizar** (instalada con versionCode menor) / **Abrir** (instalada, igual o mayor).
- Móvil lista `type=phone`; reloj lista `type=watch`. Textos en español.
- La primera vez la app pide la URL de la tienda; se guarda en `SharedPreferences`. Se aceptan `http://` y `https://` (tiendas en red local).
- Instalación con `PackageInstaller` (sesión `MODE_FULL_INSTALL`, `STATUS_PENDING_USER_ACTION` → lanzar `EXTRA_INTENT`), igual que la prueba del reloj. Si `canRequestPackageInstalls()` es false, abrir `ACTION_MANAGE_UNKNOWN_APP_SOURCES`.
- Permisos: `INTERNET`, `REQUEST_INSTALL_PACKAGES`, `QUERY_ALL_PACKAGES` (hay que ver cualquier paquete instalado), `usesCleartextTraffic="true"`.
- Sin notificaciones en segundo plano ni historial de versiones (no objetivos del spec).
- Las apps tienda se actualizan desde la propia tienda: se suben a ella como cualquier otra app.

## Review Focus

1. Tienda caída, URL mal escrita o sin red → mensaje claro con botón Reintentar, sin cerrarse (test de `StoreClient` en Task 2, prueba manual en Tasks 4–5).
2. Descarga cortada a mitad → no se lanza el instalador con una APK incompleta; se muestra error y se puede reintentar (comprobación de tamaño en Task 3).
3. El usuario cancela el diálogo del sistema o niega el permiso → el botón vuelve a su estado, sin quedarse en "Instalando…" (Task 3, manual en Tasks 4–5).
4. Vector con elementos no soportados (gradientes, `clip-path`, colores por referencia irresoluble) → se dibuja lo que se pueda o se usa el icono genérico; nunca falla la subida (test en Task 1).
5. Actualizar la propia app tienda → el sistema la reinicia; al volver, la lista refleja la versión nueva (manual en Task 6).

---

### Task 1: Iconos adaptativos en el servidor

**Files:**
- Create: `server/icon.go`, `server/icon_test.go`
- Modify: `server/apk.go` (usar el renderizado cuando el icono no es PNG), `server/testdata/make-fixtures.sh`, `server/testdata/src/` (nuevo manifest y recursos)
- Create (generadas): `server/testdata/adaptive-v1.apk`, `server/testdata/badvector-v1.apk`

**Interfaces:**
- Produces:
```go
// renderIcon devuelve un PNG cuadrado de iconSize px para el recurso de icono indicado
// (ruta dentro de la APK), o nil si no sabe dibujarlo.
func renderIcon(zr *apkparser.ZipReader, res *apkparser.ResourceTable, path string) []byte
const iconSize = 192
```
- Fixtures: `adaptive-v1.apk` (package `com.example.adaptive`, `mipmap-anydpi-v26/ic_launcher.xml` = `<adaptive-icon>` con `background` = `@color/bg` `#FF3DDC84` y `foreground` = `@drawable/fg`, un `<vector>` 108×108 con un `<group translateX="24" translateY="24">` y un `<path fillColor="#FFFFFFFF" pathData="M0,0h60v60h-60z"/>`). `badvector-v1.apk`: igual pero `fg` solo contiene un path con `fillColor` apuntando a un gradiente (`@drawable/grad`, un `<gradient>`).

Comportamiento: PNG/WebP → decodificar y escalar a 192; `<vector>` → rasterizar paths con relleno (`M m L l H h V v C c S s Q q T t A a Z z`; `<group>` con translate/scale/rotate/pivot; `fillColor` `#RGB/#ARGB/#RRGGBB/#AARRGGBB` y `fillAlpha`; `strokeColor`/gradientes se ignoran); `<adaptive-icon>` → lienzo de 108 dp, pintar fondo (color o drawable) y primer plano, recortar el 72 % central (zona visible de la máscara) y escalar a 192. Cualquier error → nil (el servidor usa el genérico).

- [ ] **Step 1: Fixtures** — añadir recursos y `build adaptive-v1.apk adaptive.xml 1 1.0 a res-adaptive` (y `badvector`) al script, regenerar y comprobar con `aapt2 dump badging` que `application-icon` apunta a `res/mipmap-anydpi-v26/...xml`.
- [ ] **Step 2: Tests que fallan**
```go
func TestAdaptiveIcon(t *testing.T)   // ParseAPK("testdata/adaptive-v1.apk") → icon PNG 192×192;
                                     // píxel (2,2) ≈ #3DDC84 (fondo); píxel central (96,96) ≈ blanco (primer plano)
func TestBadVectorIcon(t *testing.T) // badvector-v1.apk → err == nil; icon es PNG con fondo #3DDC84 o nil, nunca error
func TestPathParser(t *testing.T)    // parsePath("M10,10 l5,0 a5,5 0 0 1 0,10 z") sin error y con 4+ segmentos; "M10" → error
```
- [ ] **Step 3: Run** `cd server && go test ./...` → FAIL.
- [ ] **Step 4: Implementar** `icon.go` (parsear XML del recurso con `apkparser.ParseXml` y la `ResourceTable` de la APK) y llamarlo desde `ParseAPK` cuando el icono no es PNG.
- [ ] **Step 5: Run** `go test ./...` → PASS. **Comprobación real:** subir `workstation/android/phone/build/outputs/apk/debug/phone-debug.apk` a un servidor local y ver el icono en la web (captura).
- [ ] **Step 6: Commit** `feat(server): render adaptive and vector launcher icons`

---

### Task 2: Proyecto Android y módulo `:core` (lógica pura)

**Files:**
- Create: `android/settings.gradle.kts`, `android/build.gradle.kts`, `android/gradle.properties`, `android/gradlew*`, `android/gradle/wrapper/*`, `android/.gitignore`, `android/core/build.gradle.kts`, `android/core/src/main/AndroidManifest.xml`
- Create: `android/core/src/main/java/dev/mystore/core/Models.kt`, `StoreClient.kt`
- Test: `android/core/src/test/java/dev/mystore/core/ModelsTest.kt`, `StoreClientTest.kt`

**Interfaces:**
- Produces:
```kotlin
data class StoreApp(val id: String, val pkg: String, val type: String, val label: String,
    val versionCode: Long, val versionName: String, val size: Long, val description: String,
    val notes: String, val iconUrl: String, val apkUrl: String)   // URLs ya absolutas
enum class AppAction { INSTALL, UPDATE, OPEN }
fun actionFor(storeVersion: Long, installedVersion: Long?): AppAction
fun normalizeStoreUrl(input: String): String?   // añade https:// si falta, quita "/" final; null si no es http(s) válida
fun parseApps(json: String, baseUrl: String): List<StoreApp>
fun List<StoreApp>.search(q: String): List<StoreApp> // por label o pkg, sin mayúsculas ni tildes
class StoreClient(val baseUrl: String) {
    suspend fun apps(type: String): List<StoreApp>           // GET /api/apps?type=
    suspend fun icon(app: StoreApp): ByteArray
}
class StoreException(message: String) : Exception(message) // mensaje en español para la UI
```
- [ ] **Step 1: Esqueleto Gradle** (plugins como `spikes/wear-install`, más `com.android.library` y `org.jetbrains.kotlin.plugin.compose`), `local.properties` con `sdk.dir=C:/Users/adria/AppData/Local/Android/Sdk` (ignorado por git). `testImplementation("junit:junit:4.13.2")`, `testImplementation("org.json:json:20240303")` (org.json real en la JVM).
- [ ] **Step 2: Tests que fallan**
```kotlin
@Test fun actionForStates()        // (5, null)→INSTALL; (5, 4)→UPDATE; (5, 5)→OPEN; (5, 6)→OPEN
@Test fun normalizesUrls()         // "tienda.com/"→"https://tienda.com"; "http://192.168.1.2:8080"→igual; "ftp://x"→null; ""→null
@Test fun parsesApps()             // JSON de ejemplo con iconUrl "/api/apps/a-phone/icon" y base "http://h:8080" → iconUrl "http://h:8080/api/apps/a-phone/icon"
@Test fun searchIgnoresCaseAndAccents() // label "Cámara" encontrada con "camara"
@Test fun clientReportsUnreachableStore() // StoreClient("http://127.0.0.1:1").apps("phone") lanza StoreException con mensaje no vacío
@Test fun clientReportsHttpError()  // servidor com.sun.net.httpserver devolviendo 500 {"error":"x"} → StoreException("x")
@Test fun clientParsesList()        // mismo servidor devolviendo una lista → 1 StoreApp
```
- [ ] **Step 3: Run** `cd android && ./gradlew :core:testDebugUnitTest` → FAIL (compilación).
- [ ] **Step 4: Implementar** (`withContext(Dispatchers.IO)`, timeouts de conexión 10 s / lectura 30 s).
- [ ] **Step 5: Run** → PASS.
- [ ] **Step 6: Commit** `feat(android): core models and store client`

---

### Task 3: Descarga, instalación y estado instalado (`:core`)

**Files:**
- Create: `android/core/src/main/java/dev/mystore/core/Installer.kt`, `InstallReceiver.kt`, `Settings.kt`
- Modify: `android/core/src/main/AndroidManifest.xml` (permisos de Global Constraints y el receiver)

**Interfaces:**
- Produces:
```kotlin
sealed interface InstallState { data object Idle; data class Downloading(val progress: Float); data object Installing; data class Failed(val message: String) }
class Installer(private val context: Context) {
    val states: StateFlow<Map<String, InstallState>>       // por app id
    fun installedVersion(pkg: String): Long?
    fun canInstall(): Boolean                              // canRequestPackageInstalls()
    fun permissionIntent(): Intent                         // ACTION_MANAGE_UNKNOWN_APP_SOURCES
    suspend fun install(app: StoreApp)                     // descarga a cacheDir, comprueba tamaño == app.size, crea sesión
    fun launch(pkg: String)                                // getLaunchIntentForPackage
    fun refresh()                                          // emite de nuevo para recalcular botones (al volver a primer plano)
}
class StoreSettings(context: Context) { var storeUrl: String? }
```
El `InstallReceiver` traduce el resultado a `Idle` (éxito o cancelado por el usuario) o `Failed(mensaje)` y lanza `EXTRA_INTENT` en `STATUS_PENDING_USER_ACTION`.

Sin tests JVM (todo es API de Android); se verifica en los dispositivos en las Tasks 4–5.

- [ ] **Step 1: Implementar.** Descarga con `HttpURLConnection` a `cacheDir/<id>.apk` informando progreso; si los bytes ≠ `app.size` → `Failed("La descarga se cortó; inténtalo de nuevo")` y no se crea la sesión.
- [ ] **Step 2: Run** `./gradlew :core:assembleDebug :core:testDebugUnitTest` → BUILD SUCCESSFUL.
- [ ] **Step 3: Commit** `feat(android): download and install via PackageInstaller`

---

### Task 4: App del móvil

**Files:**
- Create: `android/mobile/build.gradle.kts`, `android/mobile/src/main/AndroidManifest.xml`, `android/mobile/src/main/java/dev/mystore/mobile/MainActivity.kt`, `StoreScreen.kt`, `res/` (icono adaptativo propio, nombre "MyStore")

**Interfaces:**
- Consumes: `StoreClient`, `actionFor`, `search`, `Installer`, `StoreSettings`, `normalizeStoreUrl` (Tasks 2–3).

Pantallas (estado en un `ViewModel`, navegación por estado, sin librería de navegación):
1. **Configurar:** campo URL + botón Conectar (prueba `apps("phone")` antes de guardar; muestra el error si falla).
2. **Lista** (boceto A): barra superior "MyStore" con icono de ajustes, buscador, filas icono 48 dp redondeado + nombre + "v1.4.0 · 8 MB" + botón Instalar/Actualizar/Abrir; durante la descarga, barra de progreso en la fila. Tirar para recargar. Error de red → mensaje + Reintentar. Lista vacía → "Tu tienda aún no tiene apps para el móvil".
3. **Ficha** al tocar una fila: icono grande, nombre, versión, tamaño, descripción, "Novedades" (notas) y el mismo botón.
4. **Ajustes:** cambiar URL.
Al volver a primer plano (`onResume`), recargar estado instalado.

- [ ] **Step 1: Implementar.**
- [ ] **Step 2: Build** `./gradlew :mobile:assembleDebug` → BUILD SUCCESSFUL.
- [ ] **Step 3: Probar en el emulador** `Medium_Phone_API_36.1` contra un servidor local (`go run` con `MYSTORE_ADDR=0.0.0.0:18080`, el emulador lo ve en `http://10.0.2.2:18080`) con `phone-v1.apk`, `noicon-v1.apk` y `workstation` subidas: configurar, ver lista con iconos, instalar (aceptar permiso y diálogo), botón pasa a Abrir; subir `phone-v2.apk` → Actualizar → instalar; cancelar un diálogo → botón vuelve; parar el servidor → error + Reintentar. Capturas en el ledger.
- [ ] **Step 4: Commit** `feat(android): mobile store app`

---

### Task 5: App del reloj

**Files:**
- Create: `android/wear/build.gradle.kts`, `android/wear/src/main/AndroidManifest.xml` (`uses-feature android.hardware.type.watch`, `com.google.android.wearable.standalone=true`), `android/wear/src/main/java/dev/mystore/wear/MainActivity.kt`, `WearScreens.kt`, `res/`

**Interfaces:**
- Consumes: lo mismo que la Task 4.

Pantallas (boceto del reloj):
1. **Configurar:** chip "Escribir dirección" que abre el teclado del sistema (`RemoteInput` vía `androidx.wear:wear-input`) prerrellenado con `https://`; prueba `apps("watch")` antes de guardar.
2. **Lista:** `ScalingLazyColumn` con título "MyStore", chips icono redondo + nombre + punto verde si `UPDATE`; desplazable con la corona. Error → texto + chip Reintentar. Vacía → "No hay apps para el reloj". Último chip: "Ajustes".
3. **Detalle:** icono, nombre, versión y botón grande Instalar/Actualizar/Abrir con progreso circular durante la descarga.

- [ ] **Step 1: Implementar.**
- [ ] **Step 2: Build** `./gradlew :wear:assembleDebug` → BUILD SUCCESSFUL.
- [ ] **Step 3: Probar en el TicWatch real** (skill `ticwatch-adb`): instalar con adb, servidor local en `http://192.168.18.3:18080` con `watch-v1.apk` subida; configurar URL, ver lista, instalar, Abrir. Desinstalar las apps de prueba al terminar. Capturas con `screencap`.
- [ ] **Step 4: Commit** `feat(android): wear store app`

---

### Task 6: Autoactualización, CI y documentación

**Files:**
- Modify: `android/mobile/build.gradle.kts`, `android/wear/build.gradle.kts` (firma: `android/keystore.properties` si existe, si no la debug), `.github/workflows/ci.yml` (job `android`: JDK 17, `./gradlew :core:testDebugUnitTest assembleDebug`), `README.md`, `.gitignore` (`android/keystore.properties`, `*.jks` fuera de testdata)

- [ ] **Step 1: Firma configurable** — mismo keystore para móvil y reloj; documentar que para autoactualizarse todas las builds deben firmarse con la misma clave.
- [ ] **Step 2: Probar autoactualización** en el emulador: subir la app tienda v1 (versionCode 1) a la tienda, instalarla, subir v2 (versionCode 2) → aparece en su propia lista con Actualizar → actualizar → la app se reinicia y muestra la v2.
- [ ] **Step 3: CI + README** (cómo compilar las apps, primera instalación, la URL en el reloj).
- [ ] **Step 4: Run** `./gradlew :core:testDebugUnitTest assembleDebug` y `cd server && go test ./...` → todo verde.
- [ ] **Step 5: Commit** `feat: store self-update, Android CI and docs`

---

## Ampliación: ficha tipo Play Store (pedida por el usuario a mitad de fase)

Contrato de API nuevo (lo usan las Tasks 7 y 8):
- `App` JSON gana `"screenshots": ["/api/apps/{id}/screenshots/{n}", ...]` (siempre presente, `[]` si no hay), en el orden de subida.
- `POST /api/apps/{id}/screenshots` (token, multipart campo `image`, PNG/JPEG/WebP ≤ 5 MB y ≤ 4096 px por lado) → `201` App. `409` si ya hay 8. `400` si no es imagen válida.
- `DELETE /api/apps/{id}/screenshots/{n}` (token) → `200` App.
- `GET /api/apps/{id}/screenshots/{n}` público, con el `Content-Type` real.
- Al subir una versión nueva de la app, las capturas se conservan; al borrar la app, se borran.

### Task 7: Capturas en servidor y web
**Files:** `server/store.go`, `server/api.go`, `server/web/index.html` y sus tests.
Tests: subir/listar/servir/borrar captura; límite de 8 → 409; fichero no imagen → 400; sin token → 401; capturas conservadas tras subir v2; `n` fuera de rango o manipulado → 404. Web: en "Editar", miniaturas de las capturas con botón quitar y zona para añadir.

### Task 8: Ficha tipo Play Store y navegación en las apps
**Files:** `android/core` (modelo `screenshots: List<String>` absolutas, `updatedAt`), `android/mobile`, `android/wear`.
Móvil: barra superior con flecha atrás en ficha y ajustes; `BackHandler` en todas las pantallas (visor → ficha → lista); ficha como en Global Constraints de la spec ampliada; visor de capturas a pantalla completa deslizable. Reloj: `SwipeToDismissBox` (o equivalente) para volver; ficha desplazable con descripción, capturas y novedades; `TimeText` sin solapar el título. Test JVM: `parseApps` lee `screenshots` y las hace absolutas.
