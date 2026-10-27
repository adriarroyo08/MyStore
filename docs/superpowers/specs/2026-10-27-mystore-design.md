# MyStore: diseño

Fecha: 2026-10-27

## Qué es y por qué

MyStore es una "Play Store" personal, open source y autoalojada. Cada persona la monta en su propio servidor y la usa para distribuir **sus propias APKs** a **su móvil y su reloj Wear OS**.

El problema que resuelve (palabras del usuario): le molesta tener que generar la APK y meterla a mano en el móvil o el reloj. Quiere subir las APKs desde el ordenador o desde GitHub a un sitio web y descargarlas desde el móvil o el reloj, con "una app como Play Store".

Dispositivos del autor: móvil Android y **TicWatch Pro 5 Enduro** (Wear OS). Servidor: su VPS de Oracle, donde ya aloja otros proyectos con Docker.

## Criterio de éxito

"Que sea relativamente parecido a Play Store, que salgan las apps disponibles y el botón para instalarlas." En concreto: abrir la app en el móvil o en el reloj, ver la lista de apps de mi tienda y poder instalar o actualizar cualquiera con un toque, sin cable ni adb.

## Objetivos

1. **Servidor** que guarda APKs y sirve la lista de apps (última versión de cada una).
2. **Web de administración** para subir APKs desde el ordenador (arrastrar y soltar).
3. **Subida desde GitHub**: una GitHub Action reutilizable que sube la APK a la tienda en cada release.
4. **App de tienda para móvil** (Android).
5. **App de tienda para reloj** (Wear OS).
6. Despliegue sencillo para terceros: imagen Docker + `docker-compose.yml` de ejemplo + README.

## No objetivos (v1)

- Servicio multiusuario o tienda pública: **cada persona monta su propia instancia** (decidido por el usuario). No hay cuentas ni registro.
- Pagos, reseñas y valoraciones.
- (Cambio del 2026-10-27, decidido por el usuario: las **capturas de pantalla** pasan a ser objetivo; ver "Ficha tipo Play Store".)
- Historial de versiones / volver a una versión anterior: el servidor solo ofrece la última versión de cada app.
- Avisos de actualización en segundo plano: las actualizaciones se ven al abrir la tienda.
- Firmar APKs o compilar código: la tienda recibe APKs ya firmadas.

## Anti-objetivos

- Que cualquiera pueda subir APKs a una tienda ajena: subir y borrar requiere autenticación siempre.
- Que haga falta adb o cable para instalar apps en el reloj: si eso no se puede evitar, el proyecto pierde su razón de ser para el reloj (ver "Riesgo principal").

## Decisiones tomadas

Decididas por el usuario:
- Autoalojado, una instancia por persona.
- Pantalla principal del móvil en **lista** (opción A del boceto): fila con icono, nombre, versión, tamaño y botón a la derecha.
- Reloj: lista vertical redonda, desplazable con la corona, punto verde si hay actualización; al tocar una app, botón grande de Instalar.

- **Primera instalación de la app tienda**: en el reloj se hace **una sola vez** con adb (depuración inalámbrica); en el móvil, descargando la APK desde la web de la propia tienda o desde GitHub Releases. Después, las apps tienda se actualizan desde la propia tienda (aparecen en su lista como una app más). El README documenta estos pasos.

Elegidas por Claude (el usuario delegó las tecnologías):
- **Servidor en Go**, un solo binario, distribuido como imagen Docker.
- **Sin base de datos**: las APKs se guardan en un volumen en disco; los metadatos (package name, versionCode, versionName, nombre, icono, minSdk, si es app de reloj) se extraen de la propia APK al subirla y se guardan junto a ella (por ejemplo, un JSON por app).
- **Detección de apps de reloj**: automática, por `<uses-feature android:name="android.hardware.type.watch">` en el manifest de la APK. El móvil muestra las apps que no son de reloj; el reloj solo las de reloj.
- **Identidad de una app = package name + tipo (móvil o reloj)**. Muchas apps Wear OS usan el mismo `applicationId` en la APK del móvil y en la del reloj; así se guardan como dos entradas distintas y la subida de una no pisa a la otra. Versiones y borrados van por esa pareja.
- **Firma**: el servidor rechaza una subida cuyo certificado de firma no coincida con el de la versión ya guardada para esa app (Android no actualizaría una app instalada con otra firma, p. ej. una build debug encima de una release). Para cambiar de firma a propósito hay que borrar la app y volver a subirla.
- **Autenticación**: un único token de administrador configurado por variable de entorno. Lo usan la web de subida y la GitHub Action. La lista de apps y la descarga de APKs son públicas para quien conozca la URL de la tienda (es una tienda personal; se asume que el dueño protege con HTTPS).
- **Apps cliente en Kotlin + Jetpack Compose** (Compose for Wear OS en el reloj), en un único proyecto Gradle con un módulo común (cliente HTTP, modelos, lógica de instalación) y dos módulos de app (`mobile`, `wear`).
- **Instalación** mediante el instalador del sistema (`PackageInstaller`) con el permiso `REQUEST_INSTALL_PACKAGES`; el usuario concede "instalar apps desconocidas" la primera vez.
- **Estado del botón**: comparar el `versionCode` del servidor con el instalado → Instalar / Actualizar / Abrir.
- **Configuración de la tienda en el cliente**: la primera vez la app pide la URL del servidor.
- **Licencia MIT.**
- Repositorio único (monorepo): `server/`, `android/`, `github-action/` (o la Action en la raíz del repo, a criterio del builder).

