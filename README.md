# MyStore

Tu propia "Play Store" personal, open source y autoalojada. Sube tus APKs desde el ordenador o desde GitHub, y descárgalas e instálalas desde el móvil y el reloj Wear OS sin cables ni adb.

- **Servidor** (Go, un solo binario en Docker): guarda tus APKs y lee de cada una el nombre, la versión, el icono y si es una app de reloj.
- **Web de administración**: arrastra tus APKs y gestiona la lista.
- **GitHub Action**: sube la APK automáticamente en cada release.
- **Apps tienda para móvil y reloj**: llegan en la Fase 2.

Cada persona monta su propia tienda; no hay cuentas ni registro.

## Desplegar

```bash
git clone https://github.com/adriarroyo08/mystore.git
cd mystore
export MYSTORE_TOKEN="$(openssl rand -hex 32)"   # guárdalo: lo necesitas para subir APKs
docker compose up -d --build
```

La tienda queda en `http://tu-servidor:8080`. Pon delante un proxy inverso con HTTPS (Caddy, nginx, Traefik…): el token viaja en cada subida y no debe ir en claro.

| Variable | Defecto | Para qué |
|---|---|---|
| `MYSTORE_TOKEN` | (obligatoria) | Token de administrador para subir, editar y borrar |
| `MYSTORE_DATA` | `/data` | Carpeta donde se guardan las APKs |
| `MYSTORE_ADDR` | `:8080` | Dirección de escucha |
| `MYSTORE_MAX_UPLOAD_MB` | `300` | Tamaño máximo de una APK |

Cualquiera que conozca la URL puede ver la lista y descargar las APKs; solo quien tenga el token puede cambiarlas.

## Subir APKs

**Desde la web:** abre la tienda, pega el token y arrastra las APKs.

**Desde GitHub:** guarda la URL y el token como secretos del repositorio (`MYSTORE_URL`, `MYSTORE_TOKEN`) y añade el paso tras compilar:

```yaml
name: Release
on:
  release:
    types: [published]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-java@v4
        with: { distribution: temurin, java-version: 17 }
      - run: ./gradlew assembleRelease   # firma la APK con tu clave de release
      - uses: adriarroyo08/mystore@v1
        with:
          store-url: ${{ secrets.MYSTORE_URL }}
          token: ${{ secrets.MYSTORE_TOKEN }}
          apk: app/build/outputs/apk/release/*.apk
          notes: ${{ github.event.release.body }}
```

**Desde la terminal:**

```bash
curl -H "Authorization: Bearer $MYSTORE_TOKEN" -F apk=@app-release.apk -F notes="Arreglos" https://tienda.tudominio.com/api/apps
```

### Reglas al subir

- La versión del móvil y la del reloj de una misma app se guardan por separado (se detecta sola: las apps de reloj declaran `android.hardware.type.watch`).
- Solo se guarda la última versión. Se rechaza una APK con un `versionCode` igual o menor que el guardado.
- Se rechaza una APK firmada con otra clave que la guardada (por ejemplo, una build debug encima de una release), porque los dispositivos no podrían actualizarla. Para cambiar de firma, borra la app y súbela de nuevo.

## API

| Método y ruta | Token | Qué hace |
|---|---|---|
| `GET /api/apps?type=phone\|watch` | no | Lista de apps |
| `GET /api/apps/{id}/icon` | no | Icono PNG |
| `GET /api/apps/{id}/apk` | no | Descarga la APK |
| `POST /api/apps` (multipart `apk`, `notes`, `description`) | sí | Sube o actualiza |
| `PATCH /api/apps/{id}` (JSON `description`, `notes`) | sí | Edita textos |
| `DELETE /api/apps/{id}` | sí | Borra |

El `id` es `<package>-phone` o `<package>-watch`. Los errores devuelven `{"error": "..."}`.

## Primera instalación de la app tienda

Las apps tienda (Fase 2) se actualizan solas desde tu tienda, pero la primera vez hay que instalarlas:

- **Móvil:** abre la web de tu tienda en el móvil, pulsa *Descargar* en la app tienda y permite "instalar apps desconocidas".
- **Reloj:** el reloj no tiene navegador, así que se instala una vez con adb por WiFi. En el reloj: *Ajustes → Opciones de desarrollador → Depuración inalámbrica → Vincular dispositivo nuevo*. En el ordenador:

  ```bash
  adb pair <ip>:<puerto-de-vinculación>    # pide el código que muestra el reloj
  adb connect <ip>:<puerto>
  adb install mystore-wear.apk
  ```

## Desarrollo

```bash
cd server
go test ./...
MYSTORE_TOKEN=secret MYSTORE_DATA=./tmp go run .
```

Las APKs de prueba de `server/testdata/` se regeneran con `bash server/testdata/make-fixtures.sh` (necesita el SDK de Android y un JDK).

## Licencia

MIT