## Funcionalidad

### Servidor + web
- Subir APK (web o API con token). Si ya existe la misma app (package + tipo), la nueva versión sustituye a la anterior; rechazar si el `versionCode` no es mayor (evita bajar de versión por error), si la firma no coincide o si la APK no es válida. El motivo del rechazo se muestra claramente en la web y en la salida de la Action.
- Campos editables opcionales por app: descripción y notas de la versión (en la web; la Action puede enviar las notas de la release de GitHub).
- Borrar una app (con token).
- API pública de solo lectura: lista de apps con metadatos, icono y enlace de descarga.
- Web: lista de apps subidas + zona de arrastrar y soltar + login por token.

### GitHub Action
- Entrada: URL de la tienda, token (secreto del repo), ruta de la APK, notas opcionales.
- Uso típico: en un workflow que se dispara al publicar una release, después del paso que compila la APK.

### App móvil
- Lista con buscador; botón Instalar / Actualizar / Abrir por fila.
- Ficha de app al tocarla: icono, nombre, versión, tamaño, descripción, notas de la versión y el mismo botón.
- Descarga con progreso y luego el diálogo del sistema para instalar.
- Pantalla de ajustes para cambiar la URL de la tienda.

### App reloj
- Lista redonda solo con apps de reloj; punto verde si hay actualización.
- Detalle simplificado con botón grande Instalar / Actualizar / Abrir.
- Descarga por WiFi directamente desde el servidor (no a través del móvil).

## Riesgo principal y primer paso

No está confirmado que Wear OS (en el TicWatch Pro 5 Enduro) permita a una app instalar otras APKs mediante `PackageInstaller` sin adb. **El primer paso del plan es una prueba desechable**: una app mínima de Wear OS que descarga una APK de una URL fija y lanza la instalación en el reloj real del usuario. Si no funciona, se para y se vuelve a hablar con el usuario antes de construir la app de reloj (el resto del proyecto sigue siendo válido).

## Lo que queda al criterio del builder

- Diseño exacto de la API REST y del formato de almacenamiento en disco.
- Librería Go para leer el manifest y el icono de la APK (incluidos iconos adaptativos; si no se puede extraer el icono, usar uno genérico).
- Cómo se introduce la URL de la tienda en el reloj (escribirla en el reloj, código QR, o pasarla desde la app del móvil con la Data Layer API). Lo más simple que funcione en v1.
- Aspecto visual concreto, siguiendo el estilo de Material 3 y los bocetos.
- Tamaño máximo de subida y límites razonables.
- Pruebas: las mínimas que protejan la lectura de APKs, la comparación de versiones y la autenticación.

## Ficha tipo Play Store (añadido 2026-10-27)

Pedido por el usuario: "que sea navegable, que permita retroceder, que ponga descripción de la app, fotos, etc. como en Play Store".

- Capturas por app (hasta 8), subidas desde la web de administración; se conservan entre versiones.
- Móvil: ficha con cabecera (icono, nombre, versión, tamaño, botón), carrusel de capturas con visor a pantalla completa, "Acerca de esta app" (descripción), "Novedades" (notas) y fecha de actualización. Atrás con flecha y con el gesto del sistema en todas las pantallas.
- Reloj: ficha desplazable con descripción, capturas y novedades; atrás deslizando a la derecha.
